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

// InfisicalIdentityReconciler reconciles a project-managed machine identity.
type InfisicalIdentityReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalidentities,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalidentities/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalidentities/finalizers,verbs=update
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalprojects,verbs=get;list;watch
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalprojectroles,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *InfisicalIdentityReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var identity infisicalv1alpha1.InfisicalIdentity
	if err := r.Get(ctx, req.NamespacedName, &identity); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if identity.DeletionTimestamp.IsZero() && deletionPolicy(identity.Spec.DeletionPolicy) == infisicalv1alpha1.DeletionPolicyDelete {
		if added, err := ensureFinalizer(ctx, r.Client, &identity); err != nil {
			return ctrl.Result{}, err
		} else if added {
			return ctrl.Result{}, nil
		}
	}
	if !identity.DeletionTimestamp.IsZero() {
		return r.reconcileIdentityDeletion(ctx, &identity)
	}
	before := identity.Status

	var project infisicalv1alpha1.InfisicalProject
	if err := r.Get(ctx, client.ObjectKey{Namespace: identity.Namespace, Name: identity.Spec.ProjectRef.Name}, &project); err != nil {
		if apierrors.IsNotFound(err) {
			return r.identityError(ctx, &identity, "ProjectNotReady", newDependencyError("InfisicalProject %s/%s was not found", identity.Namespace, identity.Spec.ProjectRef.Name))
		}
		return r.identityError(ctx, &identity, "ProjectReadFailed", err)
	}
	if project.Status.ProjectID == "" {
		return r.identityError(ctx, &identity, "ProjectNotReady", newDependencyError("InfisicalProject %s/%s has no observed Infisical project ID", project.Namespace, project.Name))
	}

	apiClient, err := infisicalClientForConnection(ctx, r.Client, identity.Namespace, identity.Spec.ConnectionRef)
	if err != nil {
		return r.identityError(ctx, &identity, "ConnectionNotReady", err)
	}

	if identity.Status.IdentityID == "" {
		if canAdopt(identity.Spec.CreationPolicy) {
			adopted, findErr := apiClient.FindIdentity(ctx, project.Status.ProjectID, identityName(&identity))
			if findErr != nil {
				return r.identityError(ctx, &identity, "ExternalReadFailed", findErr)
			}
			if adopted != nil {
				r.setIdentityObservedState(&identity, adopted, project.Status.ProjectID)
			}
		}
	}

	if identity.Status.IdentityID == "" {
		if !canCreate(identity.Spec.CreationPolicy) {
			return r.identityError(ctx, &identity, "CreationNotAllowed", newDependencyError("identity was not found and creationPolicy is Adopt"))
		}
		created, createErr := apiClient.CreateIdentity(ctx, project.Status.ProjectID, infisicalclient.CreateIdentityRequest{
			Name:                identityName(&identity),
			HasDeleteProtection: boolValue(identity.Spec.HasDeleteProtection, false),
			Metadata:            identityMetadataFrom(identity.Spec.Metadata),
		})
		if createErr != nil {
			return r.identityError(ctx, &identity, "ExternalCreateFailed", createErr)
		}
		r.setIdentityObservedState(&identity, created, project.Status.ProjectID)
	}

	current, err := apiClient.GetIdentity(ctx, project.Status.ProjectID, identity.Status.IdentityID)
	if err != nil {
		if infisicalclient.IsNotFound(err) {
			before := identity.Status
			identity.Status.IdentityID = ""
			identity.Status.MembershipID = ""
			identity.Status.Roles = nil
			identity.Status.ObservedGeneration = identity.Generation
			setCondition(&identity.Status.Conditions, identity.Generation, "False", "RemoteIdentityMissing", "the identity no longer exists in Infisical; it will be recreated according to creationPolicy")
			return ctrl.Result{RequeueAfter: externalRetry}, persistStatus(ctx, r.Client, &identity, before, identity.Status)
		}
		return r.identityError(ctx, &identity, "ExternalReadFailed", err)
	}

	if identityNeedsUpdate(&identity, current) {
		metadata := identityMetadataFrom(identity.Spec.Metadata)
		updated, updateErr := apiClient.UpdateIdentity(ctx, project.Status.ProjectID, current.ID, infisicalclient.IdentityPatch{
			Name:                identityName(&identity),
			HasDeleteProtection: identity.Spec.HasDeleteProtection,
			Metadata:            metadataPatch(identity.Spec.Metadata, metadata),
		})
		if updateErr != nil {
			return r.identityError(ctx, &identity, "ExternalUpdateFailed", updateErr)
		}
		current = updated
	}

	r.setIdentityObservedState(&identity, current, project.Status.ProjectID)
	roleSlugs, err := identityRoleSlugs(identity.Spec.RoleSlugs)
	if err != nil {
		return r.identityError(ctx, &identity, "RoleConfigurationInvalid", err)
	}
	if err := r.reconcileIdentityMembership(ctx, apiClient, &identity, project.Status.ProjectID, roleSlugs); err != nil {
		return r.identityError(ctx, &identity, "RoleMembershipReconcileFailed", err)
	}
	setCondition(&identity.Status.Conditions, identity.Generation, "True", "Ready", "Infisical identity is reconciled")
	return ctrl.Result{RequeueAfter: driftDetectionEvery}, persistStatus(ctx, r.Client, &identity, before, identity.Status)
}

func metadataPatch(want []infisicalv1alpha1.IdentityMetadata, metadata []infisicalclient.IdentityMetadata) *[]infisicalclient.IdentityMetadata {
	if want == nil {
		return nil
	}
	return &metadata
}

func identityNeedsUpdate(identity *infisicalv1alpha1.InfisicalIdentity, current *infisicalclient.Identity) bool {
	if current.Name != identityName(identity) {
		return true
	}
	if identity.Spec.HasDeleteProtection != nil && current.HasDeleteProtection != *identity.Spec.HasDeleteProtection {
		return true
	}
	return !identityMetadataEqual(identity.Spec.Metadata, current.Metadata)
}

func (r *InfisicalIdentityReconciler) setIdentityObservedState(identity *infisicalv1alpha1.InfisicalIdentity, observed *infisicalclient.Identity, projectID string) {
	identity.Status.IdentityID = observed.ID
	identity.Status.ProjectID = projectID
	identity.Status.ObservedGeneration = identity.Generation
}

func (r *InfisicalIdentityReconciler) reconcileIdentityMembership(ctx context.Context, apiClient *infisicalclient.Client, identity *infisicalv1alpha1.InfisicalIdentity, projectID string, roleSlugs []string) error {
	if len(roleSlugs) == 0 {
		identity.Status.MembershipID = ""
		identity.Status.Roles = nil
		return nil
	}

	membership, err := apiClient.GetIdentityMembership(ctx, projectID, identity.Status.IdentityID)
	if err != nil {
		if !infisicalclient.IsNotFound(err) {
			return fmt.Errorf("read identity project membership: %w", err)
		}
		if _, err := apiClient.CreateIdentityMembership(ctx, projectID, identity.Status.IdentityID, roleSlugs); err != nil {
			return fmt.Errorf("create identity project membership: %w", err)
		}
		membership, err = apiClient.GetIdentityMembership(ctx, projectID, identity.Status.IdentityID)
		if err != nil {
			return fmt.Errorf("read created identity project membership: %w", err)
		}
	}

	if !identityRoleSlugsEqual(roleSlugs, membership.Roles) {
		if _, err := apiClient.UpdateIdentityMembership(ctx, projectID, identity.Status.IdentityID, roleSlugs); err != nil {
			return fmt.Errorf("update identity project membership: %w", err)
		}
		membership, err = apiClient.GetIdentityMembership(ctx, projectID, identity.Status.IdentityID)
		if err != nil {
			return fmt.Errorf("read updated identity project membership: %w", err)
		}
	}

	identity.Status.MembershipID = membership.ID
	identity.Status.Roles = identityRoleStatusesFrom(membership.Roles)
	return nil
}

func (r *InfisicalIdentityReconciler) identityError(ctx context.Context, identity *infisicalv1alpha1.InfisicalIdentity, reason string, err error) (ctrl.Result, error) {
	before := identity.Status
	identity.Status.ObservedGeneration = identity.Generation
	setCondition(&identity.Status.Conditions, identity.Generation, "False", reason, statusErrorMessage(err))
	return ctrl.Result{RequeueAfter: retryFor(err)}, persistStatus(ctx, r.Client, identity, before, identity.Status)
}

func (r *InfisicalIdentityReconciler) reconcileIdentityDeletion(ctx context.Context, identity *infisicalv1alpha1.InfisicalIdentity) (ctrl.Result, error) {
	if deletionPolicy(identity.Spec.DeletionPolicy) == infisicalv1alpha1.DeletionPolicyOrphan || identity.Status.IdentityID == "" || identity.Status.ProjectID == "" {
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, identity)
	}

	apiClient, err := infisicalClientForConnection(ctx, r.Client, identity.Namespace, identity.Spec.ConnectionRef)
	if err != nil {
		return ctrl.Result{RequeueAfter: retryFor(err)}, err
	}
	if err := apiClient.DeleteIdentity(ctx, identity.Status.ProjectID, identity.Status.IdentityID); err != nil && !infisicalclient.IsNotFound(err) {
		return ctrl.Result{RequeueAfter: externalRetry}, err
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, identity)
}

func (r *InfisicalIdentityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infisicalv1alpha1.InfisicalIdentity{}).
		Watches(&infisicalv1alpha1.InfisicalConnection{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var identities infisicalv1alpha1.InfisicalIdentityList
			if err := mgr.GetClient().List(ctx, &identities, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range identities.Items {
				identity := &identities.Items[i]
				if identity.Spec.ConnectionRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(identity)})
				}
			}
			return requests
		})).
		Watches(&infisicalv1alpha1.InfisicalProject{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var identities infisicalv1alpha1.InfisicalIdentityList
			if err := mgr.GetClient().List(ctx, &identities, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range identities.Items {
				identity := &identities.Items[i]
				if identity.Spec.ProjectRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(identity)})
				}
			}
			return requests
		})).
		Watches(&infisicalv1alpha1.InfisicalProjectRole{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			role, ok := object.(*infisicalv1alpha1.InfisicalProjectRole)
			if !ok {
				return nil
			}
			var identities infisicalv1alpha1.InfisicalIdentityList
			if err := mgr.GetClient().List(ctx, &identities, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range identities.Items {
				identity := &identities.Items[i]
				if identity.Spec.ProjectRef.Name != role.Spec.ProjectRef.Name || !containsString(identity.Spec.RoleSlugs, projectRoleSlug(role)) {
					continue
				}
				requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(identity)})
			}
			return requests
		})).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var identities infisicalv1alpha1.InfisicalIdentityList
			if err := mgr.GetClient().List(ctx, &identities, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range identities.Items {
				identity := &identities.Items[i]
				var connection infisicalv1alpha1.InfisicalConnection
				if err := mgr.GetClient().Get(ctx, client.ObjectKey{Namespace: identity.Namespace, Name: identity.Spec.ConnectionRef.Name}, &connection); err == nil && connection.Spec.AuthSecretRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(identity)})
				}
			}
			return requests
		})).
		Complete(r)
}
