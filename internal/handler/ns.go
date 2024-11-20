package handler

import (
	"Tupyrae/internal/k8s"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/klog"
)

const (
	key = "tupyrae"
	val = "true"
)

func NsRun(r Resource) error {
	if _, ok := r.Item.(*corev1.Namespace); !ok {
		return fmt.Errorf("Item is not a Namespace")
	}

	ns := r.Item.(*corev1.Namespace)
	checkNamespace(ns)

	return nil
}

func checkNamespace(namespace *corev1.Namespace) {
	if namespace.Labels[key] == val {
		vpas := mapperVpa(namespace)
		for _, deploy := range k8s.GetDeploys(namespace.Name) {
			key := getKey(deploy)
			if _, ok := vpas[key]; !ok {
				createVpaByDeployment(deploy)
			}
		}

		for _, cron := range k8s.GetCronJobs(namespace.Name) {
			key := getKey(cron)
			if _, ok := vpas[key]; !ok {
				createVpaByCronJob(cron)
			}
		}
	}
}

func mapperVpa(ns *corev1.Namespace) map[string]vpav1.VerticalPodAutoscaler {
	vpas, err := k8s.GetVpas(ns.Name)
	if err != nil {
		klog.Errorf("Error getting Vpas: %v", err)
		return nil
	}
	mapVpa := make((map[string]vpav1.VerticalPodAutoscaler), 0)
	for _, vpa := range vpas {
		mapKey := getKey(vpa)
		if _, ok := mapVpa[mapKey]; !ok {
			mapVpa[mapKey] = vpa
		}
	}
	return mapVpa
}

func createVpa(name string, namespace string, kind string, apiVersion string, labels map[string]string) {
	var mode vpav1.UpdateMode = vpav1.UpdateModeOff
	vpa := &vpav1.VerticalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"owener": key,
			},
		},
	}

	for _, l := range labels {
		vpa.ObjectMeta.Labels[l] = val
	}

	vpa.ObjectMeta.Labels["owener"] = key

	vpa.Spec = vpav1.VerticalPodAutoscalerSpec{
		TargetRef: &autoscaling.CrossVersionObjectReference{
			APIVersion: apiVersion,
			Kind:       kind,
			Name:       name,
		},
		UpdatePolicy: &vpav1.PodUpdatePolicy{
			UpdateMode: &mode,
		},
	}

	_, err := k8s.CreateVpa(vpa)
	if err != nil {
		klog.Errorf("Error creating VPA for %s: %v", name, err)
	}
}

func createVpaByDeployment(deploy appsv1.Deployment) {
	createVpa(deploy.Name, deploy.Namespace, "Deployment", "apps/v1", deploy.Labels)
}

func createVpaByCronJob(cron batchv1.CronJob) {
	createVpa(cron.Name, cron.Namespace, "CronJob", "batch/v1", cron.Labels)
}

func getKey(obj interface{}) string {
	switch obj.(type) {
	case appsv1.Deployment:
		return fmt.Sprintf("%s_%s", "Deployment", obj.(appsv1.Deployment).Name)
	case batchv1.CronJob:
		return fmt.Sprintf("%s_%s", "CronJob", obj.(batchv1.CronJob).Name)
	case vpav1.VerticalPodAutoscaler:
		vpa := obj.(vpav1.VerticalPodAutoscaler)
		return fmt.Sprintf("%s_%s", vpa.Spec.TargetRef.Kind, vpa.Spec.TargetRef.Name)
	default:
		klog.Errorf("Unknown type %T", obj)
		return ""
	}
}
