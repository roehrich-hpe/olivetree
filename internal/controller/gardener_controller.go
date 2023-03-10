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

package controller

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	daosv1alpha1 "github.com/roehrich-hpe/olivetree/api/v1alpha1"
)

// GardenerReconciler reconciles a Gardener object
type GardenerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=daos.hpe.com,resources=gardeners,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=daos.hpe.com,resources=gardeners/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=daos.hpe.com,resources=gardeners/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=pods/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Gardener object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *GardenerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	gardener := &daosv1alpha1.Gardener{}
	if err := r.Get(ctx, req.NamespacedName, gardener); err != nil {
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Request", "cmd", gardener.Spec.Cmd)

	hostnames, err := r.getServerHostnames(ctx)
	if err != nil {
		log.Error(err, "unable to get server hostnames")
		return ctrl.Result{}, err
	}
	log.Info("Servers", "hostnames", hostnames)

	return ctrl.Result{}, nil
}

func (r *GardenerReconciler) getServerHostnames(ctx context.Context) ([]string, error) {
	// Look up the pods having the DAOS server label, get the IP address
	// from each one, and construct the DNS names.

	listOptions := []client.ListOption{
		client.InNamespace(daosv1alpha1.DAOSPodNamespace),
		client.MatchingLabels(map[string]string{
			daosv1alpha1.DAOSLabel: daosv1alpha1.DAOSServerLabel,
		}),
	}

	pods := &corev1.PodList{}
	if err := r.List(ctx, pods, listOptions...); err != nil {
		return nil, err
	}

	log := log.FromContext(ctx)
	hostnames := make([]string, 0)
	for _, pod := range pods.Items {
		log.Info("POD", "name", pod.GetName())
		hname := strings.ReplaceAll(pod.Status.PodIP, ".", "-") + "." + daosv1alpha1.DAOSPodNamespace + ".pod.cluster.local"
		hostnames = append(hostnames, hname)
	}

	return hostnames, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GardenerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&daosv1alpha1.Gardener{}).
		Complete(r)
}
