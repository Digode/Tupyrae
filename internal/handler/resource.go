package handler

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/klog/v2"
)

type Requirement struct {
	Cpu    int64 `json:"cpu"`
	Memory int64 `json:"memory"`
}

type Resource struct {
	Action    string      `json:"action"`
	Name      string      `json:"name"`
	Namespace string      `json:"namespace"`
	Kind      string      `json:"kind"`
	Item      interface{} `json:"item"`
}

func Checker(r *Resource) error {
	klog.Infof("%s %s: %s/%s", r.Action, r.Kind, r.Namespace, r.Name)
	switch r.Kind {
	case "Deployment":
		if _, ok := r.Item.(*appsv1.Deployment); !ok {
			return fmt.Errorf("Item is not a Deployment")
		}
	case "CronJob":
		if _, ok := r.Item.(*batchv1.CronJob); !ok {
			return fmt.Errorf("Item is not a CronJob")
		}
	case "Namespace":
		if _, ok := r.Item.(*corev1.Namespace); !ok {
			return fmt.Errorf("Item is not a Namespace")
		}
		NsRun(*r)
	case "VerticalPodAutoscaler":
		if _, ok := r.Item.(*vpav1.VerticalPodAutoscaler); !ok {
			return fmt.Errorf("Item is not a VPA")
		}
		VpaRun(*r)
	}

	return nil
}
