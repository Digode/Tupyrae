package controller

import (
	"Tupyrae/internal/handler"
	"Tupyrae/internal/k8s"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeobj "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	rt "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	autoscalerv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"k8s.io/klog/v2"
)

type ResourceWatcher struct {
	clientset interface{}
	queue     workqueue.RateLimitingInterface
	informer  cache.SharedIndexInformer
}

func Watcher() {
	klog.Infof("Starting Controller...")

	stop := make(chan bool)
	ns := NsWatcher(stop)
	vpa := VpaWatcher(stop)
	// deploy := DeployWatcher(stop)
	// cronjob := CronjobWatcher(stop)

	stopCh := make(chan struct{})

	ns.Watch(stopCh)
	vpa.Watch(stopCh)

	// go deploy.Watch(stopCh)
	// go cronjob.Watch(stopCh)

	sigCh := make(chan os.Signal, 1)

	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}

func NsWatcher(stop <-chan bool) *ResourceWatcher {
	klog.Infof("Starting NsWatcher...")

	clientset := k8s.GetClient()
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtimeobj.Object, error) {
				klog.Infof("ListFunc NS...")
				return clientset.CoreV1().Namespaces().List(context.Background(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				klog.Infof("WatchFunc NS...")
				return clientset.CoreV1().Namespaces().Watch(context.Background(), options)
			},
		},
		&corev1.Namespace{},
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

func DeployWatcher(stop <-chan bool) *ResourceWatcher {
	klog.Infof("Starting DeployWatcher...")

	clientset := k8s.GetClient()
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
	klog.Infof("Starting CronjobWatcher...")

	clientset := k8s.GetClient()
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

func VpaWatcher(stop <-chan bool) *ResourceWatcher {
	klog.Infof("Starting VpaWatcher...")

	clientset := k8s.GetAutoscalerClient()
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtimeobj.Object, error) {
				klog.Infof("ListFunc VPA...")
				return clientset.AutoscalingV1().VerticalPodAutoscalers("").List(context.Background(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				klog.Infof("WatchFunc VPA...")
				return clientset.AutoscalingV1().VerticalPodAutoscalers("").Watch(context.Background(), options)
			},
		},
		&autoscalerv1.VerticalPodAutoscaler{},
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
	klog.Infof("Starting watcher...")

	defer watcher.queue.ShutDown()
	defer rt.HandleCrash()

	go watcher.runWorker(stopCh)
	go watcher.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, watcher.informer.HasSynced) {
		rt.HandleError(fmt.Errorf("timeout waiting for cache sync"))
		return
	}

	klog.Infof("Watcher synced!")
}

func (rw *ResourceWatcher) runWorker(stopCh <-chan struct{}) {
	defer runtime.HandleCrash()

	rw.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			enqueueResource("Add", obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			enqueueResource("Update", newObj)
		},
		DeleteFunc: func(obj interface{}) {
			enqueueResource("Delete", obj)
		},
	})

	go rw.informer.Run(stopCh)
	if !cache.WaitForCacheSync(stopCh, rw.informer.HasSynced) {
		klog.Fatalf("Fail to cache sync!")
	}

	<-stopCh
}

func enqueueResource(action string, obj interface{}) error {
	if obj == nil {
		return fmt.Errorf("Object is nil")
	}

	resource := &handler.Resource{
		Action: action,
		Kind:   "Unknown",
		Item:   obj,
	}

	if deploy, ok := obj.(*appsv1.Deployment); ok {
		resource.Kind = "Deployment"
		resource.Name = deploy.Name
		resource.Namespace = deploy.Namespace
	} else if cron, ok := obj.(*batchv1.CronJob); ok {
		resource.Kind = "CronJob"
		resource.Name = cron.Name
		resource.Namespace = cron.Namespace
	} else if ns, ok := obj.(*corev1.Namespace); ok {
		resource.Kind = "Namespace"
		resource.Name = ns.Name
		resource.Namespace = ns.Name
	} else if vpa, ok := obj.(*autoscalerv1.VerticalPodAutoscaler); ok {
		resource.Kind = "VerticalPodAutoscaler"
		resource.Name = vpa.Name
		resource.Namespace = vpa.Namespace
	}

	return handler.Checker(resource)
}
