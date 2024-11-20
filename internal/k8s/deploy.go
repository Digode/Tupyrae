package k8s

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func GetDeploys(namespace string) []appsv1.Deployment {
	resp, err := GetClient().AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		klog.Errorf("Error getting Deployments: %v", err)
	}

	return resp.Items
}

func GetDeploy(namespace string, name string) (*appsv1.Deployment, error) {
	deploy, err := GetClient().AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return deploy, nil
}

func UpdateDeploy(deploy *appsv1.Deployment) (*appsv1.Deployment, error) {
	klog.Infof("Updating Deployment %s", deploy.Name)

	deploy, err := GetClient().AppsV1().Deployments(deploy.Namespace).Update(context.TODO(), deploy, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	return deploy, nil
}
