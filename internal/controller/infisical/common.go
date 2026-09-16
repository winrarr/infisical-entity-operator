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
	"net/url"
	"reflect"
	"sort"
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
	finalizerName         = "infisical.infisical-operator.io/finalizer"
	dependencyRetry       = 15 * time.Second
	externalRetry         = 30 * time.Second
	driftDetectionEvery   = 2 * time.Minute
	readyCondition        = "Ready"
	reconcilingCondition  = "Reconciling"
	stalledCondition      = "Stalled"
	infisicalAdminRole    = "admin"
	infisicalNoAccessRole = "no-access"
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
	if err := validateConnectionSpec(&connection); err != nil {
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
	if connection.Spec.AuthSecretRef != nil {
		token, err := secretValueFromReference(ctx, kubeClient, namespace, *connection.Spec.AuthSecretRef, "token", "authentication")
		if err != nil {
			return nil, err
		}
		return infisicalclient.New(hostAPI, token, timeout)
	}
	if connection.Spec.UniversalAuth == nil {
		return nil, newDependencyError("connection must configure authSecretRef or universalAuth")
	}
	clientIDKey := connection.Spec.UniversalAuth.SecretRef.ClientIDKey
	if clientIDKey == "" {
		clientIDKey = "clientId"
	}
	clientSecretKey := connection.Spec.UniversalAuth.SecretRef.ClientSecretKey
	if clientSecretKey == "" {
		clientSecretKey = "clientSecret"
	}
	clientID, err := secretValueFromReference(ctx, kubeClient, namespace, infisicalv1alpha1.SecretKeyReference{Name: connection.Spec.UniversalAuth.SecretRef.Name, Key: clientIDKey}, "client ID", "Universal Auth")
	if err != nil {
		return nil, err
	}
	clientSecret, err := secretValueFromReference(ctx, kubeClient, namespace, infisicalv1alpha1.SecretKeyReference{Name: connection.Spec.UniversalAuth.SecretRef.Name, Key: clientSecretKey}, "client secret", "Universal Auth")
	if err != nil {
		return nil, err
	}
	return infisicalclient.NewWithUniversalAuth(hostAPI, clientID, clientSecret, connection.Spec.UniversalAuth.OrganizationSlug, timeout)
}

func validateConnectionSpec(connection *infisicalv1alpha1.InfisicalConnection) error {
	if connection == nil {
		return errors.New("InfisicalConnection is required")
	}

	hasBearerToken := connection.Spec.AuthSecretRef != nil
	hasUniversalAuth := connection.Spec.UniversalAuth != nil
	if hasBearerToken == hasUniversalAuth {
		return errors.New("exactly one of authSecretRef or universalAuth must be configured")
	}
	if hasBearerToken && strings.TrimSpace(connection.Spec.AuthSecretRef.Name) == "" {
		return errors.New("authSecretRef.name is required")
	}
	if hasUniversalAuth && strings.TrimSpace(connection.Spec.UniversalAuth.SecretRef.Name) == "" {
		return errors.New("universalAuth.secretRef.name is required")
	}

	if hostAPI := strings.TrimRight(strings.TrimSpace(connection.Spec.HostAPI), "/"); hostAPI != "" {
		parsed, err := url.Parse(hostAPI)
		if err != nil {
			return fmt.Errorf("parse Infisical API URL: %w", err)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("infisical API URL must use http or https, got %q", parsed.Scheme)
		}
		if parsed.Host == "" {
			return errors.New("infisical API URL has no host")
		}
		if parsed.RawQuery != "" || parsed.Fragment != "" {
			return errors.New("infisical API URL must not contain a query or fragment")
		}
	}
	if connection.Spec.RequestTimeout != nil && connection.Spec.RequestTimeout.Duration <= 0 {
		return errors.New("requestTimeout must be greater than zero")
	}
	return nil
}

func connectionReferencesSecret(connection *infisicalv1alpha1.InfisicalConnection, name string) bool {
	if connection.Spec.AuthSecretRef != nil && connection.Spec.AuthSecretRef.Name == name {
		return true
	}
	return connection.Spec.UniversalAuth != nil && connection.Spec.UniversalAuth.SecretRef.Name == name
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
	setSingleCondition(conditions, readyCondition, generation, status, reason, message)

	if status == metav1.ConditionTrue {
		setSingleCondition(conditions, reconcilingCondition, generation, metav1.ConditionFalse, "ReconciliationSucceeded", "reconciliation completed")
		setSingleCondition(conditions, stalledCondition, generation, metav1.ConditionFalse, "NotStalled", "reconciliation can continue")
		return
	}

	if isStalledReason(reason) {
		setSingleCondition(conditions, reconcilingCondition, generation, metav1.ConditionFalse, "Stalled", "reconciliation is blocked until the resource is corrected")
		setSingleCondition(conditions, stalledCondition, generation, metav1.ConditionTrue, reason, message)
		return
	}

	setSingleCondition(conditions, reconcilingCondition, generation, metav1.ConditionTrue, "Progressing", "reconciliation is waiting for a dependency or retrying after an external error")
	setSingleCondition(conditions, stalledCondition, generation, metav1.ConditionFalse, "NotStalled", "reconciliation can continue")
}

func setSingleCondition(conditions *[]metav1.Condition, conditionType string, generation int64, status metav1.ConditionStatus, reason, message string) {
	for i := range *conditions {
		condition := &(*conditions)[i]
		if condition.Type != conditionType {
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
			Type:               conditionType,
			Status:             status,
			ObservedGeneration: generation,
			LastTransitionTime: transitionTime,
			Reason:             reason,
			Message:            message,
		}
		return
	}
	*conditions = append(*conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

func isStalledReason(reason string) bool {
	switch reason {
	case "ConfigurationInvalid", "InvalidSpec", "RoleConfigurationInvalid", "CreationNotAllowed",
		"ExternalProjectMismatch", "ExternalIdentityMismatch", "ExternalOrganizationMismatch", "ExternalAuthMismatch":
		return true
	default:
		return false
	}
}

func conditionReady(conditions []metav1.Condition) bool {
	for _, condition := range conditions {
		if condition.Type == readyCondition && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

func persistStatus(ctx context.Context, kubeClient client.Client, object client.Object, before client.Object) error {
	if reflect.DeepEqual(before, object) {
		return nil
	}
	// The status subresource patch does not carry an optimistic resource-version
	// precondition, so unrelated spec and metadata updates are preserved.
	return kubeClient.Status().Patch(ctx, object, client.MergeFrom(before))
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

func identityRoleSlugs(spec []string) ([]string, error) {
	if len(spec) == 0 {
		return nil, nil
	}

	roleSlugs := append([]string(nil), spec...)
	seen := make(map[string]struct{}, len(roleSlugs))
	for _, slug := range roleSlugs {
		if strings.TrimSpace(slug) == "" {
			return nil, errors.New("roleSlugs cannot contain an empty value")
		}
		switch slug {
		case infisicalAdminRole, "member", "viewer", infisicalNoAccessRole:
		default:
			return nil, fmt.Errorf("project role %q is not a built-in free-tier role; custom roles are not supported", slug)
		}
		if _, exists := seen[slug]; exists {
			return nil, fmt.Errorf("roleSlugs contains duplicate value %q", slug)
		}
		seen[slug] = struct{}{}
	}
	sort.Strings(roleSlugs)
	return roleSlugs, nil
}

func identityRoleSlugsEqual(want []string, current []infisicalclient.IdentityMembershipRole) bool {
	observed := make([]string, 0, len(current))
	for _, role := range current {
		if role.IsTemporary {
			return false
		}
		observed = append(observed, role.Slug())
	}
	sort.Strings(observed)
	return reflect.DeepEqual(want, observed)
}

func identityRoleStatusesFrom(roles []infisicalclient.IdentityMembershipRole) []infisicalv1alpha1.IdentityRoleStatus {
	observed := make([]infisicalv1alpha1.IdentityRoleStatus, 0, len(roles))
	for _, role := range roles {
		observed = append(observed, infisicalv1alpha1.IdentityRoleStatus{
			RoleID:      role.ID,
			Slug:        role.Slug(),
			Name:        role.Name(),
			IsTemporary: role.IsTemporary,
		})
	}
	sort.Slice(observed, func(i, j int) bool { return observed[i].Slug < observed[j].Slug })
	return observed
}
