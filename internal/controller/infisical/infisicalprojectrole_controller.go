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

// InfisicalProjectRoleReconciler reconciles an Infisical project role.
type InfisicalProjectRoleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalprojectroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalprojectroles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalprojectroles/finalizers,verbs=update
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalprojects,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *InfisicalProjectRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var role infisicalv1alpha1.InfisicalProjectRole
	if err := r.Get(ctx, req.NamespacedName, &role); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if role.DeletionTimestamp.IsZero() && deletionPolicy(role.Spec.DeletionPolicy) == infisicalv1alpha1.DeletionPolicyDelete {
		if added, err := ensureFinalizer(ctx, r.Client, &role); err != nil {
			return ctrl.Result{}, err
		} else if added {
			return ctrl.Result{}, nil
		}
	}
	if !role.DeletionTimestamp.IsZero() {
		return r.reconcileProjectRoleDeletion(ctx, &role)
	}
	before := role.Status

	var project infisicalv1alpha1.InfisicalProject
	if err := r.Get(ctx, client.ObjectKey{Namespace: role.Namespace, Name: role.Spec.ProjectRef.Name}, &project); err != nil {
		if apierrors.IsNotFound(err) {
			return r.projectRoleError(ctx, &role, "ProjectNotReady", newDependencyError("InfisicalProject %s/%s was not found", role.Namespace, role.Spec.ProjectRef.Name))
		}
		return r.projectRoleError(ctx, &role, "ProjectReadFailed", err)
	}
	if project.Status.ProjectID == "" {
		return r.projectRoleError(ctx, &role, "ProjectNotReady", newDependencyError("InfisicalProject %s/%s has no observed Infisical project ID", project.Namespace, project.Name))
	}

	apiClient, err := infisicalClientForConnection(ctx, r.Client, role.Namespace, role.Spec.ConnectionRef)
	if err != nil {
		return r.projectRoleError(ctx, &role, "ConnectionNotReady", err)
	}

	if role.Status.RoleID == "" && canAdopt(role.Spec.CreationPolicy) {
		adopted, findErr := apiClient.GetProjectRoleBySlug(ctx, project.Status.ProjectID, projectRoleSlug(&role))
		if findErr != nil && !infisicalclient.IsNotFound(findErr) {
			return r.projectRoleError(ctx, &role, "ExternalReadFailed", findErr)
		}
		if adopted != nil {
			r.setProjectRoleObservedState(&role, adopted, project.Status.ProjectID)
		}
	}

	permissions := projectRolePermissionsFrom(role.Spec.Permissions)
	if role.Status.RoleID == "" {
		if !canCreate(role.Spec.CreationPolicy) {
			return r.projectRoleError(ctx, &role, "CreationNotAllowed", newDependencyError("project role was not found and creationPolicy is Adopt"))
		}
		created, createErr := apiClient.CreateProjectRole(ctx, project.Status.ProjectID, infisicalclient.CreateProjectRoleRequest{
			Slug:        projectRoleSlug(&role),
			Name:        projectRoleName(&role),
			Description: role.Spec.Description,
			Permissions: permissions,
		})
		if createErr != nil {
			return r.projectRoleError(ctx, &role, "ExternalCreateFailed", createErr)
		}
		r.setProjectRoleObservedState(&role, created, project.Status.ProjectID)
	}

	current, err := apiClient.GetProjectRoleByID(ctx, project.Status.ProjectID, role.Status.RoleID)
	if err != nil {
		if infisicalclient.IsNotFound(err) {
			before := role.Status
			role.Status.RoleID = ""
			role.Status.ObservedGeneration = role.Generation
			setCondition(&role.Status.Conditions, role.Generation, "False", "RemoteProjectRoleMissing", "the project role no longer exists in Infisical; it will be recreated according to creationPolicy")
			return ctrl.Result{RequeueAfter: externalRetry}, persistStatus(ctx, r.Client, &role, before, role.Status)
		}
		return r.projectRoleError(ctx, &role, "ExternalReadFailed", err)
	}

	if projectRoleNeedsUpdate(&role, current, permissions) {
		updated, updateErr := apiClient.UpdateProjectRole(ctx, project.Status.ProjectID, current.ID, infisicalclient.ProjectRolePatch{
			Name:        projectRoleName(&role),
			Description: role.Spec.Description,
			Permissions: permissions,
		})
		if updateErr != nil {
			return r.projectRoleError(ctx, &role, "ExternalUpdateFailed", updateErr)
		}
		current = updated
	}

	r.setProjectRoleObservedState(&role, current, project.Status.ProjectID)
	setCondition(&role.Status.Conditions, role.Generation, "True", "Ready", "Infisical project role is reconciled")
	return ctrl.Result{RequeueAfter: driftDetectionEvery}, persistStatus(ctx, r.Client, &role, before, role.Status)
}

func projectRoleNeedsUpdate(role *infisicalv1alpha1.InfisicalProjectRole, current *infisicalclient.ProjectRole, permissions []infisicalclient.ProjectRolePermission) bool {
	if current.Name != projectRoleName(role) || current.Description != role.Spec.Description {
		return true
	}
	return !reflect.DeepEqual(current.Permissions, permissions)
}

func (r *InfisicalProjectRoleReconciler) setProjectRoleObservedState(role *infisicalv1alpha1.InfisicalProjectRole, observed *infisicalclient.ProjectRole, projectID string) {
	role.Status.RoleID = observed.ID
	role.Status.ProjectID = projectID
	role.Status.Name = observed.Name
	role.Status.Slug = observed.Slug
	role.Status.Description = observed.Description
	role.Status.Permissions = projectRolePermissionsTo(observed.Permissions)
	role.Status.ObservedGeneration = role.Generation
}

func (r *InfisicalProjectRoleReconciler) projectRoleError(ctx context.Context, role *infisicalv1alpha1.InfisicalProjectRole, reason string, err error) (ctrl.Result, error) {
	before := role.Status
	role.Status.ObservedGeneration = role.Generation
	setCondition(&role.Status.Conditions, role.Generation, "False", reason, statusErrorMessage(err))
	return ctrl.Result{RequeueAfter: retryFor(err)}, persistStatus(ctx, r.Client, role, before, role.Status)
}

func (r *InfisicalProjectRoleReconciler) reconcileProjectRoleDeletion(ctx context.Context, role *infisicalv1alpha1.InfisicalProjectRole) (ctrl.Result, error) {
	if deletionPolicy(role.Spec.DeletionPolicy) == infisicalv1alpha1.DeletionPolicyOrphan || role.Status.RoleID == "" || role.Status.ProjectID == "" {
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, role)
	}

	apiClient, err := infisicalClientForConnection(ctx, r.Client, role.Namespace, role.Spec.ConnectionRef)
	if err != nil {
		return ctrl.Result{RequeueAfter: retryFor(err)}, err
	}
	if err := apiClient.DeleteProjectRole(ctx, role.Status.ProjectID, role.Status.RoleID); err != nil && !infisicalclient.IsNotFound(err) {
		return ctrl.Result{RequeueAfter: externalRetry}, err
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, role)
}

func (r *InfisicalProjectRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infisicalv1alpha1.InfisicalProjectRole{}).
		Watches(&infisicalv1alpha1.InfisicalConnection{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var roles infisicalv1alpha1.InfisicalProjectRoleList
			if err := mgr.GetClient().List(ctx, &roles, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range roles.Items {
				role := &roles.Items[i]
				if role.Spec.ConnectionRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(role)})
				}
			}
			return requests
		})).
		Watches(&infisicalv1alpha1.InfisicalProject{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var roles infisicalv1alpha1.InfisicalProjectRoleList
			if err := mgr.GetClient().List(ctx, &roles, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range roles.Items {
				role := &roles.Items[i]
				if role.Spec.ProjectRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(role)})
				}
			}
			return requests
		})).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var roles infisicalv1alpha1.InfisicalProjectRoleList
			if err := mgr.GetClient().List(ctx, &roles, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range roles.Items {
				role := &roles.Items[i]
				var connection infisicalv1alpha1.InfisicalConnection
				if err := mgr.GetClient().Get(ctx, client.ObjectKey{Namespace: role.Namespace, Name: role.Spec.ConnectionRef.Name}, &connection); err == nil && connection.Spec.AuthSecretRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(role)})
				}
			}
			return requests
		})).
		Complete(r)
}
