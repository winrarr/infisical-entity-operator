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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	infisicalv1alpha1 "github.com/winrarr/infisical-entity-operator/api/infisical/v1alpha1"
	"github.com/winrarr/infisical-entity-operator/internal/infisicalclient"
)

// InfisicalEnvironmentReconciler reconciles an Infisical project environment.
type InfisicalEnvironmentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalenvironments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalenvironments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalenvironments/finalizers,verbs=update
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalprojects,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *InfisicalEnvironmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var environment infisicalv1alpha1.InfisicalEnvironment
	if err := r.Get(ctx, req.NamespacedName, &environment); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if environment.DeletionTimestamp.IsZero() && deletionPolicy(environment.Spec.DeletionPolicy) == infisicalv1alpha1.DeletionPolicyDelete {
		if added, err := ensureFinalizer(ctx, r.Client, &environment); err != nil {
			return ctrl.Result{}, err
		} else if added {
			return ctrl.Result{}, nil
		}
	}
	if !environment.DeletionTimestamp.IsZero() {
		return r.reconcileEnvironmentDeletion(ctx, &environment)
	}
	before := environment.Status

	var project infisicalv1alpha1.InfisicalProject
	if err := r.Get(ctx, client.ObjectKey{Namespace: environment.Namespace, Name: environment.Spec.ProjectRef.Name}, &project); err != nil {
		if apierrors.IsNotFound(err) {
			return r.environmentError(ctx, &environment, "ProjectNotReady", newDependencyError("InfisicalProject %s/%s was not found", environment.Namespace, environment.Spec.ProjectRef.Name))
		}
		return r.environmentError(ctx, &environment, "ProjectReadFailed", err)
	}
	if project.Status.ProjectID == "" {
		return r.environmentError(ctx, &environment, "ProjectNotReady", newDependencyError("InfisicalProject %s/%s has no observed Infisical project ID", project.Namespace, project.Name))
	}

	apiClient, err := infisicalClientForConnection(ctx, r.Client, environment.Namespace, environment.Spec.ConnectionRef)
	if err != nil {
		return r.environmentError(ctx, &environment, "ConnectionNotReady", err)
	}

	if environment.Status.EnvironmentID == "" && canAdopt(environment.Spec.CreationPolicy) {
		adopted, findErr := apiClient.GetEnvironmentBySlug(ctx, project.Status.ProjectID, environmentSlug(&environment))
		if findErr != nil && !infisicalclient.IsNotFound(findErr) {
			return r.environmentError(ctx, &environment, "ExternalReadFailed", findErr)
		}
		if adopted != nil {
			adopted, err = restoreEnvironmentIfDeleted(ctx, apiClient, project.Status.ProjectID, adopted)
			if err != nil {
				return r.environmentError(ctx, &environment, "ExternalRestoreFailed", err)
			}
			r.setEnvironmentObservedState(&environment, adopted, project.Status.ProjectID)
		}
	}

	if environment.Status.EnvironmentID == "" {
		if !canCreate(environment.Spec.CreationPolicy) {
			return r.environmentError(ctx, &environment, "CreationNotAllowed", newDependencyError("environment was not found and creationPolicy is Adopt"))
		}
		created, createErr := apiClient.CreateEnvironment(ctx, project.Status.ProjectID, infisicalclient.CreateEnvironmentRequest{
			Name:     environmentName(&environment),
			Slug:     environmentSlug(&environment),
			Position: environment.Spec.Position,
		})
		if createErr != nil {
			return r.environmentError(ctx, &environment, "ExternalCreateFailed", createErr)
		}
		r.setEnvironmentObservedState(&environment, created, project.Status.ProjectID)
	}

	current, err := apiClient.GetEnvironmentByID(ctx, project.Status.ProjectID, environment.Status.EnvironmentID)
	if err != nil {
		if infisicalclient.IsNotFound(err) {
			before := environment.Status
			environment.Status.EnvironmentID = ""
			environment.Status.ObservedGeneration = environment.Generation
			setCondition(&environment.Status.Conditions, environment.Generation, "False", "RemoteEnvironmentMissing", "the environment no longer exists in Infisical; it will be recreated according to creationPolicy")
			return ctrl.Result{RequeueAfter: externalRetry}, persistStatus(ctx, r.Client, &environment, before, environment.Status)
		}
		return r.environmentError(ctx, &environment, "ExternalReadFailed", err)
	}
	if current.SoftDeletedAt != nil {
		current, err = apiClient.RestoreEnvironment(ctx, project.Status.ProjectID, current.ID)
		if err != nil {
			return r.environmentError(ctx, &environment, "ExternalRestoreFailed", err)
		}
	}

	if environmentNeedsUpdate(&environment, current) {
		updated, updateErr := apiClient.UpdateEnvironment(ctx, project.Status.ProjectID, current.ID, infisicalclient.EnvironmentPatch{
			Name:     environmentName(&environment),
			Position: environment.Spec.Position,
		})
		if updateErr != nil {
			return r.environmentError(ctx, &environment, "ExternalUpdateFailed", updateErr)
		}
		current = updated
	}

	r.setEnvironmentObservedState(&environment, current, project.Status.ProjectID)
	setCondition(&environment.Status.Conditions, environment.Generation, "True", "Ready", "Infisical environment is reconciled")
	return ctrl.Result{RequeueAfter: driftDetectionEvery}, persistStatus(ctx, r.Client, &environment, before, environment.Status)
}

func restoreEnvironmentIfDeleted(ctx context.Context, apiClient *infisicalclient.Client, projectID string, environment *infisicalclient.Environment) (*infisicalclient.Environment, error) {
	if environment == nil || environment.SoftDeletedAt == nil {
		return environment, nil
	}
	return apiClient.RestoreEnvironment(ctx, projectID, environment.ID)
}

func environmentNeedsUpdate(environment *infisicalv1alpha1.InfisicalEnvironment, current *infisicalclient.Environment) bool {
	if current.Name != environmentName(environment) {
		return true
	}
	return environment.Spec.Position != nil && current.Position != *environment.Spec.Position
}

func (r *InfisicalEnvironmentReconciler) setEnvironmentObservedState(environment *infisicalv1alpha1.InfisicalEnvironment, observed *infisicalclient.Environment, projectID string) {
	environment.Status.EnvironmentID = observed.ID
	environment.Status.ProjectID = projectID
	environment.Status.Name = observed.Name
	environment.Status.Slug = observed.Slug
	environment.Status.Position = observed.Position
	environment.Status.ObservedGeneration = environment.Generation
}

func (r *InfisicalEnvironmentReconciler) environmentError(ctx context.Context, environment *infisicalv1alpha1.InfisicalEnvironment, reason string, err error) (ctrl.Result, error) {
	before := environment.Status
	environment.Status.ObservedGeneration = environment.Generation
	setCondition(&environment.Status.Conditions, environment.Generation, "False", reason, statusErrorMessage(err))
	return ctrl.Result{RequeueAfter: retryFor(err)}, persistStatus(ctx, r.Client, environment, before, environment.Status)
}

func (r *InfisicalEnvironmentReconciler) reconcileEnvironmentDeletion(ctx context.Context, environment *infisicalv1alpha1.InfisicalEnvironment) (ctrl.Result, error) {
	if deletionPolicy(environment.Spec.DeletionPolicy) == infisicalv1alpha1.DeletionPolicyOrphan || environment.Status.EnvironmentID == "" || environment.Status.ProjectID == "" {
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, environment)
	}

	apiClient, err := infisicalClientForConnection(ctx, r.Client, environment.Namespace, environment.Spec.ConnectionRef)
	if err != nil {
		return ctrl.Result{RequeueAfter: retryFor(err)}, err
	}
	if err := apiClient.DeleteEnvironment(ctx, environment.Status.ProjectID, environment.Status.EnvironmentID); err != nil && !infisicalclient.IsNotFound(err) {
		return ctrl.Result{RequeueAfter: externalRetry}, err
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, environment)
}

func (r *InfisicalEnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infisicalv1alpha1.InfisicalEnvironment{}).
		Watches(&infisicalv1alpha1.InfisicalConnection{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var environments infisicalv1alpha1.InfisicalEnvironmentList
			if err := mgr.GetClient().List(ctx, &environments, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range environments.Items {
				environment := &environments.Items[i]
				if environment.Spec.ConnectionRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(environment)})
				}
			}
			return requests
		})).
		Watches(&infisicalv1alpha1.InfisicalProject{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var environments infisicalv1alpha1.InfisicalEnvironmentList
			if err := mgr.GetClient().List(ctx, &environments, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range environments.Items {
				environment := &environments.Items[i]
				if environment.Spec.ProjectRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(environment)})
				}
			}
			return requests
		})).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var environments infisicalv1alpha1.InfisicalEnvironmentList
			if err := mgr.GetClient().List(ctx, &environments, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range environments.Items {
				environment := &environments.Items[i]
				var connection infisicalv1alpha1.InfisicalConnection
				if err := mgr.GetClient().Get(ctx, client.ObjectKey{Namespace: environment.Namespace, Name: environment.Spec.ConnectionRef.Name}, &connection); err == nil && connection.Spec.AuthSecretRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(environment)})
				}
			}
			return requests
		})).
		Complete(r)
}
