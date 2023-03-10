/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	daosv1alpha1 "github.com/roehrich-hpe/olivetree/api/v1alpha1"
	"github.com/roehrich-hpe/olivetree/internal/controller"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(daosv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

const (
	nameNodeLocalController   = "node"
	nameSystemLevelController = "system"
)

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var controller string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&controller, "controller", "node", "The controller type to run (node, system)")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	oliveCtrl := newOliveControllerInitializer(controller)
	if oliveCtrl == nil {
		setupLog.Info("unsupported controller type", "controller", controller)
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       oliveCtrl.electionID() + ".hpe.com",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := oliveCtrl.setupReconcilers(mgr); err != nil {
		setupLog.Error(err, "unable to create olive controller reconciler", "controller", oliveCtrl.getType())
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// oliveControllerInitializer defines an interface to initialize one of the
// Olive Controller types.
type oliveControllerInitializer interface {
	getType() string
	setupReconcilers(manager.Manager) error
	electionID() string
}

// newOliveControllerInitializer creates a new Olive Controller Initializer
// for the given type, or nil if not found.
func newOliveControllerInitializer(typ string) oliveControllerInitializer {
	switch typ {
	case nameNodeLocalController:
		return &nodeLocalController{}
	case nameSystemLevelController:
		return &systemController{}
	}
	return nil
}

// nodeLocalController defines initializer for the per-node Olive Controller
type nodeLocalController struct{}

func (*nodeLocalController) getType() string    { return nameNodeLocalController }
func (*nodeLocalController) electionID() string { return "f2f5e1f0" }

func (c *nodeLocalController) setupReconcilers(mgr manager.Manager) error {
	err := (&controller.DmgReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)

	return err
}

// systemController defines initializer for the system-level Olive Controller
type systemController struct{}

func (*systemController) getType() string    { return nameSystemLevelController }
func (*systemController) electionID() string { return "f2f5e1f1" }

func (c *systemController) setupReconcilers(mgr manager.Manager) error {
	err := (&controller.GardenerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)

	return err
}
