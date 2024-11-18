package handler

import (
	"Tupyrae/internal/k8s"
	"fmt"

	v1 "k8s.io/api/core/v1"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/klog"
)

func VpaRun(r Resource) error {
	klog.Infof("Starting ControllerRun.")

	if _, ok := r.Item.(*vpav1.VerticalPodAutoscaler); !ok {
		return fmt.Errorf("Item is not a VPA")
	}

	vpa := r.Item.(*vpav1.VerticalPodAutoscaler)
	checkVpa(vpa)

	return nil
}

func checkVpa(vpa *vpav1.VerticalPodAutoscaler) {
	klog.Infof("VPA %s", vpa.Name)

	switch vpa.Spec.TargetRef.Kind {
	case "Deployment":
		deployAdjust(vpa)
	case "CronJob":
		cronjobAdjust(vpa)
	default:
		klog.Errorf("Unsupported target kind: %s", vpa.Spec.TargetRef.Kind)
	}
}

func deployAdjust(vpa *vpav1.VerticalPodAutoscaler) {
	klog.Infof("Adjusting Deployment %s/%s", vpa.Namespace, vpa.Spec.TargetRef.Name)

	deploy, err := k8s.GetDeploy(vpa.Namespace, vpa.Spec.TargetRef.Name)
	if err != nil {
		klog.Errorf("Error getting Deployment: %v", err)
		return
	}

	var updated bool = false
	for _, r := range vpa.Status.Recommendation.ContainerRecommendations {
		for i, c := range deploy.Spec.Template.Spec.Containers {
			if c.Name == r.ContainerName {
				if r.LowerBound != nil || r.UpperBound != nil {
					klog.Infof("Adjusting %s/%s: %v %v", vpa.Namespace, r.ContainerName, r.LowerBound, r.UpperBound)

					updatedC := c.DeepCopy()
					if r.LowerBound != nil {
						updatedC.Resources.Requests = *&r.LowerBound
					}
					if r.UpperBound != nil {
						updatedC.Resources.Limits = *&r.UpperBound
					}
					deploy.Spec.Template.Spec.Containers[i] = *updatedC
					updated = true
				}
			}
		}
	}

	if updated {
		_, err := k8s.UpdateDeploy(deploy)
		if err != nil {
			klog.Errorf("Error updating Deployment: %v", err)
			return
		}
	}
}

func cronjobAdjust(vpa *vpav1.VerticalPodAutoscaler) {
	klog.Infof("Adjusting CronJob %s", vpa.Name)

	cronjob, err := k8s.GetCronJob(vpa.Namespace, vpa.Spec.TargetRef.Name)
	if err != nil {
		klog.Errorf("Error getting CronJob: %v", err)
		return
	}

	var updated bool = false
	for _, r := range vpa.Status.Recommendation.ContainerRecommendations {
		for i, c := range cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers {
			if c.Name == r.ContainerName {
				if r.LowerBound != nil || r.UpperBound != nil {
					klog.Infof("Adjusting %s/%s: %v %v", vpa.Namespace, r.ContainerName, r.LowerBound, r.UpperBound)

					updatedC := c.DeepCopy()
					if r.LowerBound != nil {
						updatedC.Resources.Requests = *&r.LowerBound
					}
					if r.UpperBound != nil {
						updatedC.Resources.Limits = *&r.UpperBound
					}
					cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[i] = *updatedC
					updated = true
				}
			}
		}
	}

	if updated {
		_, err := k8s.UpdateCronJob(cronjob)
		if err != nil {
			klog.Errorf("Error updating CronJob: %v", err)
			return
		}
	}
}

func willAdjust(request bool, resource *v1.ResourceList, vpa *v1.ResourceList) bool {

	if vpa == nil || resource == vpa {
		return false
	}

	if request {
		if request && resource.Cpu() != vpa.Cpu() {
			return true
		}
		if !request && resource.Memory() != vpa.Memory() {
			return true
		}
	} else {

	}

	return false
}
