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
	"errors"
	"os/exec"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	daosv1alpha1 "github.com/roehrich-hpe/olivetree/api/v1alpha1"
)

const finalizer = "dmg.hpe.com"

// DmgReconciler reconciles a Dmg object
type DmgReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// We maintain a map of active operations which allows us to process cancel requests
	// This is a thread safe map since multiple dmg reconcilers and go routines will be executing at the same time.
	contexts sync.Map
}

// Keep track of the context and its cancel function so that we can track
// and cancel dmg operations in progress
type dmgCancelContext struct {
	ctx    context.Context
	cancel context.CancelFunc
}

//+kubebuilder:rbac:groups=daos.hpe.com,resources=dmgs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=daos.hpe.com,resources=dmgs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=daos.hpe.com,resources=dmgs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Dmg object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *DmgReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	dmg := &daosv1alpha1.Dmg{}
	if err := r.Get(ctx, req.NamespacedName, dmg); err != nil {
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !dmg.GetDeletionTimestamp().IsZero() {
		log.Info("deleting resource")
		if err := r.cancel(ctx, dmg); err != nil {
			return ctrl.Result{}, err
		}

		if controllerutil.ContainsFinalizer(dmg, finalizer) {
			controllerutil.RemoveFinalizer(dmg, finalizer)

			if err := r.Update(ctx, dmg); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(dmg, finalizer) {
		controllerutil.AddFinalizer(dmg, finalizer)
		if err := r.Update(ctx, dmg); err != nil {
			return ctrl.Result{}, err
		}

		// An update here will cause the reconciler to run again once
		// k8s has recorded the resource it its database.
		return ctrl.Result{}, nil
	}

	// Handle cancellation.
	if dmg.Spec.Cancel {
		if err := r.cancel(ctx, dmg); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	if dmg.Status.Status == daosv1alpha1.DmgConditionTypeFinished || dmg.Status.Status == daosv1alpha1.DmgConditionTypeRunning {
		return ctrl.Result{}, nil
	}

	// Expand the context with cancel and store it in the map so
	// the cancel function can be used in another reconciler loop.
	// Also add NamespacedName so we can retrieve the resource.
	ctxCancel, cancel := context.WithCancel(ctx)
	r.contexts.Store(dmg.Name, dmgCancelContext{
		ctx:    ctxCancel,
		cancel: cancel,
	})

	cmdStr, cmdArgs := getCmdAndArgs(dmg.Spec.Cmd)
	cmd := exec.CommandContext(ctxCancel, cmdStr, cmdArgs...)
	dmg.Status.Status = daosv1alpha1.DmgConditionTypeRunning
	log.Info("Running Command", "cmd", dmg.Spec.Cmd)

	if err := r.Status().Update(ctx, dmg); err != nil {
		log.Error(err, "unable to update status")
		return ctrl.Result{}, nil
	}

	go func() {
		// Start the command.
		combinedOutBuf, err := cmd.CombinedOutput()

		if errors.Is(ctxCancel.Err(), context.Canceled) {
			log.Error(err, "command operation cancelled", "output", string(combinedOutBuf))
		} else if err != nil {
			log.Error(err, "error from cmd", "cmd", dmg.Spec.Cmd, "output", string(combinedOutBuf))
		} else {
			log.Info("completed command", "cmd", dmg.Spec.Cmd, "output", string(combinedOutBuf))
		}

		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			dmg := &daosv1alpha1.Dmg{}
			if getErr := r.Get(ctx, req.NamespacedName, dmg); getErr != nil {
				return getErr
			}
			dmg.Status.Status = daosv1alpha1.DmgConditionTypeFinished
			dmg.Status.Output = string(combinedOutBuf)

			if err != nil {
				dmg.Status.ExitStatus = err.Error()
			}

			return r.Status().Update(ctx, dmg)
		})
		if retryErr != nil {
			log.Error(retryErr, "go-routine failed to update dmg status with completion")
		}

		r.contexts.Delete(dmg.Name)
	}()

	return ctrl.Result{}, nil
}

func getCmdAndArgs(cmdArgs string) (string, []string) {
	var cmd string
	var args []string

	if len(cmdArgs) > 0 {
		cmdList := strings.Split(cmdArgs, " ")
		cmd = cmdList[0]
		args = cmdList[1:]
	}

	return cmd, args
}

func (r *DmgReconciler) cancel(ctx context.Context, dmg *daosv1alpha1.Dmg) error {
	log := log.FromContext(ctx)

	storedCancelContext, found := r.contexts.LoadAndDelete(dmg.Name)
	if !found {
		return nil // Already completed or cancelled?
	}

	cancelContext := storedCancelContext.(dmgCancelContext)
	log.Info("Cancelling operation")
	cancelContext.cancel()
	<-cancelContext.ctx.Done()

	// Nothing more to do; the go routine that is executing the command
	// will exit and the status will be recorded at that time.
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DmgReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&daosv1alpha1.Dmg{}).
		Complete(r)
}
