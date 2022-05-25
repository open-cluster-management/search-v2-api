// Copyright Contributors to the Open Cluster Management project
package rbac

import (
	"context"
	"time"

	"github.com/stolostron/search-v2-api/pkg/config"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type tokenReviewResult struct {
	updatedAt   time.Time
	tokenReview *authv1.TokenReview
	err         error
}

func (cache *Cache) IsValidToken(ctx context.Context, token string) (bool, error) {
	tr, err := cache.getTokenReview(ctx, token)

	return tr.Status.Authenticated, err
}

func (cache *Cache) getTokenReview(ctx context.Context, token string) (*authv1.TokenReview, error) {
	cache.tokenReviewsLock.Lock()

	// Check if we can use TokenReview from the cache.
	tr, tokenExists := cache.tokenReviews[token]
	if tokenExists && time.Now().Before(tr.updatedAt.Add(time.Duration(config.Cfg.AuthCacheTTL)*time.Millisecond)) {
		klog.V(5).Info("Using TokenReview from cache.")
		cache.tokenReviewsLock.Unlock()
		return tr.tokenReview, tr.err
	}
	cache.tokenReviewsLock.Unlock()

	// Start a new TokenReview request.
	result := make(chan *tokenReviewResult)
	go cache.doTokenReview(ctx, token, result)

	// Wait until the TokenReview request gets resolved.
	tr = <-result
	return tr.tokenReview, tr.err
}

func (cache *Cache) doTokenReview(ctx context.Context, token string, ch chan *tokenReviewResult) {
	cache.tokenReviewsLock.Lock()
	// Check if there's a pending TokenReview
	_, foundPending := cache.tokenReviewsPending[token]
	if foundPending {
		klog.V(5).Info("Found a pending TokenReview, adding channel to get notified when resolved.")
		cache.tokenReviewsPending[token] = append(cache.tokenReviewsPending[token], ch)
		cache.tokenReviewsLock.Unlock()
		return
	} else {
		klog.V(5).Info("Triggering a new TokenReview request.")
		cache.tokenReviewsPending[token] = []chan *tokenReviewResult{ch}
	}
	cache.tokenReviewsLock.Unlock()

	// Create a new TokenReview request.
	tr := authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{
			Token: token,
		},
	}
	result, err := config.KubeClient().AuthenticationV1().TokenReviews().Create(ctx, &tr, metav1.CreateOptions{})
	if err != nil {
		klog.Warning("Error during TokenReview. ", err.Error())
	}

	cache.tokenReviewsLock.Lock()
	defer cache.tokenReviewsLock.Unlock()

	// Send the response to all channels registered in the tokenReviewPending object.
	pending := cache.tokenReviewsPending[token]
	trResult := &tokenReviewResult{updatedAt: time.Now(), tokenReview: result, err: err}
	for _, p := range pending {
		p <- trResult
	}

	delete(cache.tokenReviewsPending, token)
	cache.tokenReviews[token] = trResult
}
