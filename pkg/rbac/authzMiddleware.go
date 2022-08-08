package rbac

import (
	"net/http"

	db "github.com/stolostron/search-v2-api/pkg/database"
	"github.com/stolostron/search-v2-api/pkg/metric"
	"k8s.io/klog/v2"
)

func AuthorizeUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		//Check db connection TODO: create time based check(s)
		cacheInst.pool = db.GetConnection()

		//Hub Cluster resources authorization:
		err := cacheInst.PopulateSharedCache(r.Context())
		if err != nil {
			klog.Warning("Unexpected error while obtaining cluster-scoped resources.", err)
			metric.AuthzFailed.WithLabelValues("UnexpectedAuthzError").Inc()
		}
		klog.Info("Finished getting shared resources. Now getting user data..")

		clientToken := r.Context().Value(ContextAuthTokenKey).(string)
		_, userErr := cacheInst.GetUserData(r.Context(), clientToken, nil)
		if userErr != nil {
			klog.Warning("Unexpected error while obtaining user data.", userErr)
		}

		//Managed Cluster resources authorization:
		// userData.getManagedClusterResources()

		klog.V(5).Info("User authorization successful!")
		next.ServeHTTP(w, r.WithContext(r.Context()))

	})
}
