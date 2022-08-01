package rbac

import (
	"context"
	"sync"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/stolostron/search-v2-api/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// Cache data shared across all users.
type SharedData struct {
	// These are the data fields.
	csResources     []resource // Cluster-scoped resources (ie. Node, ManagedCluster)
	namespaces      []string
	managedClusters []string

	// These are internal objects to track the state of the cache.
	mcErr       error      // Error while updating clusters data.
	mcLock      sync.Mutex // Locks when clusters data is being updated.
	mcUpdatedAt time.Time  // Time clusters was last updated.

	csErr       error      // Capture errors retrieving cluster-scoped resources.
	csLock      sync.Mutex // Locks the csResources map while updating it.
	csUpdatedAt time.Time  // Time when cluster-scoped data was last updated.

	nsErr       error      // Capture errors retrieving namespaces.
	nsLock      sync.Mutex // Locks the namespaces array while updating it.
	nsUpdatedAt time.Time  // Time when namespaces data was last updated.
}

type resource struct {
	apigroup string
	kind     string
}

var clusterVersionGvr = schema.GroupVersionResource{
	Group:    "cluster.open-cluster-management.io",
	Version:  "v1",
	Resource: "managedclusters",
}

func (cache *Cache) PopulateSharedCache(ctx context.Context) error {

	if sharedCacheValid(&cache.shared) { //if all cache is valid we use cache data
		klog.V(5).Info("Using shared data from cache.")
		return nil
	} else { //get data and cache

		// get all cluster-scoped resources and cache in shared.csResources
		err := cache.shared.getClusterScopedResources(cache, ctx)
		if err == nil {
			klog.V(6).Info("Successfully retrieved cluster scoped resources!")
		}
		// get all namespaces in cluster and cache in shared.namespaces.
		err = cache.shared.GetSharedNamespaces(cache, ctx)
		if err == nil {
			klog.V(6).Info("Successfully retrieved shared namespaces!")
		}
		// get all managed clustsers in cache
		err = cache.shared.GetSharedManagedClusters(cache, ctx)
		if err == nil {
			klog.V(6).Info("Successfully retrieved managed clusters!")
		}

		return err

	}

}

func sharedCacheValid(shared *SharedData) bool {

	if (time.Now().Before(shared.csUpdatedAt.Add(time.Duration(config.Cfg.SharedCacheTTL) * time.Millisecond))) &&
		(time.Now().Before(shared.nsUpdatedAt.Add(time.Duration(config.Cfg.SharedCacheTTL) * time.Millisecond))) &&
		(time.Now().Before(shared.mcUpdatedAt.Add(time.Duration(config.Cfg.SharedCacheTTL) * time.Millisecond))) {

		return true
	}
	return false
}

func (shared *SharedData) getClusterScopedResources(cache *Cache, ctx context.Context) error {

	// lock to prevent checking more than one at a time and check if cluster scoped resources already in cache
	shared.csLock.Lock()
	defer shared.csLock.Unlock()
	//clear previous cache
	shared.csResources = nil
	shared.csErr = nil
	klog.V(6).Info("Querying database for cluster-scoped resources.")

	// Building query to get cluster scoped resources
	// Original query: "SELECT DISTINCT(data->>apigroup, data->>kind) FROM search.resources WHERE
	// cluster='local-cluster' AND namespace=NULL"
	schemaTable := goqu.S("search").Table("resources")
	ds := goqu.From(schemaTable)
	query, _, err := ds.SelectDistinct(goqu.COALESCE(goqu.L(`"data"->>'apigroup'`), "").As("apigroup"),
		goqu.COALESCE(goqu.L(`"data"->>'kind_plural'`), "").As("kind")).
		Where(goqu.L(`"cluster"::TEXT = 'local-cluster'`), goqu.L(`"data"->>'namespace'`).IsNull()).ToSQL()
	if err != nil {
		klog.Errorf("Error creating query [%s]. Error: [%+v]", query, err)
		shared.csErr = err
		shared.csResources = []resource{}
		return shared.csErr
	}

	rows, queryerr := cache.pool.Query(ctx, query)
	if queryerr != nil {
		klog.Errorf("Error resolving query [%s]. Error: [%+v]", query, queryerr.Error())
		shared.csErr = queryerr
		shared.csResources = []resource{}
		return shared.csErr
	}

	if rows != nil {
		defer rows.Close()

		for rows.Next() {
			var kind string
			var apigroup string
			err := rows.Scan(&apigroup, &kind)
			if err != nil {
				klog.Errorf("Error %s retrieving rows for query:%s for apigroup %s and kind %s", err.Error(), query,
					apigroup, kind)
				continue
			}

			shared.csResources = append(shared.csResources, resource{apigroup: apigroup, kind: kind})

		}
	}
	shared.csUpdatedAt = time.Now()

	return shared.csErr
}

func (shared *SharedData) GetSharedNamespaces(cache *Cache, ctx context.Context) error {
	shared.nsLock.Lock()
	defer shared.nsLock.Unlock()
	//empty previous cache
	shared.namespaces = nil
	shared.nsErr = nil

	klog.V(5).Info("Getting namespaces from Kube Client..")

	clientset, kubeErr := kubernetes.NewForConfig(cache.restConfig)
	if kubeErr != nil {
		klog.Warning("Error with creating a new clientset.", kubeErr.Error())
		shared.nsErr = kubeErr
		return shared.nsErr
	}

	namespaceList, nsErr := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})

	if nsErr != nil {
		klog.Warning("Error resolving namespaces from KubeClient: ", nsErr)
		shared.nsErr = nsErr
		return shared.nsErr
	}

	// add namespaces to allNamespace List
	for _, n := range namespaceList.Items {
		shared.namespaces = append(shared.namespaces, n.Name)
	}
	shared.nsUpdatedAt = time.Now()

	return shared.nsErr
}

func (shared *SharedData) GetSharedManagedClusters(cache *Cache, ctx context.Context) error {

	shared.mcLock.Lock()
	defer shared.mcLock.Unlock()
	// clear previous cache
	shared.managedClusters = nil
	shared.mcErr = nil

	var managedClusters []string

	resourceObj, err := cache.dynamicClient.Resource(clusterVersionGvr).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Warning("Error resolving resources with dynamic client", err.Error())
		return shared.mcErr
	}

	for _, item := range resourceObj.Items {
		managedClusters = append(managedClusters, item.GetName())

	}

	shared.managedClusters = managedClusters
	shared.mcUpdatedAt = time.Now()
	shared.mcErr = nil
	return shared.mcErr

}
