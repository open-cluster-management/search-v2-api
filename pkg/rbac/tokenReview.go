// Copyright Contributors to the Open Cluster Management project
package rbac

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/stolostron/search-v2-api/pkg/config"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/authentication/v1"
	"k8s.io/klog/v2"
)

// Encapsulates a TokenReview to store in the cache.
type tokenReviewCache struct {
	err         error
	lock        sync.Mutex
	updatedAt   time.Time
	token       string
	tokenReview *authv1.TokenReview
}

// Verify that the token is valid using a TokenReview.
// Will use cached data if available and valid, otherwise starts a new request.
func (cache *Cache) IsValidToken(ctx context.Context, token string) (bool, error) {
	tr, err := cache.getTokenReview(ctx, token)
	klog.Infof("IsValidToken() %+v  %+v", tr, err)
	return tr.Status.Authenticated, err
}

// Get the TokenReview response for a given token.
// Will use cached data if available and valid, otherwise starts a new request.
func (cache *Cache) getTokenReview(ctx context.Context, token string) (*authv1.TokenReview, error) {
	cache.tokenReviewsLock.Lock()
	defer cache.tokenReviewsLock.Unlock()

	// Check if a TokenReviewCacheRequest exists in the cache or create a new one.
	cachedTR, tokenExists := cache.tokenReviews[token]
	if !tokenExists {
		cachedTR = &tokenReviewCache{
			token: token,
		}
		cache.tokenReviews[token] = cachedTR
	}
	return cachedTR.getTokenReview()
}

// Get the resolved TokenReview from the cached tokenReviewCachedRequest object.
func (trr *tokenReviewCache) getTokenReview() (*authv1.TokenReview, error) {
	// This ensures that only 1 process is updating the TokenReview data from API request.
	trr.lock.Lock()
	defer trr.lock.Unlock()

	// Check if cached TokenReview data is valid. Update if needed.
	if time.Now().After(trr.updatedAt.Add(time.Duration(config.Cfg.AuthCacheTTL) * time.Millisecond)) {
		klog.Infof("Resolving TokenReview. tokenReviewCacheRequest expired or never updated. Last update %s", trr.updatedAt)

		tr := authv1.TokenReview{
			Spec: authv1.TokenReviewSpec{
				Token: trr.token,
			},
		}
		// result, err := cache.getAuthClient().TokenReviews().Create(context.TODO(), &tr, metav1.CreateOptions{})
		result, err := config.KubeClient().AuthenticationV1().TokenReviews().Create(context.TODO(), &tr, metav1.CreateOptions{})
		if err != nil {
			klog.Warning("Error resolving TokenReview from Kube API.", err.Error())
		}
		klog.V(9).Infof("TokenReview Kube API result: %v\n", prettyPrint(result.Status))

		trr.updatedAt = time.Now()
		trr.err = err
		trr.tokenReview = result
	} else {
		klog.V(6).Info("Using cached TokenReview.")
	}

	return trr.tokenReview, trr.err
}

// https://stackoverflow.com/a/51270134
func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}

// Utility to allow tests to inject a fake client to mock the k8s api call.
func (cache *Cache) getAuthClient() v1.AuthenticationV1Interface {
	if cache.authClient == nil {
		cache.authClient = config.KubeClient().AuthenticationV1()
	}
	return cache.authClient
}
