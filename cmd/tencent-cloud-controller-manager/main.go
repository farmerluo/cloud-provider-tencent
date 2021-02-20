package main

import (
	"math/rand"
	"os"
	"time"

	"k8s.io/component-base/logs"
	"k8s.io/kubernetes/cmd/cloud-controller-manager/app"

	_ "k8s.io/component-base/metrics/prometheus/version" // for version metric registration
	// NOTE: Importing all in-tree cloud-providers is not required when
	// implementing an out-of-tree cloud-provider.
	_ "k8s.io/component-base/metrics/prometheus/clientgo" // load all the prometheus client-go plugins
	_ "k8s.io/kubernetes/pkg/cloudprovider/providers"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	command := app.NewCloudControllerManagerCommand()

	// TODO: once we switch everything over to Cobra commands, we can go back to calling
	// utilflag.InitFlags() (by removing its pflag.Parse() call). For now, we have to set the
	// normalize func and add the go flag set by hand.
	// utilflag.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}