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
	"reflect"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infisicalv1alpha1 "github.com/winrarr/infisical-entity-operator/api/infisical/v1alpha1"
	"github.com/winrarr/infisical-entity-operator/internal/infisicalclient"
)

const (
	finalizerName       = "infisical.infisical-operator.io/finalizer"
	dependencyRetry     = 15 * time.Second
	externalRetry       = 30 * time.Second
	driftDetectionEvery = 2 * time.Minute
	readyCondition      = "Ready"
)

type dependencyError struct {
	message string
}

func (e *dependencyError) Error() string { return e.message }

func newDependencyError(format string, args ...any) error {
	return &dependencyError{message: fmt.Sprintf(format, args...)}
}

func isDependencyError(err error) bool {
	var dependencyErr *dependencyError
	return errors.As(err, &dependencyErr)
}

func infisicalClientForConnection(ctx context.Context, kubeClient client.Client, namespace string, ref infisicalv1alpha1.InfisicalConnectionReference) (*infisicalclient.Client, error) {
	if strings.TrimSpace(ref.Name) == "" {
		return nil, newDependencyError("connectionRef.name is required")
	}

	var connection infisicalv1alpha1.InfisicalConnection
	if err := kubeClient.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: namespace}, &connection); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, newDependencyError("InfisicalConnection %s/%s was not found", namespace, ref.Name)
		}
		return nil, fmt.Errorf("get InfisicalConnection %s/%s: %w", namespace, ref.Name, err)
	}

	token, err := secretValueFromReference(ctx, kubeClient, namespace, connection.Spec.AuthSecretRef, "token", "authentication")
	if err != nil {
		return nil, err
	}

	hostAPI := connection.Spec.HostAPI
	if hostAPI == "" {
		hostAPI = "https://app.infisical.com/api"
	}
	timeout := 30 * time.Second
	if connection.Spec.RequestTimeout != nil && connection.Spec.RequestTimeout.Duration > 0 {
		timeout = connection.Spec.RequestTimeout.Duration
	}
	return infisicalclient.New(hostAPI, token, timeout)
}

func secretValueFromReference(ctx context.Context, kubeClient client.Client, namespace string, ref infisicalv1alpha1.SecretKeyReference, defaultKey, purpose string) (string, error) {
	key := ref.Key
	if key == "" {
		key = defaultKey
	}
	var secret corev1.Secret
	if err := kubeClient.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: namespace}, &secret); err != nil {
		if apierrors.IsNotFound(err) {
			return "", newDependencyError("%s Secret %s/%s was not found", purpose, namespace, ref.Name)
		}
		return "", fmt.Errorf("get %s Secret %s/%s: %w", purpose, namespace, ref.Name, err)
	}
	value, ok := secret.Data[key]
	if !ok || strings.TrimSpace(string(value)) == "" {
		return "", newDependencyError("%s Secret %s/%s does not contain a non-empty %q key", purpose, namespace, ref.Name, key)
	}
	return string(value), nil
}

func optionalSecretValueFromReference(ctx context.Context, kubeClient client.Client, namespace string, ref *infisicalv1alpha1.SecretKeyReference, defaultKey, purpose string) (string, error) {
	if ref == nil {
		return "", nil
	}
	return secretValueFromReference(ctx, kubeClient, namespace, *ref, defaultKey, purpose)
}

func ensureFinalizer(ctx context.Context, kubeClient client.Client, object client.Object) (bool, error) {
	if controllerutil.ContainsFinalizer(object, finalizerName) {
		return false, nil
	}
	controllerutil.AddFinalizer(object, finalizerName)
	return true, kubeClient.Update(ctx, object)
}

func removeFinalizer(ctx context.Context, kubeClient client.Client, object client.Object) error {
	if !controllerutil.ContainsFinalizer(object, finalizerName) {
		return nil
	}
	controllerutil.RemoveFinalizer(object, finalizerName)
	return kubeClient.Update(ctx, object)
}

func deletionPolicy(policy infisicalv1alpha1.DeletionPolicy) infisicalv1alpha1.DeletionPolicy {
	if policy == "" {
		return infisicalv1alpha1.DeletionPolicyOrphan
	}
	return policy
}

func creationPolicy(policy infisicalv1alpha1.CreationPolicy) infisicalv1alpha1.CreationPolicy {
	if policy == "" {
		return infisicalv1alpha1.CreationPolicyCreate
	}
	return policy
}

func canAdopt(policy infisicalv1alpha1.CreationPolicy) bool {
	policy = creationPolicy(policy)
	return policy == infisicalv1alpha1.CreationPolicyAdopt || policy == infisicalv1alpha1.CreationPolicyCreateOrAdopt
}

func canCreate(policy infisicalv1alpha1.CreationPolicy) bool {
	policy = creationPolicy(policy)
	return policy == infisicalv1alpha1.CreationPolicyCreate || policy == infisicalv1alpha1.CreationPolicyCreateOrAdopt
}

func setCondition(conditions *[]metav1.Condition, generation int64, status metav1.ConditionStatus, reason, message string) {
	for i := range *conditions {
		condition := &(*conditions)[i]
		if condition.Type != readyCondition {
			continue
		}
		if condition.Status == status && condition.Reason == reason && condition.Message == message && condition.ObservedGeneration == generation {
			return
		}
		transitionTime := metav1.Now()
		if condition.Status == status && !condition.LastTransitionTime.IsZero() {
			transitionTime = condition.LastTransitionTime
		}
		*condition = metav1.Condition{
			Type:               readyCondition,
			Status:             status,
			ObservedGeneration: generation,
			LastTransitionTime: transitionTime,
			Reason:             reason,
			Message:            message,
		}
		return
	}
	*conditions = append(*conditions, metav1.Condition{
		Type:               readyCondition,
		Status:             status,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

func persistStatus(ctx context.Context, kubeClient client.Client, object client.Object, before any, after any) error {
	if reflect.DeepEqual(before, after) {
		return nil
	}
	if err := kubeClient.Status().Update(ctx, object); err != nil {
		// A dependency update can race with this status write. The reconcile result
		// already requests another read, so let the next attempt use the latest object.
		if apierrors.IsConflict(err) {
			return nil
		}
		return err
	}
	return nil
}

func statusErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}

func projectName(project *infisicalv1alpha1.InfisicalProject) string {
	if project.Spec.ProjectName != "" {
		return project.Spec.ProjectName
	}
	return project.Name
}

func identityName(identity *infisicalv1alpha1.InfisicalIdentity) string {
	if identity.Spec.IdentityName != "" {
		return identity.Spec.IdentityName
	}
	return identity.Name
}

func environmentName(environment *infisicalv1alpha1.InfisicalEnvironment) string {
	if environment.Spec.EnvironmentName != "" {
		return environment.Spec.EnvironmentName
	}
	return environment.Name
}

func environmentSlug(environment *infisicalv1alpha1.InfisicalEnvironment) string {
	if environment.Spec.Slug != "" {
		return environment.Spec.Slug
	}
	return environment.Name
}

func projectRoleName(role *infisicalv1alpha1.InfisicalProjectRole) string {
	if role.Spec.RoleName != "" {
		return role.Spec.RoleName
	}
	return role.Name
}

func projectRoleSlug(role *infisicalv1alpha1.InfisicalProjectRole) string {
	if role.Spec.Slug != "" {
		return role.Spec.Slug
	}
	return role.Name
}

func projectRolePermissionsFrom(spec []infisicalv1alpha1.ProjectRolePermission) []infisicalclient.ProjectRolePermission {
	permissions := make([]infisicalclient.ProjectRolePermission, 0, len(spec))
	for _, permission := range spec {
		permissions = append(permissions, infisicalclient.ProjectRolePermission{
			Subject:    permission.Subject,
			Action:     infisicalclient.ProjectRoleActions(permission.Action),
			Inverted:   boolValue(permission.Inverted, false),
			Conditions: projectRoleConditionsFrom(permission.Conditions),
		})
	}
	return permissions
}

func projectRoleConditionsFrom(conditions *infisicalv1alpha1.ProjectRoleConditions) *infisicalclient.ProjectRoleConditions {
	if conditions == nil {
		return nil
	}
	return &infisicalclient.ProjectRoleConditions{
		Environment: projectRoleStringConditionFrom(conditions.Environment),
		SecretPath:  projectRoleStringConditionFrom(conditions.SecretPath),
		SecretName:  projectRoleStringConditionFrom(conditions.SecretName),
		SecretTags:  projectRoleSecretTagsConditionFrom(conditions.SecretTags),
		EventType:   projectRoleStringConditionFrom(conditions.EventType),
	}
}

func projectRoleStringConditionFrom(condition *infisicalv1alpha1.ProjectRoleStringCondition) *infisicalclient.ProjectRoleStringCondition {
	if condition == nil {
		return nil
	}
	return &infisicalclient.ProjectRoleStringCondition{Eq: condition.Eq, Ne: condition.Ne, In: condition.In, Glob: condition.Glob}
}

func projectRoleSecretTagsConditionFrom(condition *infisicalv1alpha1.ProjectRoleSecretTagsCondition) *infisicalclient.ProjectRoleSecretTagsCondition {
	if condition == nil {
		return nil
	}
	return &infisicalclient.ProjectRoleSecretTagsCondition{In: condition.In, All: condition.All}
}

func projectRolePermissionsTo(permissions []infisicalclient.ProjectRolePermission) []infisicalv1alpha1.ProjectRolePermission {
	result := make([]infisicalv1alpha1.ProjectRolePermission, 0, len(permissions))
	for _, permission := range permissions {
		inverted := permission.Inverted
		result = append(result, infisicalv1alpha1.ProjectRolePermission{
			Subject:    permission.Subject,
			Action:     append([]string(nil), permission.Action...),
			Inverted:   &inverted,
			Conditions: projectRoleConditionsTo(permission.Conditions),
		})
	}
	return result
}

func projectRoleConditionsTo(conditions *infisicalclient.ProjectRoleConditions) *infisicalv1alpha1.ProjectRoleConditions {
	if conditions == nil {
		return nil
	}
	return &infisicalv1alpha1.ProjectRoleConditions{
		Environment: projectRoleStringConditionTo(conditions.Environment),
		SecretPath:  projectRoleStringConditionTo(conditions.SecretPath),
		SecretName:  projectRoleStringConditionTo(conditions.SecretName),
		SecretTags:  projectRoleSecretTagsConditionTo(conditions.SecretTags),
		EventType:   projectRoleStringConditionTo(conditions.EventType),
	}
}

func projectRoleStringConditionTo(condition *infisicalclient.ProjectRoleStringCondition) *infisicalv1alpha1.ProjectRoleStringCondition {
	if condition == nil {
		return nil
	}
	return &infisicalv1alpha1.ProjectRoleStringCondition{Eq: condition.Eq, Ne: condition.Ne, In: condition.In, Glob: condition.Glob}
}

func projectRoleSecretTagsConditionTo(condition *infisicalclient.ProjectRoleSecretTagsCondition) *infisicalv1alpha1.ProjectRoleSecretTagsCondition {
	if condition == nil {
		return nil
	}
	return &infisicalv1alpha1.ProjectRoleSecretTagsCondition{In: condition.In, All: condition.All}
}

func boolValue(value *bool, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}
	return *value
}

func projectStatusFrom(project *infisicalclient.Project) (string, string, string, []infisicalv1alpha1.EnvironmentStatus) {
	environments := make([]infisicalv1alpha1.EnvironmentStatus, 0, len(project.Environments))
	for _, environment := range project.Environments {
		environments = append(environments, infisicalv1alpha1.EnvironmentStatus{
			ID:   environment.ID,
			Name: environment.Name,
			Slug: environment.Slug,
		})
	}
	return project.ID, project.Slug, project.OrganizationID, environments
}

func identityMetadataFrom(spec []infisicalv1alpha1.IdentityMetadata) []infisicalclient.IdentityMetadata {
	if spec == nil {
		return nil
	}
	metadata := make([]infisicalclient.IdentityMetadata, 0, len(spec))
	for _, item := range spec {
		metadata = append(metadata, infisicalclient.IdentityMetadata{Key: item.Key, Value: item.Value})
	}
	return metadata
}

func identityMetadataEqual(want []infisicalv1alpha1.IdentityMetadata, current []infisicalclient.IdentityMetadata) bool {
	if want == nil {
		return true
	}
	if len(want) != len(current) {
		return false
	}
	wanted := make(map[string]string, len(want))
	for _, item := range want {
		wanted[item.Key] = item.Value
	}
	observed := make(map[string]string, len(current))
	for _, item := range current {
		observed[item.Key] = item.Value
	}
	if len(wanted) != len(want) || len(observed) != len(current) {
		return false
	}
	for key, value := range wanted {
		if observed[key] != value {
			return false
		}
	}
	return true
}
