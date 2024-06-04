// Copyright Contributors to the Open Cluster Management project
package federated

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"
	"sync"
	"time"

	config "github.com/stolostron/search-v2-api/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// This function returns an http client to communicate with the search-api service on the global hub cluster.
func getLocalHttpClient() HTTPClient {
	tlsConfig := tls.Config{
		MinVersion: tls.VersionTLS13,
		RootCAs:    x509.NewCertPool(),
	}

	if config.Cfg.DevelopmentMode {
		klog.Warningf("Running in DevelopmentMode. Using local self-signed certificate.")
		// Read the local self-signed CA bundle file.
		tlsCert, err := os.ReadFile("sslcert/tls.crt")
		if err != nil {
			klog.Errorf("Error reading local self-signed certificate: %s", err)
			klog.Info("Use 'make setup' to generate the local self-signed certificate.")
		} else {
			tlsConfig.RootCAs.AppendCertsFromPEM([]byte(tlsCert))
		}
	} else {
		client := config.KubeClient()
		caBundleConfigMap, err := client.CoreV1().ConfigMaps("open-cluster-management").Get(context.TODO(), "search-ca-crt", metav1.GetOptions{})
		if err != nil {
			klog.Errorf("Error getting the search-ca-crt configmap: %s", err)
		}
		tlsConfig.RootCAs.AppendCertsFromPEM([]byte(caBundleConfigMap.Data["service-ca.crt"]))
	}

	client := &RealHTTPClient{
		&http.Client{
			Transport: &http.Transport{
				MaxIdleConns:          config.Cfg.Federation.HttpPool.MaxIdleConns,
				IdleConnTimeout:       time.Duration(config.Cfg.Federation.HttpPool.MaxIdleConnTimeout) * time.Millisecond,
				ResponseHeaderTimeout: time.Duration(config.Cfg.Federation.HttpPool.ResponseHeaderTimeout) * time.Millisecond,
				DisableKeepAlives:     false,
				TLSClientConfig:       &tlsConfig,
				MaxConnsPerHost:       config.Cfg.Federation.HttpPool.MaxConnsPerHost,
				MaxIdleConnsPerHost:   config.Cfg.Federation.HttpPool.MaxIdleConnPerHost,
			},
			Timeout: time.Duration(config.Cfg.Federation.HttpPool.RequestTimeout) * time.Millisecond,
		},
	}
	return client
}

// Returns a client to process the federated request.
func GetHttpClient(remoteService RemoteSearchService) HTTPClient {
	if remoteService.Name == config.Cfg.Federation.GlobalHubName {
		return getLocalHttpClient()
	}

	// Get http client from pool.
	client := httpClientPool.Get().(*RealHTTPClient)

	// Set the TLS client configuration.
	tlsConfig := tls.Config{
		MinVersion: tls.VersionTLS13,
		RootCAs:    x509.NewCertPool(),
	}

	if len(remoteService.CABundle) > 0 {
		ok := tlsConfig.RootCAs.AppendCertsFromPEM(remoteService.CABundle)
		if !ok {
			klog.Warningf("Failed to append CA bundle for %s", remoteService.Name)
		}
	} else {
		klog.Warningf("TLS CA bundle not provided for remote service: %s.", remoteService.Name)
	}

	client.SetTLSClientConfig(&tlsConfig)

	return client
}

// shared HTTP transport and client for efficient connection reuse as per
// godoc: https://cs.opensource.google/go/go/+/go1.21.5:src/net/http/transport.go;l=95 and
// https://stuartleeks.com/posts/connection-re-use-in-golang-with-http-client/
var tr = &http.Transport{
	MaxIdleConns:          config.Cfg.Federation.HttpPool.MaxIdleConns,
	IdleConnTimeout:       time.Duration(config.Cfg.Federation.HttpPool.MaxIdleConnTimeout) * time.Millisecond,
	ResponseHeaderTimeout: time.Duration(config.Cfg.Federation.HttpPool.ResponseHeaderTimeout) * time.Millisecond,
	DisableKeepAlives:     false,
	TLSClientConfig: &tls.Config{
		MinVersion: tls.VersionTLS13,
		RootCAs:    x509.NewCertPool(),
	},
	MaxConnsPerHost:     config.Cfg.Federation.HttpPool.MaxConnsPerHost,
	MaxIdleConnsPerHost: config.Cfg.Federation.HttpPool.MaxIdleConnPerHost,
}

var httpClientPool = sync.Pool{
	New: func() interface{} {
		klog.V(6).Infof("Creating new RealHTTPClient from pool.")
		return &RealHTTPClient{
			&http.Client{
				Transport: tr,
				Timeout:   time.Duration(config.Cfg.Federation.HttpPool.RequestTimeout) * time.Millisecond,
			},
		}
	},
}

// HTTPClient is an interface for an HTTP client.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
	SetTLSClientConfig(*tls.Config)
}

// RealHTTPClient is a real implementation of the HTTPClient interface.
type RealHTTPClient struct {
	*http.Client
}

// Do implements the HTTPClient interface for RealHTTPClient.
func (c RealHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.Client.Do(req)
}

// SetTLSClientConfig sets the TLS client configuration for the HTTP client.
func (c RealHTTPClient) SetTLSClientConfig(config *tls.Config) {
	c.Transport.(*http.Transport).TLSClientConfig = config
}
