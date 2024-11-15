package k8s

import (
	"os"

	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetClient() (*kubernetes.Clientset, error) {
	config, err := getClientConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func GetAutoscalerClient() (*autoscalingv1beta2.Clientset, error) {
	config, err := getClientConfig()
	if err != nil {
		return nil, err
	}
	return autoscalingv1beta2.NewForConfig(config)
}

func getClientConfig() (*rest.Config, error) {
	if !isRunningInContainer() {
		return clientcmd.BuildConfigFromFlags("", "/Users/digode/.kube/kubeconfig")
	}

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
