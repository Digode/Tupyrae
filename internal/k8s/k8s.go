package k8s

import (
	"os"

	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

var client *kubernetes.Clientset
var clientAutoscaling *autoscalingv1beta2.Clientset

func GetClient() *kubernetes.Clientset {
	if client == nil {
		config, err := getClientConfig()
		if err != nil {
			return nil
		}
		cli, err := kubernetes.NewForConfig(config)
		if err != nil {
			return nil
		}

		client = cli
	}
	return client
}

func GetAutoscalerClient() *autoscalingv1beta2.Clientset {
	if clientAutoscaling == nil {
		config, err := getClientConfig()
		if err != nil {
			return nil
		}

		cli, err := autoscalingv1beta2.NewForConfig(config)
		if err != nil {
			return nil
		}
		clientAutoscaling = cli
	}

	return clientAutoscaling
}

func getClientConfig() (*rest.Config, error) {
	if !isRunningInContainer() {
		klog.Info("Running locally, using kubeconfig")
		return clientcmd.BuildConfigFromFlags("", "/Users/digode/.kube/config")
	}

	klog.Info("Running in container, using in-cluster config")
	return rest.InClusterConfig()
}

func buildInClusterConfig() (*kubernetes.Clientset, error) {
	config, err := getClientConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

func buildInClusterAutoscalerConfig() (*autoscalingv1beta2.Clientset, error) {
	config, err := getClientConfig()
	if err != nil {
		return nil, err
	}

	return autoscalingv1beta2.NewForConfig(config)
}

func isRunningInContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err != nil {
		return false
	}
	return true
}
