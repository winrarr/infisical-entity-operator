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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	infisicalv1alpha1 "github.com/winrarr/infisical-entity-operator/api/infisical/v1alpha1"
	"github.com/winrarr/infisical-entity-operator/internal/infisicalclient"
)

// InfisicalOrganizationReconciler reconciles an Infisical top-level organization.
type InfisicalOrganizationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalorganizations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalorganizations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalorganizations/finalizers,verbs=update
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *InfisicalOrganizationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var organization infisicalv1alpha1.InfisicalOrganization
	if err := r.Get(ctx, req.NamespacedName, &organization); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if organization.DeletionTimestamp.IsZero() && deletionPolicy(organization.Spec.DeletionPolicy) == infisicalv1alpha1.DeletionPolicyDelete {
		if added, err := ensureFinalizer(ctx, r.Client, &organization); err != nil {
			return ctrl.Result{}, err
		} else if added {
			return ctrl.Result{}, nil
		}
	}
	if !organization.DeletionTimestamp.IsZero() {
		return r.reconcileOrganizationDeletion(ctx, &organization)
	}
	return r.reconcileOrganization(ctx, &organization)
}

func (r *InfisicalOrganizationReconciler) reconcileOrganization(ctx context.Context, organization *infisicalv1alpha1.InfisicalOrganization) (ctrl.Result, error) {
	before := organization.DeepCopy()
	if err := validateOrganizationSpec(organization); err != nil {
		return r.organizationError(ctx, organization, "ConfigurationInvalid", err)
	}

	apiClient, err := infisicalClientForConnection(ctx, r.Client, organization.Namespace, organization.Spec.ConnectionRef)
	if err != nil {
		return r.organizationError(ctx, organization, "ConnectionNotReady", err)
	}

	if organization.Status.OrganizationID == "" && canAdopt(organization.Spec.CreationPolicy) {
		adopted, findErr := apiClient.FindOrganization(ctx, organization.Spec.OrganizationID, organizationName(organization))
		if findErr != nil {
			return r.organizationError(ctx, organization, "ExternalReadFailed", findErr)
		}
		if adopted != nil {
			r.setOrganizationObservedState(organization, adopted)
		} else if !canCreate(organization.Spec.CreationPolicy) {
			return r.organizationError(ctx, organization, "RemoteOrganizationMissing", newDependencyError("the organization was not found in Infisical and creationPolicy is Adopt"))
		}
	}

	justCreated := false
	if organization.Status.OrganizationID == "" {
		if !canCreate(organization.Spec.CreationPolicy) {
			return r.organizationError(ctx, organization, "CreationNotAllowed", newDependencyError("organization was not found and creationPolicy is Adopt"))
		}
		created, createErr := apiClient.CreateOrganization(ctx, infisicalclient.CreateOrganizationRequest{Name: organizationName(organization)})
		if createErr != nil {
			return r.organizationError(ctx, organization, "ExternalCreateFailed", createErr)
		}
		r.setOrganizationObservedState(organization, created)
		justCreated = true
	}
	if justCreated {
		setCondition(&organization.Status.Conditions, organization.Generation, "True", "Ready", "Infisical organization is reconciled")
		return ctrl.Result{RequeueAfter: driftDetectionEvery}, persistStatus(ctx, r.Client, organization, before)
	}

	current, err := apiClient.FindOrganization(ctx, organization.Status.OrganizationID, "")
	if err != nil {
		return r.organizationError(ctx, organization, "ExternalReadFailed", err)
	}
	if current == nil {
		before := organization.DeepCopy()
		organization.Status.OrganizationID = ""
		organization.Status.OrganizationName = ""
		organization.Status.Slug = ""
		organization.Status.ObservedGeneration = organization.Generation
		setCondition(&organization.Status.Conditions, organization.Generation, "False", "RemoteOrganizationMissing", "the organization is not visible to the configured Infisical credential; it will be reacquired according to creationPolicy")
		return ctrl.Result{RequeueAfter: externalRetry}, persistStatus(ctx, r.Client, organization, before)
	}
	if organization.Spec.OrganizationID != "" && current.ID != organization.Spec.OrganizationID {
		return r.organizationError(ctx, organization, "ExternalOrganizationMismatch", fmt.Errorf("infisical organization ID %q does not match requested organizationID %q", current.ID, organization.Spec.OrganizationID))
	}
	r.setOrganizationObservedState(organization, current)
	setCondition(&organization.Status.Conditions, organization.Generation, "True", "Ready", "Infisical organization is reconciled")
	return ctrl.Result{RequeueAfter: driftDetectionEvery}, persistStatus(ctx, r.Client, organization, before)
}

func validateOrganizationSpec(organization *infisicalv1alpha1.InfisicalOrganization) error {
	if organization.Spec.OrganizationID != "" && creationPolicy(organization.Spec.CreationPolicy) == infisicalv1alpha1.CreationPolicyCreate {
		return fmt.Errorf("organizationID cannot be set with creationPolicy Create; use Adopt or CreateOrAdopt")
	}
	if organizationName(organization) == "" {
		return fmt.Errorf("organizationName or metadata.name is required")
	}
	return nil
}

func organizationName(organization *infisicalv1alpha1.InfisicalOrganization) string {
	if organization.Spec.OrganizationName != "" {
		return organization.Spec.OrganizationName
	}
	return organization.Name
}

func (r *InfisicalOrganizationReconciler) setOrganizationObservedState(organization *infisicalv1alpha1.InfisicalOrganization, observed *infisicalclient.Organization) {
	organization.Status.OrganizationID = observed.ID
	organization.Status.OrganizationName = observed.Name
	organization.Status.Slug = observed.Slug
	organization.Status.ObservedGeneration = organization.Generation
}

func (r *InfisicalOrganizationReconciler) organizationError(ctx context.Context, organization *infisicalv1alpha1.InfisicalOrganization, reason string, err error) (ctrl.Result, error) {
	before := organization.DeepCopy()
	organization.Status.ObservedGeneration = organization.Generation
	setCondition(&organization.Status.Conditions, organization.Generation, "False", reason, statusErrorMessage(err))
	return ctrl.Result{RequeueAfter: retryFor(err)}, persistStatus(ctx, r.Client, organization, before)
}

func (r *InfisicalOrganizationReconciler) reconcileOrganizationDeletion(ctx context.Context, organization *infisicalv1alpha1.InfisicalOrganization) (ctrl.Result, error) {
	if deletionPolicy(organization.Spec.DeletionPolicy) == infisicalv1alpha1.DeletionPolicyOrphan || organization.Status.OrganizationID == "" {
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, organization)
	}

	apiClient, err := infisicalClientForConnection(ctx, r.Client, organization.Namespace, organization.Spec.ConnectionRef)
	if err != nil {
		return ctrl.Result{RequeueAfter: retryFor(err)}, err
	}
	if err := apiClient.DeleteOrganization(ctx, organization.Status.OrganizationID); err != nil && !infisicalclient.IsNotFound(err) {
		return ctrl.Result{RequeueAfter: externalRetry}, err
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, organization)
}

func (r *InfisicalOrganizationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infisicalv1alpha1.InfisicalOrganization{}).
		Watches(&infisicalv1alpha1.InfisicalConnection{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var organizations infisicalv1alpha1.InfisicalOrganizationList
			if err := mgr.GetClient().List(ctx, &organizations, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range organizations.Items {
				organization := &organizations.Items[i]
				if organization.Spec.ConnectionRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(organization)})
				}
			}
			return requests
		})).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var organizations infisicalv1alpha1.InfisicalOrganizationList
			if err := mgr.GetClient().List(ctx, &organizations, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range organizations.Items {
				organization := &organizations.Items[i]
				var connection infisicalv1alpha1.InfisicalConnection
				if err := mgr.GetClient().Get(ctx, client.ObjectKey{Namespace: organization.Namespace, Name: organization.Spec.ConnectionRef.Name}, &connection); err == nil && connectionReferencesSecret(&connection, object.GetName()) {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(organization)})
				}
			}
			return requests
		})).
		Complete(r)
}
