package schema

import (
	"context"
	"fmt"
	"log"

	// "fmt"
	"strconv"
	"strings"

	klog "k8s.io/klog/v2"

	"github.com/SherinV/search-api/graph/model"
	db "github.com/SherinV/search-api/pkg/database"

	// "github.com/jackc/pgx/v4"
	"github.com/lib/pq"
)

var trimAND string = " AND "

func Search(ctx context.Context, input []*model.SearchInput) ([]*model.SearchResult, error) {
	limit := 0
	srchResult := make([]*model.SearchResult, 0)

	if len(input) > 0 {
		for _, in := range input {
			query, args := searchQuery(ctx, in, &limit)
			klog.Infof("Search Query:", query)
			//TODO: Check error
			srchRes, _ := searchResults(query, args)
			srchResult = append(srchResult, srchRes)
		}
	}
	return srchResult, nil
}

func searchQuery(ctx context.Context, input *model.SearchInput, limit *int) (string, []interface{}) {
	var selectClause, whereClause, limitClause, limitStr, query string
	var args []interface{}
	// SELECT uid, cluster, data FROM resources  WHERE lower(data->> 'kind') IN (lower('Pod')) AND lower(data->> 'cluster') IN (lower('local-cluster')) LIMIT 10000
	selectClause = "SELECT uid, cluster, data FROM resources "
	limitClause = " LIMIT "

	whereClause = " WHERE "

	for i, filter := range input.Filters {
		klog.Infof("Filters%d: %+v", i, *filter)
		// TODO: Handle other column names like kind and namespace
		if filter.Property == "cluster" {
			whereClause = whereClause + filter.Property
		} else {
			// TODO: To be removed when indexer handles this as adding lower hurts index scans
			whereClause = whereClause + "lower(data->> '" + filter.Property + "')"
		}
		var values []string

		if len(filter.Values) > 1 {
			for _, val := range filter.Values {
				klog.Infof("Filter value: %s", *val)
				values = append(values, strings.ToLower(*val))
				//TODO: Here, assuming value is string. Check for other cases.
				//TODO: Remove lower() conversion once data is correctly loaded from indexer
				// "SELECT id FROM resources WHERE status = any($1)"
				//SELECT id FROM resources WHERE status = ANY('{"Running", "Error"}');
			}
			whereClause = whereClause + "=any($" + strconv.Itoa(i+1) + ") AND "
			args = append(args, pq.Array(values))
		} else if len(filter.Values) == 1 {
			whereClause = whereClause + "=$" + strconv.Itoa(i+1) + " AND "
			val := filter.Values[0]
			args = append(args, strings.ToLower(*val))
		}
	}
	if input.Limit != nil {
		limitStr = strconv.Itoa(*input.Limit)
	}
	if limitStr != "" {
		limitClause = " LIMIT " + limitStr
		query = selectClause + strings.TrimRight(whereClause, trimAND) + limitClause

	} else {
		query = selectClause + strings.TrimRight(whereClause, trimAND)
	}
	klog.Infof("args: %+v", args)

	return query, args
}

func searchResults(query string, args []interface{}) (*model.SearchResult, error) {

	pool := db.GetConnection()
	rows, _ := pool.Query(context.Background(), query, args...)
	//TODO: Handle error
	defer rows.Close()
	var uid, cluster string
	var data map[string]interface{}
	items := []map[string]interface{}{}
	uidArray := make([]string, 0, len(items))

	for rows.Next() {

		// rowValues, _ := rows.Values()
		err := rows.Scan(&uid, &cluster, &data)
		if err != nil {
			klog.Errorf("Error %s retrieving rows for query:%s", err.Error(), query)
		}

		// TODO: To be removed when indexer handles this. Currently only string type is handled.
		currItem := make(map[string]interface{})
		for k, myInterface := range data {
			switch v := myInterface.(type) {
			case string:
				currItem[k] = strings.ToLower(v)
			default:
				// klog.Info("Not string type.", k, v)
				continue
			}

		}
		currUid := uid
		currItem["_uid"] = currUid
		currCluster := cluster
		currItem["cluster"] = currCluster
		items = append(items, currItem)

		uidArray = append(uidArray, currUid)

	}
	klog.Info("len search result items: ", len(items))
	totalCount := len(items)

	srchrelatedresult := getRelations(uidArray)

	srchresult1 := model.SearchResult{
		Count:   &totalCount,
		Items:   items,
		Related: srchrelatedresult,
	}
	return &srchresult1, nil
}

func getRelations(uidArray []string) []*model.SearchRelatedResult {

	pool := db.GetConnection()

	fmt.Println("FIRST UIDARRAY: ", uidArray[0])

	recrusiveQuery := `with recursive
	search_graph(uid, data, sourcekind, destkind, sourceid, destid, path, level)
	as (
	SELECT r.uid, r.data, e.sourcekind, e.destkind, e.sourceid, e.destid, ARRAY[r.uid] as path, 1 as level
		from resources r
		INNER JOIN
			edges e ON (r.uid = e.sourceid)
		 where r.uid in ($1)
	union
	select r.uid, r.data, e.sourcekind, e.destkind, e.sourceid, e.destid, path||r.uid, level+1 as level
		from resources r
		INNER JOIN
			edges e ON (r.uid = e.sourceid)
		, search_graph sg
		where (e.sourceid = sg.destid or e.destid = sg.sourceid)
		and r.uid <> all(sg.path)
		)

	select data, destid, destkind from search_graph where level= 1 or destid in ($1)`

	// TO-DO:need to find a way to improve performance when a list of uids is passed into uidArray
	// idea: we can save values into a CTE and create a second join.
	relations, QueryError := pool.Query(context.Background(), recrusiveQuery, uidArray[1])
	// cluster0/ee447c21-2360-4c8f-a673-5752df348e2f -uidArray[0]
	// "cluster0/636213bc-abeb-4f9e-923a-2834ffd26fe3"
	//need to give more context:
	if QueryError != nil {
		log.Fatal("query error :", QueryError)
	}

	defer relations.Close()

	items := []map[string]interface{}{}
	var kindSlice []string
	var totalCount []int
	// counter := make(map[string]int)
	// fmt.Printf("%T", relations)

	for relations.Next() {
		var destkind, destid string
		var data map[string]interface{}
		relatedResultError := relations.Scan(&data, &destid, &destkind)
		if relatedResultError != nil {
			klog.Errorf("Error %s retrieving rows for relationships:%s", relatedResultError.Error(), relations)
		} else {
			fmt.Println("No error with retrieving row results.")
		}
		currItem := make(map[string]interface{})
		for k, myInterface := range data {
			switch v := myInterface.(type) {
			case string:
				currItem[k] = strings.ToLower(v)
			default:
				// klog.Info("Not string type.", k, v)
				continue
			}
		}
		fmt.Println("Got the first item.")
		currKind := destkind
		currItem["Kind"] = currKind

		kindSlice = append(kindSlice, currKind)

		// // counter := 1
		// if kindInSlice(currKind, kindSlice) == true {
		// 	fmt.Println(currKind, "kind already exists.")
		// 	// kindSlice = append(kindSlice, currKind)
		// // counter++
		// // totalCount = append(totalCount, counter)

		// } else {
		// 	kindSlice = append(kindSlice, currKind)
		// // totalCount = append(totalCount, counter)
		// }

		// fmt.Println("counter:", counter)

		// totalCount = append(totalCount, counter)
		currUid := destid
		currItem["_id"] = currUid
		items = append(items, currItem)
		fmt.Println("appended items")

		fmt.Println("current kindSlice:", kindSlice)
		fmt.Println("current totalCount:", totalCount)
		fmt.Println("current items:", items)

	}

	count := printUniqueValue(kindSlice)

	fmt.Println("Count:", count)

	relatedSearch := make([]*model.SearchRelatedResult, len(count))

	// fmt.Println("total len of kindslice", len(kindSlice))
	// fmt.Println("total len of totalCount", len(totalCount))
	var kindList []string
	var countList []int

	for k, v := range count {

		fmt.Println("Keys:", k)
		kindList = append(kindList, k)
		fmt.Println("Values:", v)
		countList = append(countList, v)
	}

	// kind := count[k]
	// // fmt.Println(kind)
	// count := count[v]
	// // fmt.Println(totalCount)
	for i := range kindList {

		kind := kindList[i]
		count := countList[i]
		relatedSearch[i] = &model.SearchRelatedResult{kind, &count, items}
		// fmt.Println("Output: ", relatedSearch)
	}

	return relatedSearch
}

// func kindInSlice(destkind string, kindList []string) bool {
// 	for _, b := range kindList {
// 		if b == destkind {
// 			return true
// 		}
// 	}
// 	return false
// }

func printUniqueValue(arr []string) map[string]int {
	//Create a   dictionary of values for each element
	dict := make(map[string]int)
	for _, num := range arr {
		dict[num] = dict[num] + 1
	}
	return dict
}
