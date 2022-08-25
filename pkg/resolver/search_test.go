// Copyright Contributors to the Open Cluster Management project
package resolver

import (
	"context"
	"testing"

	"github.com/doug-martin/goqu/v9"
	"github.com/golang/mock/gomock"
	"github.com/stolostron/search-v2-api/graph/model"
	"github.com/stolostron/search-v2-api/pkg/rbac"
	"github.com/stretchr/testify/assert"
)

func Test_SearchResolver_Count(t *testing.T) {

	// Create a SearchResolver instance with a mock connection pool.
	val1 := "Pod"
	searchInput := &model.SearchInput{Filters: []*model.SearchFilter{{Property: "kind", Values: []*string{&val1}}}}
	resolver, mockPool := newMockSearchResolver(t, searchInput, nil, &rbac.UserData{})

	// Mock the database query
	mockRow := &Row{MockValue: 10}
	mockPool.EXPECT().QueryRow(gomock.Any(),
		gomock.Eq(`SELECT COUNT("uid") FROM "search"."resources" WHERE (("data"->>'kind' IN ('Pod')) AND (("cluster" = ANY (NULL)) OR ((data->>'_hubClusterResource' = 'true') AND NULL)))`),
		gomock.Eq([]interface{}{})).Return(mockRow)

	// Execute function
	r := resolver.Count()

	// Verify response
	if r != mockRow.MockValue {
		t.Errorf("Incorrect Count() expected [%d] got [%d]", mockRow.MockValue, r)
	}
}

func Test_SearchResolver_Count_WithRBAC(t *testing.T) {
	csRes, nsRes, managedClusters := newUserResourceAccess()
	ud := rbac.UserData{CsResources: csRes, NsResources: nsRes, ManagedClusters: managedClusters}
	// Create a SearchResolver instance with a mock connection pool.
	val1 := "Pod"
	searchInput := &model.SearchInput{Filters: []*model.SearchFilter{{Property: "kind", Values: []*string{&val1}}}}
	resolver, mockPool := newMockSearchResolver(t, searchInput, nil, &ud)

	// Mock the database query
	mockRow := &Row{MockValue: 10}
	mockPool.EXPECT().QueryRow(gomock.Any(),
		gomock.Eq(`SELECT COUNT("uid") FROM "search"."resources" WHERE (("data"->>'kind' IN ('Pod')) AND (("cluster" = ANY ('{"managed1","managed2"}')) OR ((data->>'_hubClusterResource' = 'true') AND (((COALESCE(data->>'namespace', '') = '') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'nodes')) OR ((COALESCE(data->>'apigroup', '') = 'storage.k8s.io') AND (data->>'kind_plural' = 'csinodes')))) OR (((data->>'namespace' = 'default') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'configmaps')) OR ((COALESCE(data->>'apigroup', '') = 'v4') AND (data->>'kind_plural' = 'services')))) OR ((data->>'namespace' = 'ocm') AND (((COALESCE(data->>'apigroup', '') = 'v1') AND (data->>'kind_plural' = 'pods')) OR ((COALESCE(data->>'apigroup', '') = 'v2') AND (data->>'kind_plural' = 'deployments')))))))))`),
		gomock.Eq([]interface{}{})).Return(mockRow)

	// Execute function
	r := resolver.Count()

	// Verify response
	if r != mockRow.MockValue {
		t.Errorf("Incorrect Count() expected [%d] got [%d]", mockRow.MockValue, r)
	}
}

func Test_SearchResolver_CountWithOperator(t *testing.T) {
	// Create a SearchResolver instance with a mock connection pool.
	val1 := ">=1"
	searchInput := &model.SearchInput{Filters: []*model.SearchFilter{{Property: "current", Values: []*string{&val1}}}}
	ud := rbac.UserData{}
	resolver, mockPool := newMockSearchResolver(t, searchInput, nil, &ud)

	// Mock the database query
	mockRow := &Row{MockValue: 1}
	mockPool.EXPECT().QueryRow(gomock.Any(),
		gomock.Eq(`SELECT COUNT("uid") FROM "search"."resources" WHERE (("data"->>'current' >= '1') AND (("cluster" = ANY (NULL)) OR ((data->>'_hubClusterResource' = 'true') AND NULL)))`),
		gomock.Eq([]interface{}{})).Return(mockRow)

	// Execute function
	r := resolver.Count()
	// Verify response
	if r != mockRow.MockValue {
		t.Errorf("Incorrect Count() expected [%d] got [%d]", mockRow.MockValue, r)
	}
}
func Test_SearchResolver_Items(t *testing.T) {
	// Create a SearchResolver instance with a mock connection pool.
	val1 := "template"
	searchInput := &model.SearchInput{Filters: []*model.SearchFilter{{Property: "kind", Values: []*string{&val1}}}}
	ud := rbac.UserData{}
	resolver, mockPool := newMockSearchResolver(t, searchInput, nil, &ud)
	// Mock the database queries.
	mockRows := newMockRowsWithoutRBAC("./mocks/mock.json", searchInput, "", 0)

	mockPool.EXPECT().Query(gomock.Any(),
		gomock.Eq(`SELECT DISTINCT "uid", "cluster", "data" FROM "search"."resources" WHERE (("data"->>'kind' ILIKE ANY ('{"template"}')) AND (("cluster" = ANY (NULL)) OR ((data->>'_hubClusterResource' = 'true') AND NULL))) LIMIT 1000`),
		gomock.Eq([]interface{}{}),
	).Return(mockRows, nil)

	// Execute the function
	result := resolver.Items()

	// Verify returned items.
	if len(result) != len(mockRows.mockData) {
		t.Errorf("Items() received incorrect number of items. Expected %d Got: %d", len(mockRows.mockData), len(result))
	}

	// Verify properties for each returned item.
	for i, item := range result {
		mockRow := mockRows.mockData[i]
		expectedRow := formatDataMap(mockRow["data"].(map[string]interface{}))
		expectedRow["_uid"] = mockRow["uid"]
		expectedRow["cluster"] = mockRow["cluster"]

		if len(item) != len(expectedRow) {
			t.Errorf("Number of properties don't match for item[%d]. Expected: %d Got: %d", i, len(expectedRow), len(item))
		}

		for key, val := range item {
			if val != expectedRow[key] {
				t.Errorf("Value of key [%s] does not match for item [%d].\nExpected: %s\nGot: %s", key, i, expectedRow[key], val)
			}
		}
	}
}

type TestOperatorItem struct {
	searchInput *model.SearchInput
	mockQuery   string
}

func Test_SearchResolver_ItemsWithNumOperator(t *testing.T) {
	// rbac := buildRbacWhereClause(context.TODO(), &rbac.UserResourceAccess{csres, nsres, mc})

	val1 := ">1"
	testOperatorGreater := TestOperatorItem{
		searchInput: &model.SearchInput{Filters: []*model.SearchFilter{{Property: "current", Values: []*string{&val1}}}},
		mockQuery:   `SELECT DISTINCT "uid", "cluster", "data" FROM "search"."resources" WHERE (("data"->>'current' > '1') AND (("cluster" = ANY ('{"managed1","managed2"}')) OR ((data->>'_hubClusterResource' = 'true') AND (((COALESCE(data->>'namespace', '') = '') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'nodes')) OR ((COALESCE(data->>'apigroup', '') = 'storage.k8s.io') AND (data->>'kind_plural' = 'csinodes')))) OR (((data->>'namespace' = 'default') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'configmaps')) OR ((COALESCE(data->>'apigroup', '') = 'v4') AND (data->>'kind_plural' = 'services')))) OR ((data->>'namespace' = 'ocm') AND (((COALESCE(data->>'apigroup', '') = 'v1') AND (data->>'kind_plural' = 'pods')) OR ((COALESCE(data->>'apigroup', '') = 'v2') AND (data->>'kind_plural' = 'deployments'))))))))) LIMIT 1000`,
	}
	val2 := "<4"
	testOperatorLesser := TestOperatorItem{
		searchInput: &model.SearchInput{Filters: []*model.SearchFilter{{Property: "current", Values: []*string{&val2}}}},
		mockQuery:   `SELECT DISTINCT "uid", "cluster", "data" FROM "search"."resources" WHERE (("data"->>'current' < '4') AND (("cluster" = ANY ('{"managed1","managed2"}')) OR ((data->>'_hubClusterResource' = 'true') AND (((COALESCE(data->>'namespace', '') = '') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'nodes')) OR ((COALESCE(data->>'apigroup', '') = 'storage.k8s.io') AND (data->>'kind_plural' = 'csinodes')))) OR (((data->>'namespace' = 'default') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'configmaps')) OR ((COALESCE(data->>'apigroup', '') = 'v4') AND (data->>'kind_plural' = 'services')))) OR ((data->>'namespace' = 'ocm') AND (((COALESCE(data->>'apigroup', '') = 'v1') AND (data->>'kind_plural' = 'pods')) OR ((COALESCE(data->>'apigroup', '') = 'v2') AND (data->>'kind_plural' = 'deployments'))))))))) LIMIT 1000`,
	}
	val3 := ">=1"
	testOperatorGreaterorEqual := TestOperatorItem{
		searchInput: &model.SearchInput{Filters: []*model.SearchFilter{{Property: "current", Values: []*string{&val3}}}},
		mockQuery:   `SELECT DISTINCT "uid", "cluster", "data" FROM "search"."resources" WHERE (("data"->>'current' >= '1') AND (("cluster" = ANY ('{"managed1","managed2"}')) OR ((data->>'_hubClusterResource' = 'true') AND (((COALESCE(data->>'namespace', '') = '') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'nodes')) OR ((COALESCE(data->>'apigroup', '') = 'storage.k8s.io') AND (data->>'kind_plural' = 'csinodes')))) OR (((data->>'namespace' = 'default') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'configmaps')) OR ((COALESCE(data->>'apigroup', '') = 'v4') AND (data->>'kind_plural' = 'services')))) OR ((data->>'namespace' = 'ocm') AND (((COALESCE(data->>'apigroup', '') = 'v1') AND (data->>'kind_plural' = 'pods')) OR ((COALESCE(data->>'apigroup', '') = 'v2') AND (data->>'kind_plural' = 'deployments'))))))))) LIMIT 1000`,
	}
	val4 := "<=3"
	testOperatorLesserorEqual := TestOperatorItem{
		searchInput: &model.SearchInput{Filters: []*model.SearchFilter{{Property: "current", Values: []*string{&val4}}}},
		mockQuery:   `SELECT DISTINCT "uid", "cluster", "data" FROM "search"."resources" WHERE (("data"->>'current' <= '3') AND (("cluster" = ANY ('{"managed1","managed2"}')) OR ((data->>'_hubClusterResource' = 'true') AND (((COALESCE(data->>'namespace', '') = '') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'nodes')) OR ((COALESCE(data->>'apigroup', '') = 'storage.k8s.io') AND (data->>'kind_plural' = 'csinodes')))) OR (((data->>'namespace' = 'default') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'configmaps')) OR ((COALESCE(data->>'apigroup', '') = 'v4') AND (data->>'kind_plural' = 'services')))) OR ((data->>'namespace' = 'ocm') AND (((COALESCE(data->>'apigroup', '') = 'v1') AND (data->>'kind_plural' = 'pods')) OR ((COALESCE(data->>'apigroup', '') = 'v2') AND (data->>'kind_plural' = 'deployments'))))))))) LIMIT 1000`,
	}

	val5 := "!4"
	testOperatorNot := TestOperatorItem{
		searchInput: &model.SearchInput{Filters: []*model.SearchFilter{{Property: "current", Values: []*string{&val5}}}},
		mockQuery:   `SELECT DISTINCT "uid", "cluster", "data" FROM "search"."resources" WHERE (("data"->>'current' NOT IN ('4')) AND (("cluster" = ANY ('{"managed1","managed2"}')) OR ((data->>'_hubClusterResource' = 'true') AND (((COALESCE(data->>'namespace', '') = '') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'nodes')) OR ((COALESCE(data->>'apigroup', '') = 'storage.k8s.io') AND (data->>'kind_plural' = 'csinodes')))) OR (((data->>'namespace' = 'default') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'configmaps')) OR ((COALESCE(data->>'apigroup', '') = 'v4') AND (data->>'kind_plural' = 'services')))) OR ((data->>'namespace' = 'ocm') AND (((COALESCE(data->>'apigroup', '') = 'v1') AND (data->>'kind_plural' = 'pods')) OR ((COALESCE(data->>'apigroup', '') = 'v2') AND (data->>'kind_plural' = 'deployments'))))))))) LIMIT 1000`,
	}

	val6 := "!=4"
	testOperatorNotEqual := TestOperatorItem{
		searchInput: &model.SearchInput{Filters: []*model.SearchFilter{{Property: "current", Values: []*string{&val6}}}},
		mockQuery:   `SELECT DISTINCT "uid", "cluster", "data" FROM "search"."resources" WHERE (("data"->>'current' NOT IN ('4')) AND (("cluster" = ANY ('{"managed1","managed2"}')) OR ((data->>'_hubClusterResource' = 'true') AND (((COALESCE(data->>'namespace', '') = '') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'nodes')) OR ((COALESCE(data->>'apigroup', '') = 'storage.k8s.io') AND (data->>'kind_plural' = 'csinodes')))) OR (((data->>'namespace' = 'default') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'configmaps')) OR ((COALESCE(data->>'apigroup', '') = 'v4') AND (data->>'kind_plural' = 'services')))) OR ((data->>'namespace' = 'ocm') AND (((COALESCE(data->>'apigroup', '') = 'v1') AND (data->>'kind_plural' = 'pods')) OR ((COALESCE(data->>'apigroup', '') = 'v2') AND (data->>'kind_plural' = 'deployments'))))))))) LIMIT 1000`,
	}

	val7 := "=3"
	testOperatorEqual := TestOperatorItem{
		searchInput: &model.SearchInput{Filters: []*model.SearchFilter{{Property: "current", Values: []*string{&val7}}}},
		mockQuery:   `SELECT DISTINCT "uid", "cluster", "data" FROM "search"."resources" WHERE (("data"->>'current' IN ('3')) AND (("cluster" = ANY ('{"managed1","managed2"}')) OR ((data->>'_hubClusterResource' = 'true') AND (((COALESCE(data->>'namespace', '') = '') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'nodes')) OR ((COALESCE(data->>'apigroup', '') = 'storage.k8s.io') AND (data->>'kind_plural' = 'csinodes')))) OR (((data->>'namespace' = 'default') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'configmaps')) OR ((COALESCE(data->>'apigroup', '') = 'v4') AND (data->>'kind_plural' = 'services')))) OR ((data->>'namespace' = 'ocm') AND (((COALESCE(data->>'apigroup', '') = 'v1') AND (data->>'kind_plural' = 'pods')) OR ((COALESCE(data->>'apigroup', '') = 'v2') AND (data->>'kind_plural' = 'deployments'))))))))) LIMIT 1000`,
	}

	testOperatorMultiple := TestOperatorItem{
		searchInput: &model.SearchInput{Filters: []*model.SearchFilter{{Property: "current", Values: []*string{&val1, &val2}}}},
		mockQuery:   `SELECT DISTINCT "uid", "cluster", "data" FROM "search"."resources" WHERE ((("data"->>'current' < '4') OR ("data"->>'current' > '1')) AND (("cluster" = ANY ('{"managed1","managed2"}')) OR ((data->>'_hubClusterResource' = 'true') AND (((COALESCE(data->>'namespace', '') = '') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'nodes')) OR ((COALESCE(data->>'apigroup', '') = 'storage.k8s.io') AND (data->>'kind_plural' = 'csinodes')))) OR (((data->>'namespace' = 'default') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'configmaps')) OR ((COALESCE(data->>'apigroup', '') = 'v4') AND (data->>'kind_plural' = 'services')))) OR ((data->>'namespace' = 'ocm') AND (((COALESCE(data->>'apigroup', '') = 'v1') AND (data->>'kind_plural' = 'pods')) OR ((COALESCE(data->>'apigroup', '') = 'v2') AND (data->>'kind_plural' = 'deployments'))))))))) LIMIT 1000`,
	}

	testOperators := []TestOperatorItem{
		testOperatorGreater, testOperatorLesser, testOperatorGreaterorEqual,
		testOperatorLesserorEqual, testOperatorNot, testOperatorNotEqual, testOperatorEqual,
		testOperatorMultiple,
	}
	testAllOperators(t, testOperators)
}
func Test_SearchResolver_ItemsWithDateOperator(t *testing.T) {
	//define schema table:
	schemaTable := goqu.S("search").Table("resources")
	ds := goqu.From(schemaTable)
	csres, nsres, mc := newUserResourceAccess()
	rbac := buildRbacWhereClause(context.TODO(), &rbac.UserData{CsResources: csres, NsResources: nsres, ManagedClusters: mc})
	val8 := "year"
	opValMap := getOperatorAndNumDateFilter([]string{val8})
	mockQueryYear, _, _ := ds.SelectDistinct("uid", "cluster", "data").Where(goqu.L(`"data"->>?`, "created").Gt(opValMap[">"][0]), rbac).Limit(1000).ToSQL()

	testOperatorYear := TestOperatorItem{
		searchInput: &model.SearchInput{Filters: []*model.SearchFilter{{Property: "created", Values: []*string{&val8}}}},
		mockQuery:   mockQueryYear, // `SELECT "uid", "cluster", "data" FROM "search"."resources" WHERE ("data"->>'created' > ('2021-05-16T13:11:12Z')) LIMIT 1000`,
	}

	val9 := "hour"
	opValMap = getOperatorAndNumDateFilter([]string{val9})
	mockQueryHour, _, _ := ds.SelectDistinct("uid", "cluster", "data").Where(goqu.L(`"data"->>?`, "created").Gt(opValMap[">"][0]), rbac).Limit(1000).ToSQL()

	testOperatorHour := TestOperatorItem{
		searchInput: &model.SearchInput{Filters: []*model.SearchFilter{{Property: "created", Values: []*string{&val9}}}},
		mockQuery:   mockQueryHour, // `SELECT "uid", "cluster", "data" FROM "search"."resources" WHERE ("data"->>'created' > ('2021-05-16T13:11:12Z')) LIMIT 1000`,
	}

	val10 := "day"
	opValMap = getOperatorAndNumDateFilter([]string{val10})
	mockQueryDay, _, _ := ds.SelectDistinct("uid", "cluster", "data").Where(goqu.L(`"data"->>?`, "created").Gt(goqu.L("?", opValMap[">"][0])), rbac).Limit(1000).ToSQL()

	testOperatorDay := TestOperatorItem{
		searchInput: &model.SearchInput{Filters: []*model.SearchFilter{{Property: "created", Values: []*string{&val10}}}},
		mockQuery:   mockQueryDay, // `SELECT "uid", "cluster", "data" FROM "search"."resources" WHERE ("data"->>'created' > ('2021-05-16T13:11:12Z')) LIMIT 1000`,
	}

	val11 := "week"
	opValMap = getOperatorAndNumDateFilter([]string{val11})
	mockQueryWeek, _, _ := ds.SelectDistinct("uid", "cluster", "data").Where(goqu.L(`"data"->>?`, "created").Gt(goqu.L("?", opValMap[">"][0])), rbac).Limit(1000).ToSQL()

	testOperatorWeek := TestOperatorItem{
		searchInput: &model.SearchInput{Filters: []*model.SearchFilter{{Property: "created", Values: []*string{&val11}}}},
		mockQuery:   mockQueryWeek, // `SELECT "uid", "cluster", "data" FROM "search"."resources" WHERE ("data"->>'created' > ('2021-05-16T13:11:12Z')) LIMIT 1000`,
	}

	val12 := "month"
	opValMap = getOperatorAndNumDateFilter([]string{val12})
	mockQueryMonth, _, _ := ds.SelectDistinct("uid", "cluster", "data").Where(goqu.L(`"data"->>?`, "created").Gt(goqu.L("?", opValMap[">"][0])), rbac).Limit(1000).ToSQL()

	testOperatorMonth := TestOperatorItem{
		searchInput: &model.SearchInput{Filters: []*model.SearchFilter{{Property: "created", Values: []*string{&val12}}}},
		mockQuery:   mockQueryMonth, // `SELECT "uid", "cluster", "data" FROM "search"."resources" WHERE ("data"->>'created' > ('2021-05-16T13:11:12Z')) LIMIT 1000`,
	}

	opValMap = getOperatorAndNumDateFilter([]string{val8, val9})
	mockQueryMultiple, _, _ := ds.SelectDistinct("uid", "cluster", "data").Where(goqu.Or(goqu.L(`"data"->>?`, "created").Gt(opValMap[">"][0]),
		goqu.L(`"data"->>?`, "created").Gt(opValMap[">"][1])), rbac).Limit(1000).ToSQL()

	testoperatorMultiple := TestOperatorItem{
		searchInput: &model.SearchInput{Filters: []*model.SearchFilter{{Property: "created", Values: []*string{&val8, &val9}}}},
		mockQuery:   mockQueryMultiple, // `SELECT "uid", "cluster", "data" FROM "search"."resources" WHERE ("data"->>'created' > ('2021-05-16T13:11:12Z')) LIMIT 1000`,
	}
	testOperators := []TestOperatorItem{
		testOperatorYear, testOperatorHour, testOperatorDay, testOperatorWeek, testOperatorMonth,
		testoperatorMultiple,
	}
	testAllOperators(t, testOperators)

}

func testAllOperators(t *testing.T, testOperators []TestOperatorItem) {
	for _, currTest := range testOperators {
		csRes, nsRes, mc := newUserResourceAccess()
		ud := rbac.UserData{CsResources: csRes, NsResources: nsRes, ManagedClusters: mc}
		// Create a SearchResolver instance with a mock connection pool.
		resolver, mockPool := newMockSearchResolver(t, currTest.searchInput, nil, &ud)
		// Mock the database queries.
		mockRows := newMockRowsWithoutRBAC("./mocks/mock.json", currTest.searchInput, "", 0)

		mockPool.EXPECT().Query(gomock.Any(),
			gomock.Eq(currTest.mockQuery),
			gomock.Eq([]interface{}{}),
		).Return(mockRows, nil)

		// Execute the function
		result := resolver.Items()
		// Verify returned items.
		if len(result) != len(mockRows.mockData) {
			t.Errorf("Items() received incorrect number of items. Expected %d Got: %d", len(mockRows.mockData), len(result))
		}

		// // Verify properties for each returned item.
		for i, item := range result {
			mockRow := mockRows.mockData[i]
			expectedRow := formatDataMap(mockRow["data"].(map[string]interface{}))
			expectedRow["_uid"] = mockRow["uid"]
			expectedRow["cluster"] = mockRow["cluster"]

			if len(item) != len(expectedRow) {
				t.Errorf("Number of properties don't match for item[%d]. Expected: %d Got: %d", i, len(expectedRow), len(item))
			}

			for key, val := range item {
				if val != expectedRow[key] {
					t.Errorf("Value of key [%s] does not match for item [%d].\nExpected: %s\nGot: %s", key, i, expectedRow[key], val)
				}
			}
		}
	}
}
func Test_SearchResolver_Items_Multiple_Filter(t *testing.T) {
	// Create a SearchResolver instance with a mock connection pool.
	val1 := "openshift"
	val2 := "openshift-monitoring"
	cluster := "local-cluster"
	limit := 10
	searchInput := &model.SearchInput{Filters: []*model.SearchFilter{{Property: "namespace", Values: []*string{&val1, &val2}}, {Property: "cluster", Values: []*string{&cluster}}}, Limit: &limit}
	ud := rbac.UserData{}
	resolver, mockPool := newMockSearchResolver(t, searchInput, nil, &ud)

	// Mock the database queries.
	mockRows := newMockRowsWithoutRBAC("./mocks/mock.json", searchInput, "", 0)
	mockPool.EXPECT().Query(gomock.Any(),
		gomock.Eq(`SELECT DISTINCT "uid", "cluster", "data" FROM "search"."resources" WHERE (("data"->>'namespace' IN ('openshift', 'openshift-monitoring')) AND ("cluster" IN ('local-cluster')) AND (("cluster" = ANY (NULL)) OR ((data->>'_hubClusterResource' = 'true') AND NULL))) LIMIT 10`),
		// gomock.Eq("SELECT uid, cluster, data FROM search.resources  WHERE lower(data->> 'namespace')=any($1) AND cluster=$2 LIMIT 10"),
		gomock.Eq([]interface{}{}),
	).Return(mockRows, nil)

	// Execute the function
	result := resolver.Items()

	// Verify returned items.
	if len(result) != len(mockRows.mockData) {
		t.Errorf("Items() received incorrect number of items. Expected %d Got: %d", len(mockRows.mockData), len(result))
	}

	// Verify properties for each returned item.
	for i, item := range result {
		mockRow := mockRows.mockData[i]
		expectedRow := formatDataMap(mockRow["data"].(map[string]interface{}))
		expectedRow["_uid"] = mockRow["uid"]
		expectedRow["cluster"] = mockRow["cluster"]

		if len(item) != len(expectedRow) {
			t.Errorf("Number of properties don't match for item[%d]. Expected: %d Got: %d", i, len(expectedRow), len(item))
		}

		for key, val := range item {
			if val != expectedRow[key] {
				t.Errorf("Value of key [%s] does not match for item [%d].\nExpected: %s\nGot: %s", key, i, expectedRow[key], val)
			}
		}
	}
}

func Test_SearchWithMultipleClusterFilter_NegativeLimit_Query(t *testing.T) {
	// Create a SearchResolver instance with a mock connection pool.
	value1 := "openshift"
	cluster1 := "local-cluster"
	cluster2 := "remote-1"
	limit := -1
	searchInput := &model.SearchInput{Filters: []*model.SearchFilter{{Property: "namespace", Values: []*string{&value1}}, {Property: "cluster", Values: []*string{&cluster1, &cluster2}}}, Limit: &limit}
	ud := rbac.UserData{}
	resolver, mockPool := newMockSearchResolver(t, searchInput, nil, &ud)

	// Mock the database queries.
	mockRows := newMockRowsWithoutRBAC("../resolver/mocks/mock.json", searchInput, "", 0)

	// Mock the database query
	mockPool.EXPECT().Query(gomock.Any(),
		gomock.Eq(`SELECT DISTINCT "uid", "cluster", "data" FROM "search"."resources" WHERE (("data"->>'namespace' IN ('openshift')) AND ("cluster" IN ('local-cluster', 'remote-1')) AND (("cluster" = ANY (NULL)) OR ((data->>'_hubClusterResource' = 'true') AND NULL)))`),
		gomock.Eq([]interface{}{})).Return(mockRows, nil)

	// Execute function
	result := resolver.Items()

	// Verify returned items.
	if len(result) != len(mockRows.mockData) {
		t.Errorf("Items() received incorrect number of items. Expected %d Got: %d", len(mockRows.mockData), len(result))
	}

	// Verify properties for each returned item.
	for i, item := range result {
		mockRow := mockRows.mockData[i]
		expectedRow := formatDataMap(mockRow["data"].(map[string]interface{}))
		expectedRow["_uid"] = mockRow["uid"]
		expectedRow["cluster"] = mockRow["cluster"]

		if len(item) != len(expectedRow) {
			t.Errorf("Number of properties don't match for item[%d]. Expected: %d Got: %d", i, len(expectedRow), len(item))
		}

		for key, val := range item {
			if val != expectedRow[key] {
				t.Errorf("Value of key [%s] does not match for item [%d].\nExpected: %s\nGot: %s", key, i, expectedRow[key], val)
			}
		}
	}
}

func Test_SearchResolver_Keywords(t *testing.T) {
	// Create a SearchResolver instance with a mock connection pool.
	val1 := "Template"
	limit := 10
	searchInput := &model.SearchInput{Keywords: []*string{&val1}, Limit: &limit}
	ud := rbac.UserData{}
	resolver, mockPool := newMockSearchResolver(t, searchInput, nil, &ud)

	// Mock the database queries.
	mockRows := newMockRowsWithoutRBAC("./mocks/mock.json", searchInput, "", 0)

	mockPool.EXPECT().Query(gomock.Any(),
		gomock.Eq(`SELECT DISTINCT "uid", "cluster", "data" FROM "search"."resources", jsonb_each_text("data") WHERE (("value" LIKE '%Template%') AND (("cluster" = ANY (NULL)) OR ((data->>'_hubClusterResource' = 'true') AND NULL))) LIMIT 10`),
		gomock.Eq([]interface{}{}),
	).Return(mockRows, nil)

	// // Execute the function
	result := resolver.Items()

	// Verify properties for each returned item.
	for i, item := range result {
		mockRow := mockRows.mockData[i]
		expectedRow := formatDataMap(mockRow["data"].(map[string]interface{}))
		expectedRow["_uid"] = mockRow["uid"]
		expectedRow["cluster"] = mockRow["cluster"]

		if len(item) != len(expectedRow) {
			t.Errorf("Number of properties don't match for item[%d]. Expected: %d Got: %d", i, len(expectedRow), len(item))
		}

		for key, val := range item {
			if val != expectedRow[key] {
				t.Errorf("Value of key [%s] does not match for item [%d].\nExpected: %s\nGot: %s", key, i, expectedRow[key], val)
			}
		}
	}
}

func Test_SearchResolver_Uids(t *testing.T) {
	// Create a SearchResolver instance with a mock connection pool.
	val1 := "template"
	searchInput := &model.SearchInput{Filters: []*model.SearchFilter{{Property: "kind", Values: []*string{&val1}}}}
	resolver, mockPool := newMockSearchResolver(t, searchInput, nil, &rbac.UserData{})
	// Mock the database queries.
	mockRows := newMockRowsWithoutRBAC("./mocks/mock.json", searchInput, "", 0)

	mockPool.EXPECT().Query(gomock.Any(),
		gomock.Eq(`SELECT "uid" FROM "search"."resources" WHERE (("data"->>'kind' ILIKE ANY ('{"template"}')) AND (("cluster" = ANY (NULL)) OR ((data->>'_hubClusterResource' = 'true') AND NULL))) LIMIT 1000`),
		gomock.Eq([]interface{}{}),
	).Return(mockRows, nil)

	// Execute the function
	resolver.Uids()

	// Verify returned items.
	if len(resolver.uids) != len(mockRows.mockData) {
		t.Errorf("Items() received incorrect number of items. Expected %d Got: %d", len(mockRows.mockData), len(resolver.uids))
	}

	// Verify properties for each returned item.
	for i, item := range resolver.uids {
		mockRow := mockRows.mockData[i]
		expectedRow := formatDataMap(mockRow["data"].(map[string]interface{}))
		expectedRow["_uid"] = mockRow["uid"]

		if *item != mockRow["uid"].(string) {
			t.Errorf("Value of key [uid] does not match for item [%d].\nExpected: %s\nGot: %s", i, expectedRow["_uid"], *item)
		}
	}
}

func Test_buildRbacWhereClauseCs(t *testing.T) {
	csres, _, _ := newUserResourceAccess()
	ud := rbac.UserData{CsResources: csres}

	rbacCombined := buildRbacWhereClause(context.TODO(), &ud)
	expectedSql := `SELECT * WHERE (("cluster" = ANY (NULL)) OR ((data->>'_hubClusterResource' = 'true') AND ((COALESCE(data->>'namespace', '') = '') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'nodes')) OR ((COALESCE(data->>'apigroup', '') = 'storage.k8s.io') AND (data->>'kind_plural' = 'csinodes'))))))`
	gotSql, _, _ := goqu.Select().Where(rbacCombined).ToSQL()
	assert.Equal(t, expectedSql, gotSql)

}

func Test_buildRbacWhereClauseNs(t *testing.T) {
	_, nsScopeAccess, _ := newUserResourceAccess()
	ud := rbac.UserData{NsResources: nsScopeAccess}
	rbacCombined := buildRbacWhereClause(context.TODO(), &ud)
	expectedSql := `SELECT * WHERE (("cluster" = ANY (NULL)) OR ((data->>'_hubClusterResource' = 'true') AND (NULL OR (((data->>'namespace' = 'default') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'configmaps')) OR ((COALESCE(data->>'apigroup', '') = 'v4') AND (data->>'kind_plural' = 'services')))) OR ((data->>'namespace' = 'ocm') AND (((COALESCE(data->>'apigroup', '') = 'v1') AND (data->>'kind_plural' = 'pods')) OR ((COALESCE(data->>'apigroup', '') = 'v2') AND (data->>'kind_plural' = 'deployments'))))))))`
	gotSql, _, _ := goqu.Select().Where(rbacCombined).ToSQL()
	assert.Equal(t, expectedSql, gotSql)

}

func Test_buildRbacWhereClauseCsAndNs(t *testing.T) {
	res, nsScopeAccess, _ := newUserResourceAccess()
	ud := rbac.UserData{CsResources: res, NsResources: nsScopeAccess}
	rbacCombined := buildRbacWhereClause(context.TODO(), &ud)
	expectedSql := `SELECT * WHERE (("cluster" = ANY (NULL)) OR ((data->>'_hubClusterResource' = 'true') AND (((COALESCE(data->>'namespace', '') = '') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'nodes')) OR ((COALESCE(data->>'apigroup', '') = 'storage.k8s.io') AND (data->>'kind_plural' = 'csinodes')))) OR (((data->>'namespace' = 'default') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'configmaps')) OR ((COALESCE(data->>'apigroup', '') = 'v4') AND (data->>'kind_plural' = 'services')))) OR ((data->>'namespace' = 'ocm') AND (((COALESCE(data->>'apigroup', '') = 'v1') AND (data->>'kind_plural' = 'pods')) OR ((COALESCE(data->>'apigroup', '') = 'v2') AND (data->>'kind_plural' = 'deployments'))))))))`
	gotSql, _, _ := goqu.Select().Where(rbacCombined).ToSQL()
	assert.Equal(t, expectedSql, gotSql)

}

func Test_buildRbacWhereClauseCsNsAndMc(t *testing.T) {
	csres, nsScopeAccess, managedClusters := newUserResourceAccess()
	ud := rbac.UserData{CsResources: csres, NsResources: nsScopeAccess, ManagedClusters: managedClusters}
	rbacCombined := buildRbacWhereClause(context.TODO(), &ud)
	expectedSql := `SELECT * WHERE (("cluster" = ANY ('{"managed1","managed2"}')) OR ((data->>'_hubClusterResource' = 'true') AND (((COALESCE(data->>'namespace', '') = '') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'nodes')) OR ((COALESCE(data->>'apigroup', '') = 'storage.k8s.io') AND (data->>'kind_plural' = 'csinodes')))) OR (((data->>'namespace' = 'default') AND (((COALESCE(data->>'apigroup', '') = '') AND (data->>'kind_plural' = 'configmaps')) OR ((COALESCE(data->>'apigroup', '') = 'v4') AND (data->>'kind_plural' = 'services')))) OR ((data->>'namespace' = 'ocm') AND (((COALESCE(data->>'apigroup', '') = 'v1') AND (data->>'kind_plural' = 'pods')) OR ((COALESCE(data->>'apigroup', '') = 'v2') AND (data->>'kind_plural' = 'deployments'))))))))`
	gotSql, _, _ := goqu.Select().Where(rbacCombined).ToSQL()
	assert.Equal(t, expectedSql, gotSql)
}
