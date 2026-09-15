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
	"errors"
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

// InfisicalIdentityReconciler reconciles project- and organization-managed machine identities.
type InfisicalIdentityReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type identityConfigurationError struct {
	err error
}

func (e *identityConfigurationError) Error() string { return e.err.Error() }

func (e *identityConfigurationError) Unwrap() error { return e.err }

// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalidentities,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalidentities/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalidentities/finalizers,verbs=update
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalorganizations,verbs=get;list;watch
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
	return r.reconcileIdentity(ctx, &identity)
}

func (r *InfisicalIdentityReconciler) reconcileIdentity(ctx context.Context, identity *infisicalv1alpha1.InfisicalIdentity) (ctrl.Result, error) {
	before := identity.Status
	scope := identityScope(identity)
	if err := validateIdentitySpec(identity); err != nil {
		return r.identityError(ctx, identity, "ConfigurationInvalid", err)
	}

	anchor, err := r.identityAnchor(ctx, identity, scope)
	if err != nil {
		reason := "ProjectNotReady"
		if scope == infisicalv1alpha1.IdentityScopeOrganization {
			reason = "OrganizationNotReady"
		}
		var configurationErr *identityConfigurationError
		if errors.As(err, &configurationErr) {
			reason = "ConfigurationInvalid"
		}
		return r.identityError(ctx, identity, reason, err)
	}

	apiClient, err := infisicalClientForConnection(ctx, r.Client, identity.Namespace, identity.Spec.ConnectionRef)
	if err != nil {
		return r.identityError(ctx, identity, "ConnectionNotReady", err)
	}

	if identity.Status.IdentityID == "" && canAdopt(identity.Spec.CreationPolicy) {
		var adopted *infisicalclient.Identity
		if scope == infisicalv1alpha1.IdentityScopeOrganization {
			adopted, err = apiClient.FindOrganizationIdentity(ctx, anchor.organizationID, identityName(identity))
		} else {
			adopted, err = apiClient.FindIdentity(ctx, anchor.projectID, identityName(identity))
		}
		if err != nil {
			return r.identityError(ctx, identity, "ExternalReadFailed", err)
		}
		if adopted != nil {
			if err := validateObservedIdentity(adopted, anchor, scope); err != nil {
				return r.identityError(ctx, identity, "ExternalIdentityMismatch", err)
			}
			r.setIdentityObservedState(identity, adopted, anchor, scope)
		}
	}

	if identity.Status.IdentityID == "" {
		if !canCreate(identity.Spec.CreationPolicy) {
			return r.identityError(ctx, identity, "CreationNotAllowed", newDependencyError("identity was not found and creationPolicy is Adopt"))
		}

		var created *infisicalclient.Identity
		if scope == infisicalv1alpha1.IdentityScopeOrganization {
			created, err = apiClient.CreateOrganizationIdentity(ctx, anchor.organizationID, infisicalclient.CreateOrganizationIdentityRequest{
				Name:                identityName(identity),
				OrganizationID:      anchor.organizationID,
				Role:                organizationRole(identity),
				HasDeleteProtection: boolValue(identity.Spec.HasDeleteProtection, false),
				Metadata:            identityMetadataFrom(identity.Spec.Metadata),
			})
		} else {
			created, err = apiClient.CreateIdentity(ctx, anchor.projectID, infisicalclient.CreateIdentityRequest{
				Name:                identityName(identity),
				HasDeleteProtection: boolValue(identity.Spec.HasDeleteProtection, false),
				Metadata:            identityMetadataFrom(identity.Spec.Metadata),
			})
		}
		if err != nil {
			return r.identityError(ctx, identity, "ExternalCreateFailed", err)
		}
		if err := validateObservedIdentity(created, anchor, scope); err != nil {
			return r.identityError(ctx, identity, "ExternalIdentityMismatch", err)
		}
		r.setIdentityObservedState(identity, created, anchor, scope)
	}

	var current *infisicalclient.Identity
	if scope == infisicalv1alpha1.IdentityScopeOrganization {
		current, err = apiClient.GetOrganizationIdentity(ctx, identity.Status.IdentityID)
	} else {
		current, err = apiClient.GetIdentity(ctx, anchor.projectID, identity.Status.IdentityID)
	}
	if err != nil {
		if infisicalclient.IsNotFound(err) {
			before := identity.Status
			identity.Status.IdentityID = ""
			identity.Status.ProjectID = ""
			identity.Status.OrganizationID = ""
			identity.Status.OrganizationRole = ""
			identity.Status.MembershipID = ""
			identity.Status.Roles = nil
			identity.Status.ProjectMemberships = nil
			identity.Status.ObservedGeneration = identity.Generation
			setCondition(&identity.Status.Conditions, identity.Generation, "False", "RemoteIdentityMissing", "the identity no longer exists in Infisical; it will be recreated according to creationPolicy")
			return ctrl.Result{RequeueAfter: externalRetry}, persistStatus(ctx, r.Client, identity, before, identity.Status)
		}
		return r.identityError(ctx, identity, "ExternalReadFailed", err)
	}
	if err := validateObservedIdentity(current, anchor, scope); err != nil {
		return r.identityError(ctx, identity, "ExternalIdentityMismatch", err)
	}

	if identityNeedsUpdate(identity, current, scope) {
		metadata := identityMetadataFrom(identity.Spec.Metadata)
		patch := infisicalclient.IdentityPatch{
			Name:                identityName(identity),
			HasDeleteProtection: identity.Spec.HasDeleteProtection,
			Metadata:            metadataPatch(identity.Spec.Metadata, metadata),
		}
		if scope == infisicalv1alpha1.IdentityScopeOrganization {
			role := organizationRole(identity)
			patch.Role = &role
			current, err = apiClient.UpdateOrganizationIdentity(ctx, current.ID, patch)
		} else {
			current, err = apiClient.UpdateIdentity(ctx, anchor.projectID, current.ID, patch)
		}
		if err != nil {
			return r.identityError(ctx, identity, "ExternalUpdateFailed", err)
		}
	}

	r.setIdentityObservedState(identity, current, anchor, scope)
	if scope == infisicalv1alpha1.IdentityScopeOrganization {
		if err := r.reconcileOrganizationProjectMemberships(ctx, apiClient, identity); err != nil {
			return r.identityError(ctx, identity, "RoleMembershipReconcileFailed", err)
		}
	} else {
		roleSlugs, err := identityRoleSlugs(identity.Spec.RoleSlugs)
		if err != nil {
			return r.identityError(ctx, identity, "RoleConfigurationInvalid", err)
		}
		membership, err := r.reconcileIdentityMembership(ctx, apiClient, identity, anchor.projectID, roleSlugs)
		if err != nil {
			return r.identityError(ctx, identity, "RoleMembershipReconcileFailed", err)
		}
		if membership == nil {
			identity.Status.MembershipID = ""
			identity.Status.Roles = nil
		} else {
			identity.Status.MembershipID = membership.MembershipID
			identity.Status.Roles = membership.Roles
		}
	}
	setCondition(&identity.Status.Conditions, identity.Generation, "True", "Ready", "Infisical identity is reconciled")
	return ctrl.Result{RequeueAfter: driftDetectionEvery}, persistStatus(ctx, r.Client, identity, before, identity.Status)
}

type identityAnchor struct {
	projectID      string
	organizationID string
}

func (r *InfisicalIdentityReconciler) identityAnchor(ctx context.Context, identity *infisicalv1alpha1.InfisicalIdentity, scope infisicalv1alpha1.IdentityScope) (*identityAnchor, error) {
	if scope == infisicalv1alpha1.IdentityScopeOrganization {
		var organization infisicalv1alpha1.InfisicalOrganization
		if err := r.Get(ctx, client.ObjectKey{Namespace: identity.Namespace, Name: identity.Spec.OrganizationRef.Name}, &organization); err != nil {
			if apierrors.IsNotFound(err) {
				return nil, newDependencyError("InfisicalOrganization %s/%s was not found", identity.Namespace, identity.Spec.OrganizationRef.Name)
			}
			return nil, err
		}
		if organization.Status.OrganizationID == "" {
			return nil, newDependencyError("InfisicalOrganization %s/%s has no observed Infisical organization ID", organization.Namespace, organization.Name)
		}

		anchor := &identityAnchor{organizationID: organization.Status.OrganizationID}
		for _, binding := range identity.Spec.ProjectRoleBindings {
			var boundProject infisicalv1alpha1.InfisicalProject
			if err := r.Get(ctx, client.ObjectKey{Namespace: identity.Namespace, Name: binding.ProjectRef.Name}, &boundProject); err != nil {
				if apierrors.IsNotFound(err) {
					return nil, newDependencyError("InfisicalProject %s/%s was not found", identity.Namespace, binding.ProjectRef.Name)
				}
				return nil, err
			}
			if boundProject.Status.ProjectID == "" {
				return nil, newDependencyError("InfisicalProject %s/%s has no observed Infisical project ID", boundProject.Namespace, boundProject.Name)
			}
			if boundProject.Status.OrganizationID != anchor.organizationID {
				return nil, &identityConfigurationError{err: fmt.Errorf("projectRoleBindings project %q belongs to organization %q, want %q", binding.ProjectRef.Name, boundProject.Status.OrganizationID, anchor.organizationID)}
			}
		}
		return anchor, nil
	}

	projectName := identity.Spec.ProjectRef.Name
	var project infisicalv1alpha1.InfisicalProject
	if err := r.Get(ctx, client.ObjectKey{Namespace: identity.Namespace, Name: projectName}, &project); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, newDependencyError("InfisicalProject %s/%s was not found", identity.Namespace, projectName)
		}
		return nil, err
	}
	if project.Status.ProjectID == "" {
		return nil, newDependencyError("InfisicalProject %s/%s has no observed Infisical project ID", project.Namespace, project.Name)
	}
	return &identityAnchor{projectID: project.Status.ProjectID}, nil
}

func validateIdentitySpec(identity *infisicalv1alpha1.InfisicalIdentity) error {
	scope := identityScope(identity)
	switch scope {
	case infisicalv1alpha1.IdentityScopeProject:
		if identity.Spec.ProjectRef == nil || identity.Spec.ProjectRef.Name == "" {
			return fmt.Errorf("projectRef is required for Project scope")
		}
		if identity.Spec.OrganizationRef != nil {
			return fmt.Errorf("organizationRef is only valid for Organization scope")
		}
		if identity.Spec.OrganizationRole != "" {
			return fmt.Errorf("organizationRole is only valid for Organization scope")
		}
		if len(identity.Spec.ProjectRoleBindings) != 0 {
			return fmt.Errorf("projectRoleBindings is only valid for Organization scope")
		}
	case infisicalv1alpha1.IdentityScopeOrganization:
		if identity.Spec.OrganizationRef == nil || identity.Spec.OrganizationRef.Name == "" {
			return fmt.Errorf("organizationRef is required for Organization scope")
		}
		if identity.Spec.ProjectRef != nil && identity.Spec.ProjectRef.Name != "" {
			return fmt.Errorf("projectRef is only valid for Project scope")
		}
		seenProjects := make(map[string]struct{}, len(identity.Spec.ProjectRoleBindings))
		for _, binding := range identity.Spec.ProjectRoleBindings {
			if binding.ProjectRef.Name == "" {
				return fmt.Errorf("projectRoleBindings cannot contain an empty projectRef")
			}
			if _, exists := seenProjects[binding.ProjectRef.Name]; exists {
				return fmt.Errorf("projectRoleBindings contains duplicate projectRef %q", binding.ProjectRef.Name)
			}
			seenProjects[binding.ProjectRef.Name] = struct{}{}
			if _, err := identityRoleSlugs(binding.RoleSlugs); err != nil {
				return fmt.Errorf("projectRoleBindings project %q: %w", binding.ProjectRef.Name, err)
			}
		}
	default:
		return fmt.Errorf("scope %q is unsupported", identity.Spec.Scope)
	}
	return nil
}

func identityScope(identity *infisicalv1alpha1.InfisicalIdentity) infisicalv1alpha1.IdentityScope {
	if identity.Spec.Scope == "" {
		return infisicalv1alpha1.IdentityScopeProject
	}
	return identity.Spec.Scope
}

func organizationRole(identity *infisicalv1alpha1.InfisicalIdentity) string {
	if identity.Spec.OrganizationRole == "" {
		return "no-access"
	}
	return identity.Spec.OrganizationRole
}

func validateObservedIdentity(observed *infisicalclient.Identity, anchor *identityAnchor, scope infisicalv1alpha1.IdentityScope) error {
	if scope == infisicalv1alpha1.IdentityScopeOrganization {
		if observed.OrganizationID != "" && observed.OrganizationID != anchor.organizationID {
			return fmt.Errorf("infisical identity belongs to organization %q, want %q", observed.OrganizationID, anchor.organizationID)
		}
		return nil
	}
	if observed.ProjectID != "" && observed.ProjectID != anchor.projectID {
		return fmt.Errorf("infisical identity belongs to project %q, want %q", observed.ProjectID, anchor.projectID)
	}
	return nil
}

func metadataPatch(want []infisicalv1alpha1.IdentityMetadata, metadata []infisicalclient.IdentityMetadata) *[]infisicalclient.IdentityMetadata {
	if want == nil {
		return nil
	}
	return &metadata
}

func identityNeedsUpdate(identity *infisicalv1alpha1.InfisicalIdentity, current *infisicalclient.Identity, scope infisicalv1alpha1.IdentityScope) bool {
	if current.Name != identityName(identity) {
		return true
	}
	if identity.Spec.HasDeleteProtection != nil && current.HasDeleteProtection != *identity.Spec.HasDeleteProtection {
		return true
	}
	if scope == infisicalv1alpha1.IdentityScopeOrganization && current.OrganizationRole != organizationRole(identity) {
		return true
	}
	return !identityMetadataEqual(identity.Spec.Metadata, current.Metadata)
}

func (r *InfisicalIdentityReconciler) setIdentityObservedState(identity *infisicalv1alpha1.InfisicalIdentity, observed *infisicalclient.Identity, anchor *identityAnchor, scope infisicalv1alpha1.IdentityScope) {
	identity.Status.IdentityID = observed.ID
	identity.Status.ObservedGeneration = identity.Generation
	if scope == infisicalv1alpha1.IdentityScopeOrganization {
		identity.Status.ProjectID = ""
		identity.Status.OrganizationID = anchor.organizationID
		identity.Status.OrganizationRole = observed.OrganizationRole
		if identity.Status.OrganizationRole == "" {
			identity.Status.OrganizationRole = organizationRole(identity)
		}
		identity.Status.MembershipID = ""
		identity.Status.Roles = nil
		return
	}
	identity.Status.ProjectID = anchor.projectID
	identity.Status.OrganizationID = ""
	identity.Status.OrganizationRole = ""
	identity.Status.ProjectMemberships = nil
}

func (r *InfisicalIdentityReconciler) reconcileOrganizationProjectMemberships(ctx context.Context, apiClient *infisicalclient.Client, identity *infisicalv1alpha1.InfisicalIdentity) error {
	identity.Status.ProjectMemberships = nil
	for _, binding := range identity.Spec.ProjectRoleBindings {
		project := &infisicalv1alpha1.InfisicalProject{}
		if err := r.Get(ctx, client.ObjectKey{Namespace: identity.Namespace, Name: binding.ProjectRef.Name}, project); err != nil {
			return fmt.Errorf("read project %q for identity membership: %w", binding.ProjectRef.Name, err)
		}
		roleSlugs, err := identityRoleSlugs(binding.RoleSlugs)
		if err != nil {
			return fmt.Errorf("projectRoleBindings project %q: %w", binding.ProjectRef.Name, err)
		}
		membership, err := r.reconcileIdentityMembership(ctx, apiClient, identity, project.Status.ProjectID, roleSlugs)
		if err != nil {
			return fmt.Errorf("project %q: %w", binding.ProjectRef.Name, err)
		}
		if membership == nil {
			continue
		}
		identity.Status.ProjectMemberships = append(identity.Status.ProjectMemberships, *membership)
	}
	return nil
}

func (r *InfisicalIdentityReconciler) reconcileIdentityMembership(ctx context.Context, apiClient *infisicalclient.Client, identity *infisicalv1alpha1.InfisicalIdentity, projectID string, roleSlugs []string) (*infisicalv1alpha1.IdentityProjectMembershipStatus, error) {
	if len(roleSlugs) == 0 {
		return nil, nil
	}

	membership, err := apiClient.GetIdentityMembership(ctx, projectID, identity.Status.IdentityID)
	if err != nil {
		if !infisicalclient.IsNotFound(err) {
			return nil, fmt.Errorf("read identity project membership: %w", err)
		}
		if _, err := apiClient.CreateIdentityMembership(ctx, projectID, identity.Status.IdentityID, roleSlugs); err != nil {
			return nil, fmt.Errorf("create identity project membership: %w", err)
		}
		membership, err = apiClient.GetIdentityMembership(ctx, projectID, identity.Status.IdentityID)
		if err != nil {
			return nil, fmt.Errorf("read created identity project membership: %w", err)
		}
	}

	if !identityRoleSlugsEqual(roleSlugs, membership.Roles) {
		if _, err := apiClient.UpdateIdentityMembership(ctx, projectID, identity.Status.IdentityID, roleSlugs); err != nil {
			return nil, fmt.Errorf("update identity project membership: %w", err)
		}
		membership, err = apiClient.GetIdentityMembership(ctx, projectID, identity.Status.IdentityID)
		if err != nil {
			return nil, fmt.Errorf("read updated identity project membership: %w", err)
		}
	}

	projectIDObserved := membership.ProjectID
	if projectIDObserved == "" {
		projectIDObserved = projectID
	}
	return &infisicalv1alpha1.IdentityProjectMembershipStatus{
		ProjectID:    projectIDObserved,
		MembershipID: membership.ID,
		Roles:        identityRoleStatusesFrom(membership.Roles),
	}, nil
}

func (r *InfisicalIdentityReconciler) identityError(ctx context.Context, identity *infisicalv1alpha1.InfisicalIdentity, reason string, err error) (ctrl.Result, error) {
	before := identity.Status
	identity.Status.ObservedGeneration = identity.Generation
	setCondition(&identity.Status.Conditions, identity.Generation, "False", reason, statusErrorMessage(err))
	return ctrl.Result{RequeueAfter: retryFor(err)}, persistStatus(ctx, r.Client, identity, before, identity.Status)
}

func (r *InfisicalIdentityReconciler) reconcileIdentityDeletion(ctx context.Context, identity *infisicalv1alpha1.InfisicalIdentity) (ctrl.Result, error) {
	if deletionPolicy(identity.Spec.DeletionPolicy) == infisicalv1alpha1.DeletionPolicyOrphan || identity.Status.IdentityID == "" {
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, identity)
	}

	apiClient, err := infisicalClientForConnection(ctx, r.Client, identity.Namespace, identity.Spec.ConnectionRef)
	if err != nil {
		return ctrl.Result{RequeueAfter: retryFor(err)}, err
	}

	var deleteErr error
	if identityScope(identity) == infisicalv1alpha1.IdentityScopeOrganization {
		deleteErr = apiClient.DeleteOrganizationIdentity(ctx, identity.Status.IdentityID)
	} else if identity.Status.ProjectID != "" {
		deleteErr = apiClient.DeleteIdentity(ctx, identity.Status.ProjectID, identity.Status.IdentityID)
	}
	if deleteErr != nil && !infisicalclient.IsNotFound(deleteErr) {
		return ctrl.Result{RequeueAfter: externalRetry}, deleteErr
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, identity)
}

func identityReferencesProject(identity *infisicalv1alpha1.InfisicalIdentity, projectName string) bool {
	if identity.Spec.ProjectRef != nil && identity.Spec.ProjectRef.Name == projectName {
		return true
	}
	for _, binding := range identity.Spec.ProjectRoleBindings {
		if binding.ProjectRef.Name == projectName {
			return true
		}
	}
	return false
}

func identityReferencesOrganization(identity *infisicalv1alpha1.InfisicalIdentity, organizationName string) bool {
	return identity.Spec.OrganizationRef != nil && identity.Spec.OrganizationRef.Name == organizationName
}

func identityUsesRole(identity *infisicalv1alpha1.InfisicalIdentity, projectName, roleSlug string) bool {
	if identityScope(identity) == infisicalv1alpha1.IdentityScopeOrganization {
		for _, binding := range identity.Spec.ProjectRoleBindings {
			if binding.ProjectRef.Name == projectName && containsString(binding.RoleSlugs, roleSlug) {
				return true
			}
		}
		return false
	}
	return identity.Spec.ProjectRef != nil && identity.Spec.ProjectRef.Name == projectName && containsString(identity.Spec.RoleSlugs, roleSlug)
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
				if identityReferencesProject(identity, object.GetName()) {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(identity)})
				}
			}
			return requests
		})).
		Watches(&infisicalv1alpha1.InfisicalOrganization{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var identities infisicalv1alpha1.InfisicalIdentityList
			if err := mgr.GetClient().List(ctx, &identities, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range identities.Items {
				identity := &identities.Items[i]
				if identityReferencesOrganization(identity, object.GetName()) {
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
				if !identityUsesRole(identity, role.Spec.ProjectRef.Name, projectRoleSlug(role)) {
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
