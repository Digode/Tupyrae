package handler

import (
	"Tupyrae/internal/k8s"
	"fmt"
	"math"
	"time"

	"github.com/patrickmn/go-cache"
	v1 "k8s.io/api/core/v1"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/klog"
)

const (
	// DefaultExpiration is the default expiration time for an item in the cache
	DefaultExpiration = 15 * time.Minute
)

var resourcesCache = cache.New(DefaultExpiration, 30*time.Minute)

func VpaRun(r Resource) error {
	if _, ok := r.Item.(*vpav1.VerticalPodAutoscaler); !ok {
		return fmt.Errorf("Item is not a VPA")
	}

	vpa := r.Item.(*vpav1.VerticalPodAutoscaler)
	checkVpa(vpa)

	return nil
}

func keyCache(vpa *vpav1.VerticalPodAutoscaler) string {
	return vpa.Namespace + "/" + vpa.Spec.TargetRef.Name
}

func checkCache(vpa *vpav1.VerticalPodAutoscaler) bool {
	if _, found := resourcesCache.Get(keyCache(vpa)); found {
		return true
	}
	return false
}

func setCache(vpa *vpav1.VerticalPodAutoscaler) {
	resourcesCache.Set(keyCache(vpa), true, DefaultExpiration)
}

func checkVpa(vpa *vpav1.VerticalPodAutoscaler) {
	if checkCache(vpa) {
		return
	}

	switch vpa.Spec.TargetRef.Kind {
	case "Deployment":
		deployAdjust(vpa)
	case "CronJob":
		cronjobAdjust(vpa)
	default:
		klog.Errorf("Unsupported target kind: %s", vpa.Spec.TargetRef.Kind)
	}
}

func isIgnored(annotation map[string]string) bool {
	if _, ok := annotation["tupyrae/ignore"]; ok {
		return true
	}
	return false
}

func deployAdjust(vpa *vpav1.VerticalPodAutoscaler) {
	deploy, err := k8s.GetDeploy(vpa.Namespace, vpa.Spec.TargetRef.Name)
	if err != nil {
		klog.Error(err)
		return
	}

	if isIgnored(deploy.Annotations) {
		klog.Infof("Ignoring %s/%s", vpa.Namespace, vpa.Spec.TargetRef.Name)
		return
	}

	var updated bool = false
	if vpa.Status.Recommendation == nil || vpa.Status.Recommendation.ContainerRecommendations == nil || len(vpa.Status.Recommendation.ContainerRecommendations) == 0 {
		klog.Infof("No recommendation for %s/%s yet", vpa.Namespace, vpa.Spec.TargetRef.Name)
		return
	}
	for _, r := range vpa.Status.Recommendation.ContainerRecommendations {
		for i, c := range deploy.Spec.Template.Spec.Containers {
			if c.Name == r.ContainerName {
				if r.LowerBound != nil || r.UpperBound != nil {
					updatedC := c.DeepCopy()
					if r.LowerBound != nil && willAdjust(true, &c.Resources.Requests, &r.LowerBound) {
						updatedC.Resources.Requests = *&r.LowerBound
						updated = true
					}
					if r.UpperBound != nil && willAdjust(false, &c.Resources.Limits, &r.UpperBound) {
						updatedC.Resources.Limits = *&r.UpperBound
						updated = true
					}
					if updated {
						deploy.Spec.Template.Spec.Containers[i] = *updatedC
					}
				}
			}
		}
	}

	if updated {
		klog.Infof("Adjusting Deploy %s/%s: %v %v", vpa.Namespace, vpa.Spec.TargetRef.Name, vpa.Status.Recommendation.ContainerRecommendations[0].LowerBound, vpa.Status.Recommendation.ContainerRecommendations[0].UpperBound)
		_, err := k8s.UpdateDeploy(deploy)
		if err != nil {
			klog.Error(err)
			return
		}
		setCache(vpa)
	}
}

func cronjobAdjust(vpa *vpav1.VerticalPodAutoscaler) {
	cronjob, err := k8s.GetCronJob(vpa.Namespace, vpa.Spec.TargetRef.Name)
	if err != nil {
		klog.Error(err)
		return
	}

	if isIgnored(cronjob.Annotations) {
		klog.Infof("Ignoring %s/%s", vpa.Namespace, vpa.Spec.TargetRef.Name)
		return
	}

	var updated bool = false
	if vpa.Status.Recommendation == nil || vpa.Status.Recommendation.ContainerRecommendations == nil || len(vpa.Status.Recommendation.ContainerRecommendations) == 0 {
		klog.Infof("No recommendation for %s/%s yet", vpa.Namespace, vpa.Spec.TargetRef.Name)
		return
	}
	for _, r := range vpa.Status.Recommendation.ContainerRecommendations {
		for i, c := range cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers {
			if c.Name == r.ContainerName {
				if r.LowerBound != nil || r.UpperBound != nil {
					updatedC := c.DeepCopy()
					if r.LowerBound != nil && willAdjust(true, &c.Resources.Requests, &r.LowerBound) {
						updatedC.Resources.Requests = *&r.LowerBound
						updated = true
					}
					if r.UpperBound != nil && willAdjust(false, &c.Resources.Limits, &r.UpperBound) {
						updatedC.Resources.Limits = *&r.UpperBound
						updated = true
					}
					if updated {
						cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[i] = *updatedC
					}
				}
			}
		}
	}

	if updated {
		klog.Infof("Adjusting %s/%s: %v %v", vpa.Namespace, vpa.Spec.TargetRef.Name, vpa.Status.Recommendation.ContainerRecommendations[0].LowerBound, vpa.Status.Recommendation.ContainerRecommendations[0].UpperBound)
		_, err := k8s.UpdateCronJob(cronjob)
		if err != nil {
			klog.Errorf("Error updating CronJob: %v", err)
			return
		}
		setCache(vpa)
	}
}

func willAdjust(request bool, resource *v1.ResourceList, vpa *v1.ResourceList) bool {

	if vpa == nil || resource == vpa {
		return false
	}

	if request {
		if request && resource.Cpu().MilliValue() != vpa.Cpu().MilliValue() {
			return true
		}

		if !request && resource.Memory().MilliValue() != vpa.Memory().MilliValue() {
			return true
		}
	} else {
		if outOfLimit(resource.Cpu().MilliValue(), vpa.Cpu().MilliValue()) {
			return true
		}
		if outOfLimit(resource.Memory().MilliValue(), vpa.Memory().MilliValue()) {
			return true
		}
	}

	return false
}

func outOfLimit(resourceValue int64, vpaValue int64) bool {
	diff := 0.0
	if vpaValue > resourceValue {
		diff = float64(resourceValue) / float64(vpaValue)
	} else {
		diff = float64(vpaValue) / float64(resourceValue)
	}
	porc := 1 - diff
	// Check if the difference is greater than 10%
	if math.Abs(porc) > 0.1 {
		return true
	}
	return false
}
