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

// InfisicalProjectReconciler reconciles an Infisical project.
type InfisicalProjectReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalprojects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalprojects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalprojects/finalizers,verbs=update
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalconnections,verbs=get;list;watch
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
	before := project.Status

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
				r.setProjectObservedState(&project, adopted)
			}
		}
	}

	if project.Status.ProjectID == "" {
		if !canCreate(project.Spec.CreationPolicy) {
			return r.projectError(ctx, &project, "CreationNotAllowed", newDependencyError("project was not found and creationPolicy is Adopt"))
		}
		created, createErr := apiClient.CreateProject(ctx, infisicalclient.CreateProjectRequest{
			ProjectName:             projectName(&project),
			ProjectDescription:      project.Spec.Description,
			Slug:                    project.Spec.Slug,
			Template:                "default",
			Type:                    string(projectType(&project)),
			ShouldCreateDefaultEnvs: boolValue(project.Spec.ShouldCreateDefaultEnvs, true),
			HasDeleteProtection:     boolValue(project.Spec.HasDeleteProtection, false),
		})
		if createErr != nil {
			return r.projectError(ctx, &project, "ExternalCreateFailed", createErr)
		}
		r.setProjectObservedState(&project, created)
	}

	current, err := apiClient.GetProject(ctx, project.Status.ProjectID)
	if err != nil {
		if infisicalclient.IsNotFound(err) {
			before := project.Status
			project.Status.ProjectID = ""
			project.Status.Slug = ""
			project.Status.OrganizationID = ""
			project.Status.Environments = nil
			project.Status.ObservedGeneration = project.Generation
			setCondition(&project.Status.Conditions, project.Generation, "False", "RemoteProjectMissing", "the project no longer exists in Infisical; it will be recreated according to creationPolicy")
			return ctrl.Result{RequeueAfter: externalRetry}, persistStatus(ctx, r.Client, &project, before, project.Status)
		}
		return r.projectError(ctx, &project, "ExternalReadFailed", err)
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
	return ctrl.Result{RequeueAfter: driftDetectionEvery}, persistStatus(ctx, r.Client, &project, before, project.Status)
}

func projectType(project *infisicalv1alpha1.InfisicalProject) infisicalv1alpha1.ProjectType {
	if project.Spec.Type == "" {
		return infisicalv1alpha1.ProjectTypeSecretManager
	}
	return project.Spec.Type
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
	before := project.Status
	project.Status.ObservedGeneration = project.Generation
	setCondition(&project.Status.Conditions, project.Generation, "False", reason, statusErrorMessage(err))
	return ctrl.Result{RequeueAfter: retryFor(err)}, persistStatus(ctx, r.Client, project, before, project.Status)
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
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var projects infisicalv1alpha1.InfisicalProjectList
			if err := mgr.GetClient().List(ctx, &projects, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range projects.Items {
				project := &projects.Items[i]
				var connection infisicalv1alpha1.InfisicalConnection
				if err := mgr.GetClient().Get(ctx, client.ObjectKey{Namespace: project.Namespace, Name: project.Spec.ConnectionRef.Name}, &connection); err == nil && connection.Spec.AuthSecretRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(project)})
				}
			}
			return requests
		})).
		Complete(r)
}
