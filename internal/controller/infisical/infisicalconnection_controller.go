/*
Copyright 2026 winrarr.

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

package infisical

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	infisicalv1alpha1 "github.com/winrarr/infisical-entity-operator/api/infisical/v1alpha1"
)

// InfisicalConnectionReconciler validates an Infisical API connection.
type InfisicalConnectionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalconnections,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalconnections/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *InfisicalConnectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var connection infisicalv1alpha1.InfisicalConnection
	if err := r.Get(ctx, req.NamespacedName, &connection); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	before := connection.Status
	apiClient, err := infisicalClientForConnection(ctx, r.Client, connection.Namespace, infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name})
	if err == nil {
		err = apiClient.Check(ctx)
	}
	if err != nil {
		connection.Status.ObservedGeneration = connection.Generation
		setCondition(&connection.Status.Conditions, connection.Generation, "False", "ConnectionUnavailable", statusErrorMessage(err))
		return ctrl.Result{RequeueAfter: retryFor(err)}, persistStatus(ctx, r.Client, &connection, before, connection.Status)
	}

	connection.Status.ObservedGeneration = connection.Generation
	setCondition(&connection.Status.Conditions, connection.Generation, "True", "Ready", "Infisical API connection is authenticated and reachable")
	return ctrl.Result{RequeueAfter: driftDetectionEvery}, persistStatus(ctx, r.Client, &connection, before, connection.Status)
}

func retryFor(err error) time.Duration {
	if isDependencyError(err) {
		return dependencyRetry
	}
	return externalRetry
}

func (r *InfisicalConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infisicalv1alpha1.InfisicalConnection{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var connections infisicalv1alpha1.InfisicalConnectionList
			if err := mgr.GetClient().List(ctx, &connections, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range connections.Items {
				connection := &connections.Items[i]
				if connection.Spec.AuthSecretRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(connection)})
				}
			}
			return requests
		})).
		Complete(r)
}
