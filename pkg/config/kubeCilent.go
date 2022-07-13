package config

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

func KubeClient() *kubernetes.Clientset {
	config := GetClientConfig()

	kClientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		klog.Fatal(err.Error())
	}
	return kClientset

}

func getKubeConfigPath() string {
	defaultKubePath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	if _, err := os.Stat(defaultKubePath); os.IsNotExist(err) {
		// set default to empty string if path does not reslove
		defaultKubePath = ""
	}

	kubeConfig := getEnv("KUBECONFIG", defaultKubePath)
	return kubeConfig
}

func GetClientConfig() *rest.Config {
	kubeConfigPath := getKubeConfigPath()
	var clientConfig *rest.Config
	var clientConfigError error

	if kubeConfigPath != "" {
		klog.V(6).Infof("Creating k8s client using KubeConfig at: %s", kubeConfigPath)
		clientConfig, clientConfigError = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	} else {
		klog.V(2).Info("Creating k8s client using InClusterClientConfig")
		clientConfig, clientConfigError = rest.InClusterConfig()
	}

	if clientConfigError != nil {
		klog.Fatal("Error getting Kube Config: ", clientConfigError)
	}

	return clientConfig
}

var dynamicClient dynamic.Interface

// Get the kubernetes dynamic client.
func GetDynamicClient() dynamic.Interface {

	if dynamicClient != nil {
		return dynamicClient
	}
	newDynamicClient, err := dynamic.NewForConfig(GetClientConfig())
	if err != nil {
		klog.Fatal("Cannot Construct Dynamic Client ", err)
	}
	dynamicClient = newDynamicClient

	return dynamicClient
}
