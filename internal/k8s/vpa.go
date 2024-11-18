package k8s

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/klog/v2"
)

func GetVpa(namespace string, name string) (*v1.VerticalPodAutoscaler, error) {
	vpas, err := clientAutoscaling.AutoscalingV1().VerticalPodAutoscalers(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return vpas, nil
}

func GetVpas(namespace string) ([]v1.VerticalPodAutoscaler, error) {
	vpas, err := clientAutoscaling.AutoscalingV1().VerticalPodAutoscalers(namespace).List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return nil, err
	}
	return vpas.Items, nil
}

func CreateVpa(vpa *v1.VerticalPodAutoscaler) (*v1.VerticalPodAutoscaler, error) {
	klog.Infof("Creating VPA %s", vpa.Name)

	vpa, err := clientAutoscaling.AutoscalingV1().VerticalPodAutoscalers(vpa.Namespace).Create(context.TODO(), vpa, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return vpa, nil
}
