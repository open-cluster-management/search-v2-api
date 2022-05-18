// Copyright Contributors to the Open Cluster Management project
package resolver

import (
	"context"
	"fmt"

	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stolostron/search-v2-api/graph/model"
	"github.com/stretchr/testify/assert"
)

func Test_SearchComplete_Query(t *testing.T) {
	// Create a SearchCompleteResolver instance with a mock connection pool.
	prop1 := "kind"
	searchInput := &model.SearchInput{}
	resolver, mockPool := newMockSearchComplete(t, searchInput, prop1)
	val1 := "Template"
	val2 := "ReplicaSet"
	val3 := "ConfigMap"
	expectedProps := []*string{&val1, &val2, &val3}

	// Mock the database queries.
	mockRows := newMockRows("../resolver/mocks/mock.json", searchInput, prop1)
	// Mock the database query
	mockPool.EXPECT().Query(gomock.Any(),
		gomock.Eq(`SELECT DISTINCT "data"->>'kind' FROM "search"."resources" WHERE ("data"->>'kind' IS NOT NULL) ORDER BY "data"->>'kind' ASC LIMIT 10000`),
		gomock.Eq([]interface{}{})).Return(mockRows, nil)

	// Execute function
	result, err := resolver.autoComplete(context.TODO())
	if err != nil {
		t.Errorf("Incorrect results. expected error to be [%v] got [%v]", nil, err)

	}
	// Verify response
	AssertStringArrayEqual(t, result, expectedProps, "Error in Test_SearchComplete_Query")
}

func Test_SearchCompleteNoProp_Query(t *testing.T) {
	// Create a SearchCompleteResolver instance with a mock connection pool.
	prop1 := ""
	searchInput := &model.SearchInput{}
	resolver, mockPool := newMockSearchComplete(t, searchInput, prop1)
	expectedProps := []*string{}

	// Mock the database query
	mockPool.EXPECT().Query(gomock.Any(),
		gomock.Eq(""),
		gomock.Eq([]interface{}{})).Return(nil, fmt.Errorf("Error in search complete query. No property specified."))

	// Execute function
	result, err := resolver.autoComplete(context.TODO())
	// Verify response
	AssertStringArrayEqual(t, result, expectedProps, "Error in Test_SearchCompleteNoProp_Query")
	assert.NotNil(t, err, "Expected error")
}

func Test_SearchCompleteWithFilter_Query(t *testing.T) {
	// Create a SearchCompleteResolver instance with a mock connection pool.
	prop1 := "kind"
	value1 := "openshift"
	value2 := "openshift-monitoring"
	cluster := "local-cluster"
	limit := 10
	searchInput := &model.SearchInput{Filters: []*model.SearchFilter{{Property: "namespace", Values: []*string{&value1, &value2}}, {Property: "cluster", Values: []*string{&cluster}}}, Limit: &limit}
	resolver, mockPool := newMockSearchComplete(t, searchInput, prop1)
	val1 := "Template"
	val2 := "ReplicaSet"
	val3 := "ConfigMap"
	expectedProps := []*string{&val1, &val2, &val3}

	// Mock the database queries.
	mockRows := newMockRows("../resolver/mocks/mock.json", searchInput, prop1)
	// Mock the database query
	mockPool.EXPECT().Query(gomock.Any(),
		gomock.Eq(`SELECT DISTINCT "data"->>'kind' FROM "search"."resources" WHERE (("data"->>'namespace' IN ('openshift', 'openshift-monitoring')) AND ("cluster" IN ('local-cluster')) AND ("data"->>'kind' IS NOT NULL)) ORDER BY "data"->>'kind' ASC LIMIT 10`),
		gomock.Eq([]interface{}{})).Return(mockRows, nil)

	// Execute function
	result, _ := resolver.autoComplete(context.TODO())

	// Verify response
	AssertStringArrayEqual(t, result, expectedProps, "Error in Test_SearchCompleteWithFilter_Query")
}

func Test_SearchCompleteWithCluster(t *testing.T) {
	// Create a SearchCompleteResolver instance with a mock connection pool.
	prop1 := "cluster"

	cluster := "local-cluster"
	limit := 10
	searchInput := &model.SearchInput{Limit: &limit}

	resolver, mockPool := newMockSearchComplete(t, searchInput, prop1)
	expectedProps := []*string{&cluster}

	// Mock the database queries.
	mockRows := newMockRows("../resolver/mocks/mock.json", searchInput, prop1)
	// Mock the database query
	mockPool.EXPECT().Query(gomock.Any(),
		gomock.Eq(`SELECT DISTINCT "cluster" FROM "search"."resources" WHERE (("cluster" IS NOT NULL) AND ("cluster" != '')) ORDER BY "cluster" ASC LIMIT 10`),
		gomock.Eq([]interface{}{})).Return(mockRows, nil)

	// Execute function
	result, _ := resolver.autoComplete(context.TODO())

	// Verify response
	AssertStringArrayEqual(t, result, expectedProps, "Error in Test_SearchCompleteWithFilter_Query")
}

func Test_SearchCompleteQuery_PropDate(t *testing.T) {
	// Create a SearchCompleteResolver instance with a mock connection pool.
	prop1 := "created"
	searchInput := &model.SearchInput{}
	resolver, mockPool := newMockSearchComplete(t, searchInput, prop1)
	val1 := "isDate"
	expectedProps := []*string{&val1} //, &val2, &val3}

	// Mock the database queries.
	mockRows := newMockRows("../resolver/mocks/mock.json", searchInput, prop1)
	// Mock the database query
	mockPool.EXPECT().Query(gomock.Any(),
		gomock.Eq(`SELECT DISTINCT "data"->>'created' FROM "search"."resources" WHERE ("data"->>'created' IS NOT NULL) ORDER BY "data"->>'created' ASC LIMIT 10000`),
		gomock.Eq([]interface{}{})).Return(mockRows, nil)

	// Execute function
	result, err := resolver.autoComplete(context.TODO())
	if err != nil {
		t.Errorf("Incorrect results. expected error to be [%v] got [%v]", nil, err)

	}
	// Verify response
	AssertStringArrayEqual(t, result, expectedProps, "Error in Test_SearchCompleteQuery_PropDate")
}

func Test_SearchCompleteQuery_PropNum(t *testing.T) {
	// Create a SearchCompleteResolver instance with a mock connection pool.
	prop1 := "current"
	searchInput := &model.SearchInput{}
	resolver, mockPool := newMockSearchComplete(t, searchInput, prop1)
	val1 := "isNumber"
	val2 := "3"
	expectedProps := []*string{&val1, &val2} //, &val3}

	// Mock the database queries.
	mockRows := newMockRows("../resolver/mocks/mock.json", searchInput, prop1)
	// Mock the database query
	mockPool.EXPECT().Query(gomock.Any(),
		gomock.Eq(`SELECT DISTINCT "data"->>'current' FROM "search"."resources" WHERE ("data"->>'current' IS NOT NULL) ORDER BY "data"->>'current' ASC LIMIT 10000`),
		gomock.Eq([]interface{}{})).Return(mockRows, nil)

	// Execute function
	result, err := resolver.autoComplete(context.TODO())
	if err != nil {
		t.Errorf("Incorrect results. expected error to be [%v] got [%v]", nil, err)

	}
	// Verify response
	AssertStringArrayEqual(t, result, expectedProps, "Error in Test_SearchCompleteQuery_PropNum")
}
