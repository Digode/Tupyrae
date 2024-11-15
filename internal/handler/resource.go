package handler

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
)

type Requirement struct {
	Cpu    int64 `json:"cpu"`
	Memory int64 `json:"memory"`
}

type Resource struct {
	Name      string      `json:"name"`
	Namespace string      `json:"namespace"`
	Kind      string      `json:"kind"`
	Item      interface{} `json:"item"`
}

func (r *Resource) Checker() error {
	switch r.Kind {
	case "Deployment":
		if _, ok := r.Item.(*appsv1.Deployment); !ok {
			return fmt.Errorf("Item is not a Deployment")
		}
	case "CronJob":
		if _, ok := r.Item.(*batchv1.CronJob); !ok {
			return fmt.Errorf("Item is not a CronJob")
		}
	}

	return nil
}
