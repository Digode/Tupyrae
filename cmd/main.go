package main

import (
	"Tupyrae/internal/controller"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/klog/v2"
)

func main() {
	klog.Infof("Starting main")
	stop := make(chan bool, 1)
	defer close(stop)
	stop <- true

	deployController := controller.DeployWatcher(stop)
	cronjobController := controller.CronjobWatcher(stop)

	stopCh := make(chan struct{})

	go deployController.Watch(stopCh)
	go cronjobController.WatchCronjob(stopCh)

	sigCh := make(chan os.Signal, 1)

	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
