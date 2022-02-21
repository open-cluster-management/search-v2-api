// Copyright Contributors to the Open Cluster Management project
package resolver

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"sync"
	"testing"

	"github.com/driftprogramming/pgxpoolmock"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgproto3/v2"
	"github.com/stolostron/search-v2-api/graph/model"
	"k8s.io/klog/v2"
)

func newMockSearchResolver(t *testing.T, input *model.SearchInput, uids []*string) (*SearchResult, *pgxpoolmock.MockPgxPool) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockPool := pgxpoolmock.NewMockPgxPool(ctrl)

	mockResolver := &SearchResult{
		input: input,
		pool:  mockPool,
		uids:  uids,
		wg:    sync.WaitGroup{},
	}

	return mockResolver, mockPool
}
func newMockSearchComplete(t *testing.T, input *model.SearchInput, property string) (*SearchCompleteResult, *pgxpoolmock.MockPgxPool) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockPool := pgxpoolmock.NewMockPgxPool(ctrl)

	mockResolver := &SearchCompleteResult{
		input:    input,
		pool:     mockPool,
		property: property,
	}
	return mockResolver, mockPool
}

// ====================================================
// Mock the Row interface defined in the pgx library.
// https://github.com/jackc/pgx/blob/master/rows.go#L24
// ====================================================
type Row struct {
	MockValue int
}

func (r *Row) Scan(dest ...interface{}) error {
	*dest[0].(*int) = r.MockValue
	return nil
}

// ====================================================
// Mock the Rows interface defined in the pgx library.
// https://github.com/jackc/pgx/blob/master/rows.go#L24
// ====================================================

func newMockRows(mockDataFile string) *MockRows {
	// Read json file and build mock data
	bytes, _ := ioutil.ReadFile(mockDataFile)
	var data map[string]interface{}
	if err := json.Unmarshal(bytes, &data); err != nil {
		panic(err)
	}

	columns := data["columns"].([]interface{})
	columnHeaders := make([]string, len(columns))
	for i, col := range columns {
		columnHeaders[i] = col.(string)
	}

	items := data["records"].([]interface{})

	mockData := make([]map[string]interface{}, len(items))

	for i, item := range items {
		uid := item.(map[string]interface{})["uid"]
		mockData[i] = map[string]interface{}{
			"uid":      uid,
			"cluster":  strings.Split(uid.(string), "/")[0],
			"data":     item.(map[string]interface{})["properties"],
			"destid":   item.(map[string]interface{})["DestUID"],
			"destkind": item.(map[string]interface{})["DestKind"],
		}
	}

	return &MockRows{
		mockData:      mockData,
		index:         0,
		columnHeaders: columnHeaders,
	}
}

type MockRows struct {
	mockData      []map[string]interface{}
	index         int
	columnHeaders []string
}

func (r *MockRows) Close() {}

func (r *MockRows) Err() error { return nil }

func (r *MockRows) CommandTag() pgconn.CommandTag { return nil }

func (r *MockRows) FieldDescriptions() []pgproto3.FieldDescription { return nil }

func (r *MockRows) Next() bool {
	r.index = r.index + 1
	return r.index <= len(r.mockData)
}

func (r *MockRows) Scan(dest ...interface{}) error {
	if len(dest) > 1 { // For search function
		for i := range dest {
			switch v := dest[i].(type) {
			case *int:
				*dest[i].(*int) = r.mockData[r.index-1][r.columnHeaders[i]].(int)
			case *string:
				*dest[i].(*string) = r.mockData[r.index-1][r.columnHeaders[i]].(string)
			case *map[string]interface{}:
				*dest[i].(*map[string]interface{}) = r.mockData[r.index-1][r.columnHeaders[i]].(map[string]interface{})
			case nil:
				klog.Info("error type %T", v)
			default:
				klog.Info("unexpected type %T", v)

			}

		}
	} else if len(dest) == 1 { // For searchComplete function
		dataMap := r.mockData[r.index-1]["data"].(map[string]interface{})
		*dest[0].(*string) = dataMap["kind"].(string)
	}
	return nil
}

func (r *MockRows) Values() ([]interface{}, error) { return nil, nil }

func (r *MockRows) RawValues() [][]byte { return nil }

func AssertStringArrayEqual(t *testing.T, result, expected []*string, message string) {
	for i, exp := range expected {
		if *result[i] != *exp {
			t.Errorf("%s expected [%v] got [%v]", message, expected, result)
			return
		}
	}
}
