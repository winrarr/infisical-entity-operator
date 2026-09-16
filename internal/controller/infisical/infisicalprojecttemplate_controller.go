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
	"reflect"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	infisicalv1alpha1 "github.com/winrarr/infisical-entity-operator/api/infisical/v1alpha1"
	"github.com/winrarr/infisical-entity-operator/internal/infisicalclient"
)

// InfisicalProjectTemplateReconciler reconciles an Infisical project template.
type InfisicalProjectTemplateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalprojecttemplates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalprojecttemplates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalprojecttemplates/finalizers,verbs=update
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalorganizations,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *InfisicalProjectTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var template infisicalv1alpha1.InfisicalProjectTemplate
	if err := r.Get(ctx, req.NamespacedName, &template); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if template.DeletionTimestamp.IsZero() && deletionPolicy(template.Spec.DeletionPolicy) == infisicalv1alpha1.DeletionPolicyDelete {
		if added, err := ensureFinalizer(ctx, r.Client, &template); err != nil {
			return ctrl.Result{}, err
		} else if added {
			return ctrl.Result{}, nil
		}
	}
	if !template.DeletionTimestamp.IsZero() {
		return r.reconcileProjectTemplateDeletion(ctx, &template)
	}
	before := template.DeepCopy()
	expectedOrganizationID, err := r.expectedProjectTemplateOrganizationID(ctx, &template)
	if err != nil {
		return r.projectTemplateError(ctx, &template, "OrganizationNotReady", err)
	}

	apiClient, err := infisicalClientForConnection(ctx, r.Client, template.Namespace, template.Spec.ConnectionRef)
	if err != nil {
		return r.projectTemplateError(ctx, &template, "ConnectionNotReady", err)
	}

	if template.Status.TemplateID == "" && canAdopt(template.Spec.CreationPolicy) {
		adopted, findErr := apiClient.FindProjectTemplate(ctx, projectTemplateName(&template))
		if findErr != nil {
			return r.projectTemplateError(ctx, &template, "ExternalReadFailed", findErr)
		}
		if adopted != nil {
			if err := validateProjectTemplateOrganization(adopted, expectedOrganizationID); err != nil {
				return r.projectTemplateError(ctx, &template, "ExternalTemplateMismatch", err)
			}
			r.setProjectTemplateObservedState(&template, adopted)
		}
	}

	if template.Status.TemplateID == "" {
		if !canCreate(template.Spec.CreationPolicy) {
			return r.projectTemplateError(ctx, &template, "CreationNotAllowed", newDependencyError("project template was not found and creationPolicy is Adopt"))
		}
		created, createErr := apiClient.CreateProjectTemplate(ctx, projectTemplateRequestFrom(&template))
		if createErr != nil {
			return r.projectTemplateError(ctx, &template, "ExternalCreateFailed", createErr)
		}
		if err := validateProjectTemplateOrganization(created, expectedOrganizationID); err != nil {
			r.setProjectTemplateObservedState(&template, created)
			return r.projectTemplateError(ctx, &template, "ExternalTemplateMismatch", err)
		}
		r.setProjectTemplateObservedState(&template, created)
	}

	current, err := apiClient.GetProjectTemplate(ctx, template.Status.TemplateID)
	if err != nil {
		if infisicalclient.IsNotFound(err) {
			before := template.DeepCopy()
			template.Status.TemplateID = ""
			template.Status.Name = ""
			template.Status.OrganizationID = ""
			template.Status.ObservedGeneration = template.Generation
			setCondition(&template.Status.Conditions, template.Generation, "False", "RemoteProjectTemplateMissing", "the project template no longer exists in Infisical; it will be recreated according to creationPolicy")
			return ctrl.Result{RequeueAfter: externalRetry}, persistStatus(ctx, r.Client, &template, before)
		}
		return r.projectTemplateError(ctx, &template, "ExternalReadFailed", err)
	}
	if err := validateProjectTemplateOrganization(current, expectedOrganizationID); err != nil {
		return r.projectTemplateError(ctx, &template, "ExternalTemplateMismatch", err)
	}

	if projectTemplateNeedsUpdate(&template, current) {
		updated, updateErr := apiClient.UpdateProjectTemplate(ctx, current.ID, projectTemplatePatchFrom(&template))
		if updateErr != nil {
			return r.projectTemplateError(ctx, &template, "ExternalUpdateFailed", updateErr)
		}
		current = updated
	}

	r.setProjectTemplateObservedState(&template, current)
	setCondition(&template.Status.Conditions, template.Generation, "True", "Ready", "Infisical project template is reconciled")
	return ctrl.Result{RequeueAfter: driftDetectionEvery}, persistStatus(ctx, r.Client, &template, before)
}

func projectTemplateName(template *infisicalv1alpha1.InfisicalProjectTemplate) string {
	if template.Spec.TemplateName != "" {
		return template.Spec.TemplateName
	}
	return template.Name
}

func projectTemplateType(template *infisicalv1alpha1.InfisicalProjectTemplate) infisicalv1alpha1.ProjectType {
	if template.Spec.Type == "" {
		return infisicalv1alpha1.ProjectTypeSecretManager
	}
	return template.Spec.Type
}

func projectTemplateRequestFrom(template *infisicalv1alpha1.InfisicalProjectTemplate) infisicalclient.CreateProjectTemplateRequest {
	return infisicalclient.CreateProjectTemplateRequest{
		Name:                     projectTemplateName(template),
		Description:              template.Spec.Description,
		Type:                     string(projectTemplateType(template)),
		Roles:                    projectTemplateRolesFrom(template.Spec.Roles),
		Environments:             projectTemplateEnvironmentsFrom(template.Spec.Environments),
		Users:                    projectTemplateUsersFrom(template.Spec.Users),
		Groups:                   projectTemplateGroupsFrom(template.Spec.Groups),
		Identities:               projectTemplateIdentitiesFrom(template.Spec.Identities),
		ProjectManagedIdentities: projectTemplateManagedIdentitiesFrom(template.Spec.ProjectManagedIdentities),
	}
}

func projectTemplatePatchFrom(template *infisicalv1alpha1.InfisicalProjectTemplate) infisicalclient.ProjectTemplatePatch {
	request := projectTemplateRequestFrom(template)
	return infisicalclient.ProjectTemplatePatch{
		Name:                     request.Name,
		Description:              request.Description,
		Roles:                    request.Roles,
		Environments:             request.Environments,
		Users:                    request.Users,
		Groups:                   request.Groups,
		Identities:               request.Identities,
		ProjectManagedIdentities: request.ProjectManagedIdentities,
	}
}

func projectTemplateRolesFrom(roles []infisicalv1alpha1.ProjectTemplateRole) []infisicalclient.ProjectTemplateRole {
	result := make([]infisicalclient.ProjectTemplateRole, 0, len(roles))
	for _, role := range roles {
		result = append(result, infisicalclient.ProjectTemplateRole{
			Name:        role.Name,
			Slug:        role.Slug,
			Permissions: projectRolePermissionsFrom(role.Permissions),
		})
	}
	return result
}

func projectTemplateEnvironmentsFrom(environments []infisicalv1alpha1.ProjectTemplateEnvironment) []infisicalclient.ProjectTemplateEnvironment {
	result := make([]infisicalclient.ProjectTemplateEnvironment, 0, len(environments))
	for _, environment := range environments {
		result = append(result, infisicalclient.ProjectTemplateEnvironment{Name: environment.Name, Slug: environment.Slug, Position: environment.Position})
	}
	return result
}

func projectTemplateUsersFrom(users []infisicalv1alpha1.ProjectTemplateUser) []infisicalclient.ProjectTemplateUser {
	result := make([]infisicalclient.ProjectTemplateUser, 0, len(users))
	for _, user := range users {
		result = append(result, infisicalclient.ProjectTemplateUser{Username: user.Username, Roles: append([]string(nil), user.Roles...)})
	}
	return result
}

func projectTemplateGroupsFrom(groups []infisicalv1alpha1.ProjectTemplateGroup) []infisicalclient.ProjectTemplateGroup {
	result := make([]infisicalclient.ProjectTemplateGroup, 0, len(groups))
	for _, group := range groups {
		result = append(result, infisicalclient.ProjectTemplateGroup{GroupSlug: group.GroupSlug, Roles: append([]string(nil), group.Roles...)})
	}
	return result
}

func projectTemplateIdentitiesFrom(identities []infisicalv1alpha1.ProjectTemplateIdentity) []infisicalclient.ProjectTemplateIdentity {
	result := make([]infisicalclient.ProjectTemplateIdentity, 0, len(identities))
	for _, identity := range identities {
		result = append(result, infisicalclient.ProjectTemplateIdentity{IdentityID: identity.IdentityID, Roles: append([]string(nil), identity.Roles...)})
	}
	return result
}

func projectTemplateManagedIdentitiesFrom(identities []infisicalv1alpha1.ProjectTemplateManagedIdentity) []infisicalclient.ProjectTemplateManagedIdentity {
	result := make([]infisicalclient.ProjectTemplateManagedIdentity, 0, len(identities))
	for _, identity := range identities {
		result = append(result, infisicalclient.ProjectTemplateManagedIdentity{Name: identity.Name, Roles: append([]string(nil), identity.Roles...)})
	}
	return result
}

func projectTemplateNeedsUpdate(template *infisicalv1alpha1.InfisicalProjectTemplate, current *infisicalclient.ProjectTemplate) bool {
	if current.Name != projectTemplateName(template) || current.Description != template.Spec.Description {
		return true
	}
	return !reflect.DeepEqual(current.Roles, projectTemplateRolesFrom(template.Spec.Roles)) ||
		!reflect.DeepEqual(current.Environments, projectTemplateEnvironmentsFrom(template.Spec.Environments)) ||
		!reflect.DeepEqual(current.Users, projectTemplateUsersFrom(template.Spec.Users)) ||
		!reflect.DeepEqual(current.Groups, projectTemplateGroupsFrom(template.Spec.Groups)) ||
		!reflect.DeepEqual(current.Identities, projectTemplateIdentitiesFrom(template.Spec.Identities)) ||
		!reflect.DeepEqual(current.ProjectManagedIdentities, projectTemplateManagedIdentitiesFrom(template.Spec.ProjectManagedIdentities))
}

func validateProjectTemplateOrganization(template *infisicalclient.ProjectTemplate, expectedOrganizationID string) error {
	if expectedOrganizationID != "" && template.OrganizationID != expectedOrganizationID {
		return newDependencyError("infisical project template belongs to organization %q, want %q", template.OrganizationID, expectedOrganizationID)
	}
	return nil
}

func (r *InfisicalProjectTemplateReconciler) expectedProjectTemplateOrganizationID(ctx context.Context, template *infisicalv1alpha1.InfisicalProjectTemplate) (string, error) {
	if template.Spec.OrganizationRef == nil || template.Spec.OrganizationRef.Name == "" {
		return "", nil
	}
	var organization infisicalv1alpha1.InfisicalOrganization
	if err := r.Get(ctx, client.ObjectKey{Namespace: template.Namespace, Name: template.Spec.OrganizationRef.Name}, &organization); err != nil {
		if apierrors.IsNotFound(err) {
			return "", newDependencyError("InfisicalOrganization %s/%s was not found", template.Namespace, template.Spec.OrganizationRef.Name)
		}
		return "", err
	}
	if organization.Status.OrganizationID == "" {
		return "", newDependencyError("InfisicalOrganization %s/%s has no observed Infisical organization ID", organization.Namespace, organization.Name)
	}
	return organization.Status.OrganizationID, nil
}

func (r *InfisicalProjectTemplateReconciler) setProjectTemplateObservedState(template *infisicalv1alpha1.InfisicalProjectTemplate, observed *infisicalclient.ProjectTemplate) {
	template.Status.TemplateID = observed.ID
	template.Status.Name = observed.Name
	template.Status.Description = observed.Description
	template.Status.Type = infisicalv1alpha1.ProjectType(observed.Type)
	template.Status.OrganizationID = observed.OrganizationID
	template.Status.ObservedGeneration = template.Generation
}

func (r *InfisicalProjectTemplateReconciler) projectTemplateError(ctx context.Context, template *infisicalv1alpha1.InfisicalProjectTemplate, reason string, err error) (ctrl.Result, error) {
	before := template.DeepCopy()
	template.Status.ObservedGeneration = template.Generation
	setCondition(&template.Status.Conditions, template.Generation, "False", reason, statusErrorMessage(err))
	return ctrl.Result{RequeueAfter: retryFor(err)}, persistStatus(ctx, r.Client, template, before)
}

func (r *InfisicalProjectTemplateReconciler) reconcileProjectTemplateDeletion(ctx context.Context, template *infisicalv1alpha1.InfisicalProjectTemplate) (ctrl.Result, error) {
	if deletionPolicy(template.Spec.DeletionPolicy) == infisicalv1alpha1.DeletionPolicyOrphan || template.Status.TemplateID == "" {
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, template)
	}
	apiClient, err := infisicalClientForConnection(ctx, r.Client, template.Namespace, template.Spec.ConnectionRef)
	if err != nil {
		return ctrl.Result{RequeueAfter: retryFor(err)}, err
	}
	if err := apiClient.DeleteProjectTemplate(ctx, template.Status.TemplateID); err != nil && !infisicalclient.IsNotFound(err) {
		return ctrl.Result{RequeueAfter: externalRetry}, err
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, template)
}

func (r *InfisicalProjectTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infisicalv1alpha1.InfisicalProjectTemplate{}).
		Watches(&infisicalv1alpha1.InfisicalConnection{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var templates infisicalv1alpha1.InfisicalProjectTemplateList
			if err := mgr.GetClient().List(ctx, &templates, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range templates.Items {
				template := &templates.Items[i]
				if template.Spec.ConnectionRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(template)})
				}
			}
			return requests
		})).
		Watches(&infisicalv1alpha1.InfisicalOrganization{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var templates infisicalv1alpha1.InfisicalProjectTemplateList
			if err := mgr.GetClient().List(ctx, &templates, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range templates.Items {
				template := &templates.Items[i]
				if template.Spec.OrganizationRef != nil && template.Spec.OrganizationRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(template)})
				}
			}
			return requests
		})).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var templates infisicalv1alpha1.InfisicalProjectTemplateList
			if err := mgr.GetClient().List(ctx, &templates, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range templates.Items {
				template := &templates.Items[i]
				var connection infisicalv1alpha1.InfisicalConnection
				if err := mgr.GetClient().Get(ctx, client.ObjectKey{Namespace: template.Namespace, Name: template.Spec.ConnectionRef.Name}, &connection); err == nil && connectionReferencesSecret(&connection, object.GetName()) {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(template)})
				}
			}
			return requests
		})).
		Complete(r)
}
