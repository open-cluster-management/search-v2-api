// Copyright Contributors to the Open Cluster Management project
package resolver

type SearchRelatedResult struct {
	// input *model.SearchInput //nolint: unused,structcheck
	Kind  string                   `json:"kind"`
	Count *int                     `json:"count"`
	Items []map[string]interface{} `json:"items"`
}

// func (s *SearchRelatedResult) Count() int {
// 	klog.Info("TODO: Resolve SearchRelatedResult: Count() - model/related.go")
// 	return 0
// }

// func (s *SearchRelatedResult) Kind() string {
// 	klog.Info("TODO: Resolve SearchRelatedResult: Kind()  - model/related.go")
// 	return "TODO:Kind"
// }

// func (s *SearchRelatedResult) Items() []map[string]interface{} {
// 	klog.Info("TODO: Resolve SearchRelatedResult: Items() - model/related.go")
// 	return nil
// }
