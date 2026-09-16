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

// InfisicalProjectReconciler reconciles an Infisical project.
type InfisicalProjectReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalprojects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalprojects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalprojects/finalizers,verbs=update
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalorganizations,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *InfisicalProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var project infisicalv1alpha1.InfisicalProject
	if err := r.Get(ctx, req.NamespacedName, &project); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if project.DeletionTimestamp.IsZero() && deletionPolicy(project.Spec.DeletionPolicy) == infisicalv1alpha1.DeletionPolicyDelete {
		if added, err := ensureFinalizer(ctx, r.Client, &project); err != nil {
			return ctrl.Result{}, err
		} else if added {
			return ctrl.Result{}, nil
		}
	}
	if !project.DeletionTimestamp.IsZero() {
		return r.reconcileProjectDeletion(ctx, &project)
	}
	before := project.DeepCopy()
	expectedOrganizationID, err := r.expectedOrganizationID(ctx, &project)
	if err != nil {
		return r.projectError(ctx, &project, "OrganizationNotReady", err)
	}

	apiClient, err := infisicalClientForConnection(ctx, r.Client, project.Namespace, project.Spec.ConnectionRef)
	if err != nil {
		return r.projectError(ctx, &project, "ConnectionNotReady", err)
	}

	if project.Status.ProjectID == "" {
		if canAdopt(project.Spec.CreationPolicy) {
			adopted, findErr := apiClient.FindProject(ctx, projectName(&project), project.Spec.Slug)
			if findErr != nil {
				return r.projectError(ctx, &project, "ExternalReadFailed", findErr)
			}
			if adopted != nil {
				if err := validateProjectOrganization(adopted, expectedOrganizationID); err != nil {
					return r.projectError(ctx, &project, "ExternalProjectMismatch", err)
				}
				r.setProjectObservedState(&project, adopted)
			} else if !canCreate(project.Spec.CreationPolicy) {
				return r.projectError(ctx, &project, "RemoteProjectMissing", newDependencyError("the project was not found in Infisical and creationPolicy is Adopt"))
			}
		}
	}

	if project.Status.ProjectID == "" {
		if !canCreate(project.Spec.CreationPolicy) {
			return r.projectError(ctx, &project, "CreationNotAllowed", newDependencyError("project was not found and creationPolicy is Adopt"))
		}
		templateName, templateErr := r.projectTemplateName(ctx, &project, expectedOrganizationID)
		if templateErr != nil {
			return r.projectError(ctx, &project, "ProjectTemplateNotReady", templateErr)
		}
		created, createErr := apiClient.CreateProject(ctx, infisicalclient.CreateProjectRequest{
			ProjectName:             projectName(&project),
			ProjectDescription:      project.Spec.Description,
			Slug:                    project.Spec.Slug,
			Template:                templateName,
			KMSKeyID:                project.Spec.KMSKeyID,
			Type:                    string(projectType(&project)),
			ShouldCreateDefaultEnvs: boolValue(project.Spec.ShouldCreateDefaultEnvs, true),
			HasDeleteProtection:     boolValue(project.Spec.HasDeleteProtection, false),
		})
		if createErr != nil {
			return r.projectError(ctx, &project, "ExternalCreateFailed", createErr)
		}
		if err := validateProjectOrganization(created, expectedOrganizationID); err != nil {
			r.setProjectObservedState(&project, created)
			return r.projectError(ctx, &project, "ExternalProjectMismatch", err)
		}
		r.setProjectObservedState(&project, created)
	}
	if err := r.ensureProjectCreatorMembership(ctx, apiClient, &project); err != nil {
		return r.projectError(ctx, &project, "ExternalMembershipFailed", err)
	}

	current, err := apiClient.GetProject(ctx, project.Status.ProjectID)
	if err != nil {
		if infisicalclient.IsNotFound(err) {
			before := project.DeepCopy()
			project.Status.ProjectID = ""
			project.Status.Slug = ""
			project.Status.OrganizationID = ""
			project.Status.Environments = nil
			project.Status.ObservedGeneration = project.Generation
			setCondition(&project.Status.Conditions, project.Generation, "False", "RemoteProjectMissing", "the project no longer exists in Infisical; it will be recreated according to creationPolicy")
			return ctrl.Result{RequeueAfter: externalRetry}, persistStatus(ctx, r.Client, &project, before)
		}
		return r.projectError(ctx, &project, "ExternalReadFailed", err)
	}
	if err := validateProjectOrganization(current, expectedOrganizationID); err != nil {
		return r.projectError(ctx, &project, "ExternalProjectMismatch", err)
	}

	if projectNeedsUpdate(&project, current) {
		updated, updateErr := apiClient.UpdateProject(ctx, current.ID, infisicalclient.ProjectPatch{
			Name:                projectName(&project),
			Description:         project.Spec.Description,
			Slug:                project.Spec.Slug,
			HasDeleteProtection: project.Spec.HasDeleteProtection,
		})
		if updateErr != nil {
			return r.projectError(ctx, &project, "ExternalUpdateFailed", updateErr)
		}
		current = updated
	}

	r.setProjectObservedState(&project, current)
	setCondition(&project.Status.Conditions, project.Generation, "True", "Ready", "Infisical project is reconciled")
	return ctrl.Result{RequeueAfter: driftDetectionEvery}, persistStatus(ctx, r.Client, &project, before)
}

func (r *InfisicalProjectReconciler) projectTemplateName(ctx context.Context, project *infisicalv1alpha1.InfisicalProject, expectedOrganizationID string) (string, error) {
	if project.Spec.TemplateRef == nil || project.Spec.TemplateRef.Name == "" {
		return "default", nil
	}

	var template infisicalv1alpha1.InfisicalProjectTemplate
	if err := r.Get(ctx, client.ObjectKey{Namespace: project.Namespace, Name: project.Spec.TemplateRef.Name}, &template); err != nil {
		if apierrors.IsNotFound(err) {
			return "", newDependencyError("InfisicalProjectTemplate %s/%s was not found", project.Namespace, project.Spec.TemplateRef.Name)
		}
		return "", err
	}
	if template.Status.TemplateID == "" || template.Status.Name == "" {
		return "", newDependencyError("InfisicalProjectTemplate %s/%s has no observed Infisical template", template.Namespace, template.Name)
	}
	if expectedOrganizationID != "" && template.Status.OrganizationID != expectedOrganizationID {
		return "", newDependencyError("InfisicalProjectTemplate %s/%s belongs to organization %q, want %q", template.Namespace, template.Name, template.Status.OrganizationID, expectedOrganizationID)
	}
	return template.Status.Name, nil
}

func (r *InfisicalProjectReconciler) ensureProjectCreatorMembership(ctx context.Context, apiClient *infisicalclient.Client, project *infisicalv1alpha1.InfisicalProject) error {
	if project.Spec.OrganizationRef == nil || project.Status.ProjectID == "" {
		return nil
	}
	identityID, err := apiClient.TokenIdentityIDContext(ctx)
	if err != nil {
		return fmt.Errorf("resolve creating identity from access token: %w", err)
	}
	if identityID == "" {
		return nil
	}
	if err := apiClient.EnsureIdentityProjectMembership(ctx, project.Status.ProjectID, identityID, []string{"admin"}); err != nil {
		return fmt.Errorf("ensure creating identity %q has project admin access: %w", identityID, err)
	}
	return nil
}

func projectType(project *infisicalv1alpha1.InfisicalProject) infisicalv1alpha1.ProjectType {
	if project.Spec.Type == "" {
		return infisicalv1alpha1.ProjectTypeSecretManager
	}
	return project.Spec.Type
}

func (r *InfisicalProjectReconciler) expectedOrganizationID(ctx context.Context, project *infisicalv1alpha1.InfisicalProject) (string, error) {
	if project.Spec.OrganizationRef == nil || project.Spec.OrganizationRef.Name == "" {
		return "", nil
	}
	var organization infisicalv1alpha1.InfisicalOrganization
	if err := r.Get(ctx, client.ObjectKey{Namespace: project.Namespace, Name: project.Spec.OrganizationRef.Name}, &organization); err != nil {
		if apierrors.IsNotFound(err) {
			return "", newDependencyError("InfisicalOrganization %s/%s was not found", project.Namespace, project.Spec.OrganizationRef.Name)
		}
		return "", err
	}
	if organization.Status.OrganizationID == "" {
		return "", newDependencyError("InfisicalOrganization %s/%s has no observed Infisical organization ID", organization.Namespace, organization.Name)
	}
	return organization.Status.OrganizationID, nil
}

func validateProjectOrganization(project *infisicalclient.Project, expectedOrganizationID string) error {
	if expectedOrganizationID != "" && project.OrganizationID != expectedOrganizationID {
		return fmt.Errorf("infisical project belongs to organization %q, want %q", project.OrganizationID, expectedOrganizationID)
	}
	return nil
}

func projectNeedsUpdate(project *infisicalv1alpha1.InfisicalProject, current *infisicalclient.Project) bool {
	if current.Name != projectName(project) || current.Description != project.Spec.Description {
		return true
	}
	if project.Spec.Slug != "" && current.Slug != project.Spec.Slug {
		return true
	}
	if project.Spec.HasDeleteProtection != nil && current.HasDeleteProtection != *project.Spec.HasDeleteProtection {
		return true
	}
	return false
}

func (r *InfisicalProjectReconciler) setProjectObservedState(project *infisicalv1alpha1.InfisicalProject, observed *infisicalclient.Project) {
	project.Status.ProjectID, project.Status.Slug, project.Status.OrganizationID, project.Status.Environments = projectStatusFrom(observed)
	project.Status.ObservedGeneration = project.Generation
}

func (r *InfisicalProjectReconciler) projectError(ctx context.Context, project *infisicalv1alpha1.InfisicalProject, reason string, err error) (ctrl.Result, error) {
	before := project.DeepCopy()
	project.Status.ObservedGeneration = project.Generation
	setCondition(&project.Status.Conditions, project.Generation, "False", reason, statusErrorMessage(err))
	return ctrl.Result{RequeueAfter: retryFor(err)}, persistStatus(ctx, r.Client, project, before)
}

func (r *InfisicalProjectReconciler) reconcileProjectDeletion(ctx context.Context, project *infisicalv1alpha1.InfisicalProject) (ctrl.Result, error) {
	if deletionPolicy(project.Spec.DeletionPolicy) == infisicalv1alpha1.DeletionPolicyOrphan || project.Status.ProjectID == "" {
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, project)
	}

	apiClient, err := infisicalClientForConnection(ctx, r.Client, project.Namespace, project.Spec.ConnectionRef)
	if err != nil {
		return ctrl.Result{RequeueAfter: retryFor(err)}, err
	}
	if err := apiClient.DeleteProject(ctx, project.Status.ProjectID); err != nil && !infisicalclient.IsNotFound(err) {
		return ctrl.Result{RequeueAfter: externalRetry}, err
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, project)
}

func (r *InfisicalProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infisicalv1alpha1.InfisicalProject{}).
		Watches(&infisicalv1alpha1.InfisicalConnection{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var projects infisicalv1alpha1.InfisicalProjectList
			if err := mgr.GetClient().List(ctx, &projects, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range projects.Items {
				project := &projects.Items[i]
				if project.Spec.ConnectionRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(project)})
				}
			}
			return requests
		})).
		Watches(&infisicalv1alpha1.InfisicalOrganization{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var projects infisicalv1alpha1.InfisicalProjectList
			if err := mgr.GetClient().List(ctx, &projects, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range projects.Items {
				project := &projects.Items[i]
				if project.Spec.OrganizationRef != nil && project.Spec.OrganizationRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(project)})
				}
			}
			return requests
		})).
		Watches(&infisicalv1alpha1.InfisicalProjectTemplate{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var projects infisicalv1alpha1.InfisicalProjectList
			if err := mgr.GetClient().List(ctx, &projects, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range projects.Items {
				project := &projects.Items[i]
				if project.Spec.TemplateRef != nil && project.Spec.TemplateRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(project)})
				}
			}
			return requests
		})).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var projects infisicalv1alpha1.InfisicalProjectList
			if err := mgr.GetClient().List(ctx, &projects, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range projects.Items {
				project := &projects.Items[i]
				var connection infisicalv1alpha1.InfisicalConnection
				if err := mgr.GetClient().Get(ctx, client.ObjectKey{Namespace: project.Namespace, Name: project.Spec.ConnectionRef.Name}, &connection); err == nil && connectionReferencesSecret(&connection, object.GetName()) {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(project)})
				}
			}
			return requests
		})).
		Complete(r)
}
