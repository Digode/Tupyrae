package k8s

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func GetCronJobs(namespace string) []batchv1.CronJob {
	resp, err := client.BatchV1().CronJobs(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Errorf("Error getting CronJobs: %v", err)
		return nil
	}

	return resp.Items
}

func GetCronJob(namespace, name string) (*batchv1.CronJob, error) {
	return client.BatchV1().CronJobs(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func UpdateCronJob(cronjob *batchv1.CronJob) (*batchv1.CronJob, error) {
	return client.BatchV1().CronJobs(cronjob.Namespace).Update(context.TODO(), cronjob, metav1.UpdateOptions{})
}
