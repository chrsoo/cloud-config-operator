package main

import (
	"strings"
	"errors"
	"reflect"
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/chrsoo/cloud-config-operator/pkg/apis/k8/v1alpha1"

	"github.com/chrsoo/cloud-config-operator/pkg/apis"
	"github.com/chrsoo/cloud-config-operator/pkg/controller"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/ready"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"k8s.io/apimachinery/pkg/util/json"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

var log = logf.Log.WithName("cmd")
var reconcile = flag.String("reconcile", "", "Reconcile the CloudConfig given as an argument")

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("operator-sdk Version: %v", sdkVersion.Version))
}

func main() {
	flag.Parse()

	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(logf.ZapLogger(false))

	printVersion()

	// -- CloudConfig reconciliation process

	if *reconcile != "" {
		spec := []byte(*reconcile)
		// parse the value as a JSON CloudConfigSpec
		var config v1alpha1.CloudConfigSpec
		if err := json.Unmarshal(spec, &config); err != nil {
			log.Error(err, "Could not unmarshal CloudConfigSpec config", "config", *reconcile)
			// Exit cleanly as we will otherwise fail the cron job causing an immediate retry
			os.Exit(0)
		}

		// Recover and report managed panics
		defer func() {
			if err := recover(); err != nil {
				switch t := err.(type) {
				case string:
					log.Info(err.(string))
					os.Exit(0)
				case error:
					log.Error(err.(error), "Unfettered panic!")
					os.Exit(1)
				default:
					panic(err)
				}
			}
		}()

		// reconcile the CloudConfigSpec
		config.Reconcile()
		os.Exit(0)
	}

	// -- Normal operator processing

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Error(err, "failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Become the leader before proceeding
	leader.Become(context.TODO(), "cloud-config-operator-lock")

	r := ready.NewFileReady()
	err = r.Set()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}
	defer r.Unset()

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{Namespace: namespace})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "manager exited non-zero")
		os.Exit(1)
	}
}
