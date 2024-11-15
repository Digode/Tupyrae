package controller

import (
	"Tupyrae/internal/k8s"
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeobj "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	rt "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"k8s.io/klog/v2"
)

type ResourceWatcher struct {
	clientset *kubernetes.Clientset
	queue     workqueue.RateLimitingInterface
	informer  cache.SharedIndexInformer
}

func DeployWatcher(stop <-chan bool) *ResourceWatcher {
	klog.Infof("Starting ControllerRun.")

	clientset, err := k8s.GetClient()
	if err != nil {
		klog.Fatalf("Error getting clientset: %v", err)
	}

	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtimeobj.Object, error) {
				return clientset.AppsV1().Deployments("").List(context.Background(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return clientset.AppsV1().Deployments("").Watch(context.Background(), options)
			},
		},
		&appsv1.Deployment{},
		0,
		cache.Indexers{},
	)

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	return &ResourceWatcher{
		clientset: clientset,
		queue:     queue,
		informer:  informer,
	}
}

func CronjobWatcher(stop <-chan bool) *ResourceWatcher {
	klog.Infof("Starting ControllerRun.")

	clientset, err := k8s.GetClient()
	if err != nil {
		klog.Fatalf("Error getting clientset: %v", err)
	}

	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtimeobj.Object, error) {
				return clientset.BatchV1().CronJobs("").List(context.Background(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return clientset.BatchV1().CronJobs("").Watch(context.Background(), options)
			},
		},
		&batchv1.CronJob{},
		0,
		cache.Indexers{},
	)

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	return &ResourceWatcher{
		clientset: clientset,
		queue:     queue,
		informer:  informer,
	}
}

func (watcher *ResourceWatcher) Watch(stopCh <-chan struct{}) {
	klog.Infof("Starting watcher.")

	defer watcher.queue.ShutDown()
	defer rt.HandleCrash()

	go watcher.runWorker(stopCh)
	go watcher.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, watcher.informer.HasSynced) {
		rt.HandleError(fmt.Errorf("timeout waiting for cache sync"))
		return
	}

	klog.Infof("Watcher synced.")
}

func (watcher *ResourceWatcher) WatchCronjob(stopCh <-chan struct{}) {
	klog.Infof("Starting watcher.")

	defer watcher.queue.ShutDown()
	defer rt.HandleCrash()

	go watcher.runWorkerCronJob(stopCh)
	go watcher.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, watcher.informer.HasSynced) {
		rt.HandleError(fmt.Errorf("timeout waiting for cache sync"))
		return
	}

	klog.Infof("Watcher synced.")
}

func (rw *ResourceWatcher) runWorker(stopCh <-chan struct{}) {
	defer runtime.HandleCrash()

	rw.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			klog.Infof("Creating deployment: %v", obj.(*appsv1.Deployment).Name)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			klog.Infof("Updating deployment from: %v, to %v", oldObj.(*appsv1.Deployment).Name, newObj.(*appsv1.Deployment).Name)

		},
		DeleteFunc: func(obj interface{}) {
			klog.Infof("Deleting deployment: %v", obj.(*appsv1.Deployment).Name)

		},
	})

	go rw.informer.Run(stopCh)
	if !cache.WaitForCacheSync(stopCh, rw.informer.HasSynced) {
		klog.Fatalf("Fail to cache sync")
	}

	<-stopCh
}

func (rw *ResourceWatcher) runWorkerCronJob(stopCh <-chan struct{}) {
	defer runtime.HandleCrash()

	rw.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			klog.Infof("Creating cronjob: %v", obj.(*batchv1.CronJob).Name)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			klog.Infof("Updating cronjob from: %v, to %v", oldObj.(*batchv1.CronJob).Name, newObj.(*batchv1.CronJob).Name)

		},
		DeleteFunc: func(obj interface{}) {
			klog.Infof("Deleting cronjob: %v", obj.(*batchv1.CronJob).Name)

		},
	})

	go rw.informer.Run(stopCh)
	if !cache.WaitForCacheSync(stopCh, rw.informer.HasSynced) {
		klog.Fatalf("Fail to cache sync")
	}

	<-stopCh
}
