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
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	infisicalv1alpha1 "github.com/winrarr/infisical-entity-operator/api/infisical/v1alpha1"
	"github.com/winrarr/infisical-entity-operator/internal/infisicalclient"
)

const universalAuthClientSecretIDAnnotation = "infisical.infisical-operator.io/client-secret-id"

// InfisicalUniversalAuthReconciler reconciles an Infisical Universal Auth method and its
// one-time client secret publication.
type InfisicalUniversalAuthReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicaluniversalauths,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicaluniversalauths/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicaluniversalauths/finalizers,verbs=update
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalidentities,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (r *InfisicalUniversalAuthReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var auth infisicalv1alpha1.InfisicalUniversalAuth
	if err := r.Get(ctx, req.NamespacedName, &auth); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if auth.DeletionTimestamp.IsZero() && deletionPolicy(auth.Spec.DeletionPolicy) == infisicalv1alpha1.DeletionPolicyDelete {
		if added, err := ensureFinalizer(ctx, r.Client, &auth); err != nil {
			return ctrl.Result{}, err
		} else if added {
			return ctrl.Result{}, nil
		}
	}
	if !auth.DeletionTimestamp.IsZero() {
		return r.reconcileUniversalAuthDeletion(ctx, &auth)
	}

	before := auth.DeepCopy()
	identity, err := r.universalAuthIdentity(ctx, &auth)
	if err != nil {
		return r.universalAuthError(ctx, &auth, "IdentityNotReady", err)
	}
	apiClient, err := infisicalClientForConnection(ctx, r.Client, auth.Namespace, auth.Spec.ConnectionRef)
	if err != nil {
		return r.universalAuthError(ctx, &auth, "ConnectionNotReady", err)
	}
	desired := universalAuthConfigRequest(&auth)

	if auth.Status.AuthID == "" && canAdopt(auth.Spec.CreationPolicy) {
		adopted, findErr := apiClient.GetUniversalAuth(ctx, identity.Status.IdentityID)
		if findErr != nil && !infisicalclient.IsNotFound(findErr) {
			return r.universalAuthError(ctx, &auth, "ExternalReadFailed", findErr)
		}
		if adopted != nil {
			if err := validateUniversalAuthOwnership(adopted, identity.Status.IdentityID); err != nil {
				return r.universalAuthError(ctx, &auth, "ExternalAuthMismatch", err)
			}
			r.setUniversalAuthObservedState(&auth, adopted)
		} else if !canCreate(auth.Spec.CreationPolicy) {
			return r.universalAuthError(ctx, &auth, "RemoteAuthMissing", newDependencyError("Universal Auth is not attached and creationPolicy is Adopt"))
		}
	}

	if auth.Status.AuthID == "" {
		if !canCreate(auth.Spec.CreationPolicy) {
			return r.universalAuthError(ctx, &auth, "CreationNotAllowed", newDependencyError("Universal Auth is not attached and creationPolicy is Adopt"))
		}
		attached, attachErr := apiClient.AttachUniversalAuth(ctx, identity.Status.IdentityID, desired)
		if attachErr != nil {
			return r.universalAuthError(ctx, &auth, "ExternalCreateFailed", attachErr)
		}
		r.setUniversalAuthObservedState(&auth, attached)
	}

	current, err := apiClient.GetUniversalAuth(ctx, identity.Status.IdentityID)
	if err != nil {
		if infisicalclient.IsNotFound(err) {
			before = auth.DeepCopy()
			clearUniversalAuthStatus(&auth)
			setCondition(&auth.Status.Conditions, auth.Generation, metav1.ConditionFalse, "RemoteAuthMissing", "Universal Auth is no longer attached; it will be recreated according to creationPolicy")
			return ctrl.Result{RequeueAfter: externalRetry}, persistStatus(ctx, r.Client, &auth, before)
		}
		return r.universalAuthError(ctx, &auth, "ExternalReadFailed", err)
	}
	if err := validateUniversalAuthOwnership(current, identity.Status.IdentityID); err != nil {
		return r.universalAuthError(ctx, &auth, "ExternalAuthMismatch", err)
	}
	if universalAuthConfigNeedsUpdate(&auth, current) {
		updated, updateErr := apiClient.UpdateUniversalAuth(ctx, identity.Status.IdentityID, desired)
		if updateErr != nil {
			return r.universalAuthError(ctx, &auth, "ExternalUpdateFailed", updateErr)
		}
		current = updated
	}
	r.setUniversalAuthObservedState(&auth, current)

	if err := r.ensureUniversalAuthClientSecret(ctx, &auth, apiClient, current); err != nil {
		return r.universalAuthError(ctx, &auth, "ClientSecretNotReady", err)
	}
	setCondition(&auth.Status.Conditions, auth.Generation, metav1.ConditionTrue, "Ready", "Infisical Universal Auth is reconciled and its client Secret is published")
	return ctrl.Result{RequeueAfter: driftDetectionEvery}, persistStatus(ctx, r.Client, &auth, before)
}

func (r *InfisicalUniversalAuthReconciler) universalAuthIdentity(ctx context.Context, auth *infisicalv1alpha1.InfisicalUniversalAuth) (*infisicalv1alpha1.InfisicalIdentity, error) {
	var identity infisicalv1alpha1.InfisicalIdentity
	if err := r.Get(ctx, client.ObjectKey{Namespace: auth.Namespace, Name: auth.Spec.IdentityRef.Name}, &identity); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, newDependencyError("InfisicalIdentity %s/%s was not found", auth.Namespace, auth.Spec.IdentityRef.Name)
		}
		return nil, err
	}
	if identity.Status.IdentityID == "" || !conditionReady(identity.Status.Conditions) {
		return nil, newDependencyError("InfisicalIdentity %s/%s is not ready", identity.Namespace, identity.Name)
	}
	return &identity, nil
}

func universalAuthConfigRequest(auth *infisicalv1alpha1.InfisicalUniversalAuth) infisicalclient.UniversalAuthConfigRequest {
	request := infisicalclient.UniversalAuthConfigRequest{}
	if auth.Spec.Config == nil {
		return request
	}
	config := auth.Spec.Config
	request.ClientSecretTrustedIPs = universalAuthTrustedIPsFrom(config.ClientSecretTrustedIPs)
	request.AccessTokenTrustedIPs = universalAuthTrustedIPsFrom(config.AccessTokenTrustedIPs)
	request.AccessTokenTTL = config.AccessTokenTTL
	request.AccessTokenMaxTTL = config.AccessTokenMaxTTL
	request.AccessTokenNumUsesLimit = config.AccessTokenNumUsesLimit
	request.AccessTokenPeriod = config.AccessTokenPeriod
	request.LockoutEnabled = config.LockoutEnabled
	request.LockoutThreshold = config.LockoutThreshold
	request.LockoutDurationSeconds = config.LockoutDurationSeconds
	request.LockoutCounterResetSeconds = config.LockoutCounterResetSeconds
	return request
}

func universalAuthTrustedIPsFrom(values []infisicalv1alpha1.UniversalAuthTrustedIP) []infisicalclient.UniversalAuthTrustedIP {
	if values == nil {
		return nil
	}
	result := make([]infisicalclient.UniversalAuthTrustedIP, len(values))
	for i, value := range values {
		result[i] = infisicalclient.UniversalAuthTrustedIP{IPAddress: value.IPAddress}
	}
	return result
}

func universalAuthConfigNeedsUpdate(auth *infisicalv1alpha1.InfisicalUniversalAuth, current *infisicalclient.UniversalAuthConfig) bool {
	if auth.Spec.Config == nil {
		return false
	}
	desired := universalAuthConfigRequest(auth)
	if desired.ClientSecretTrustedIPs != nil && !reflect.DeepEqual(desired.ClientSecretTrustedIPs, current.ClientSecretTrustedIPs) {
		return true
	}
	if desired.AccessTokenTrustedIPs != nil && !reflect.DeepEqual(desired.AccessTokenTrustedIPs, current.AccessTokenTrustedIPs) {
		return true
	}
	return optionalInt64Diff(desired.AccessTokenTTL, current.AccessTokenTTL) ||
		optionalInt64Diff(desired.AccessTokenMaxTTL, current.AccessTokenMaxTTL) ||
		optionalInt64Diff(desired.AccessTokenNumUsesLimit, current.AccessTokenNumUsesLimit) ||
		optionalInt64Diff(desired.AccessTokenPeriod, current.AccessTokenPeriod) ||
		optionalBoolDiff(desired.LockoutEnabled, current.LockoutEnabled) ||
		optionalInt64Diff(desired.LockoutThreshold, current.LockoutThreshold) ||
		optionalInt64Diff(desired.LockoutDurationSeconds, current.LockoutDurationSeconds) ||
		optionalInt64Diff(desired.LockoutCounterResetSeconds, current.LockoutCounterResetSeconds)
}

func optionalInt64Diff(desired *int64, current int64) bool {
	return desired != nil && *desired != current
}

func optionalBoolDiff(desired *bool, current bool) bool {
	return desired != nil && *desired != current
}

func validateUniversalAuthOwnership(config *infisicalclient.UniversalAuthConfig, identityID string) error {
	if config.IdentityID != identityID {
		return newDependencyError("Infisical Universal Auth belongs to identity %q, want %q", config.IdentityID, identityID)
	}
	return nil
}

func (r *InfisicalUniversalAuthReconciler) setUniversalAuthObservedState(auth *infisicalv1alpha1.InfisicalUniversalAuth, observed *infisicalclient.UniversalAuthConfig) {
	auth.Status.AuthID = observed.ID
	auth.Status.IdentityID = observed.IdentityID
	auth.Status.ClientID = observed.ClientID
	auth.Status.Config = universalAuthConfigStatus(observed)
	auth.Status.ObservedGeneration = auth.Generation
}

func universalAuthConfigStatus(config *infisicalclient.UniversalAuthConfig) infisicalv1alpha1.UniversalAuthConfigStatus {
	return infisicalv1alpha1.UniversalAuthConfigStatus{
		ClientSecretTrustedIPs:     universalAuthTrustedIPsTo(config.ClientSecretTrustedIPs),
		AccessTokenTrustedIPs:      universalAuthTrustedIPsTo(config.AccessTokenTrustedIPs),
		AccessTokenTTL:             config.AccessTokenTTL,
		AccessTokenMaxTTL:          config.AccessTokenMaxTTL,
		AccessTokenNumUsesLimit:    config.AccessTokenNumUsesLimit,
		AccessTokenPeriod:          config.AccessTokenPeriod,
		LockoutEnabled:             config.LockoutEnabled,
		LockoutThreshold:           config.LockoutThreshold,
		LockoutDurationSeconds:     config.LockoutDurationSeconds,
		LockoutCounterResetSeconds: config.LockoutCounterResetSeconds,
	}
}

func universalAuthTrustedIPsTo(values []infisicalclient.UniversalAuthTrustedIP) []infisicalv1alpha1.UniversalAuthTrustedIP {
	if values == nil {
		return nil
	}
	result := make([]infisicalv1alpha1.UniversalAuthTrustedIP, len(values))
	for i, value := range values {
		result[i] = infisicalv1alpha1.UniversalAuthTrustedIP{IPAddress: value.IPAddress}
	}
	return result
}

func (r *InfisicalUniversalAuthReconciler) ensureUniversalAuthClientSecret(ctx context.Context, auth *infisicalv1alpha1.InfisicalUniversalAuth, apiClient *infisicalclient.Client, config *infisicalclient.UniversalAuthConfig) error {
	secretName := auth.Spec.ClientSecret.SecretRef.Name
	if strings.TrimSpace(secretName) == "" {
		return fmt.Errorf("spec.clientSecret.secretRef.name is required")
	}
	clientSecretID := auth.Status.ClientSecret.ClientSecretID
	needsRotation := clientSecretID == "" || auth.Status.ClientSecret.LastRotationNonce != auth.Spec.ClientSecret.RotationNonce
	var published corev1.Secret
	secretErr := r.Get(ctx, client.ObjectKey{Namespace: auth.Namespace, Name: secretName}, &published)
	if apierrors.IsNotFound(secretErr) {
		needsRotation = true
	} else if secretErr != nil {
		return secretErr
	} else if clientSecretID == "" {
		clientSecretID = published.Annotations[universalAuthClientSecretIDAnnotation]
		if clientSecretID != "" && len(published.Data[universalAuthSecretKey(auth.Spec.ClientSecret.SecretRef, false)]) > 0 && len(published.Data[universalAuthSecretKey(auth.Spec.ClientSecret.SecretRef, true)]) > 0 {
			metadata, err := apiClient.GetUniversalAuthClientSecret(ctx, config.IdentityID, clientSecretID)
			if err == nil && !metadata.IsClientSecretRevoked {
				needsRotation = false
				auth.Status.ClientSecret.ClientSecretID = metadata.ID
				auth.Status.ClientSecret.ClientSecretPrefix = metadata.ClientSecretPrefix
				auth.Status.ClientSecret.SecretName = secretName
				auth.Status.ClientSecret.LastRotationNonce = auth.Spec.ClientSecret.RotationNonce
			}
		}
	}
	if !needsRotation && clientSecretID != "" {
		metadata, err := apiClient.GetUniversalAuthClientSecret(ctx, config.IdentityID, clientSecretID)
		if err != nil && !infisicalclient.IsNotFound(err) {
			return err
		}
		if err == nil && !metadata.IsClientSecretRevoked {
			auth.Status.ClientSecret.SecretName = secretName
			auth.Status.ClientSecret.ClientSecretID = metadata.ID
			auth.Status.ClientSecret.ClientSecretPrefix = metadata.ClientSecretPrefix
			return nil
		}
		needsRotation = true
	}
	if !needsRotation {
		return nil
	}

	created, err := apiClient.CreateUniversalAuthClientSecret(ctx, config.IdentityID, infisicalclient.CreateUniversalAuthClientSecretRequest{
		Description:  auth.Spec.ClientSecret.Description,
		NumUsesLimit: auth.Spec.ClientSecret.NumUsesLimit,
		TTL:          auth.Spec.ClientSecret.TTL,
	})
	if err != nil {
		return err
	}
	if err := r.publishUniversalAuthSecret(ctx, auth, created.ClientSecret, config.ClientID, created.ClientSecretData.ID); err != nil {
		return err
	}
	if oldID := auth.Status.ClientSecret.ClientSecretID; oldID != "" && oldID != created.ClientSecretData.ID {
		if err := apiClient.RevokeUniversalAuthClientSecret(ctx, config.IdentityID, oldID); err != nil && !infisicalclient.IsNotFound(err) {
			return fmt.Errorf("revoke previous Universal Auth client secret: %w", err)
		}
	}
	rotatedAt := metav1.Now()
	auth.Status.ClientSecret = infisicalv1alpha1.UniversalAuthClientSecretStatus{
		SecretName:         secretName,
		ClientSecretID:     created.ClientSecretData.ID,
		ClientSecretPrefix: created.ClientSecretData.ClientSecretPrefix,
		LastRotationNonce:  auth.Spec.ClientSecret.RotationNonce,
		LastRotatedAt:      &rotatedAt,
	}
	return nil
}

func universalAuthSecretKey(ref infisicalv1alpha1.UniversalAuthSecretReference, clientSecret bool) string {
	if clientSecret {
		if ref.ClientSecretKey != "" {
			return ref.ClientSecretKey
		}
		return "clientSecret"
	}
	if ref.ClientIDKey != "" {
		return ref.ClientIDKey
	}
	return "clientId"
}

func (r *InfisicalUniversalAuthReconciler) publishUniversalAuthSecret(ctx context.Context, auth *infisicalv1alpha1.InfisicalUniversalAuth, clientSecret, clientID, clientSecretID string) error {
	secretName := auth.Spec.ClientSecret.SecretRef.Name
	var secret corev1.Secret
	err := r.Get(ctx, client.ObjectKey{Namespace: auth.Namespace, Name: secretName}, &secret)
	if apierrors.IsNotFound(err) {
		secret = corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: auth.Namespace}}
	} else if err != nil {
		return err
	} else if owner := metav1.GetControllerOf(&secret); owner != nil && !ownerReferenceMatches(owner, auth) {
		return newDependencyError("Secret %s/%s is controlled by another resource", secret.Namespace, secret.Name)
	}
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}
	secret.Type = corev1.SecretTypeOpaque
	secret.Data[universalAuthSecretKey(auth.Spec.ClientSecret.SecretRef, false)] = []byte(clientID)
	secret.Data[universalAuthSecretKey(auth.Spec.ClientSecret.SecretRef, true)] = []byte(clientSecret)
	if secret.Annotations == nil {
		secret.Annotations = map[string]string{}
	}
	secret.Annotations[universalAuthClientSecretIDAnnotation] = clientSecretID
	if err := r.setUniversalAuthSecretOwner(&secret, auth); err != nil {
		return err
	}
	if secret.ResourceVersion == "" {
		return r.Create(ctx, &secret)
	}
	return r.Update(ctx, &secret)
}

func (r *InfisicalUniversalAuthReconciler) setUniversalAuthSecretOwner(secret *corev1.Secret, auth *infisicalv1alpha1.InfisicalUniversalAuth) error {
	if r.Scheme != nil {
		return controllerutil.SetControllerReference(auth, secret, r.Scheme)
	}
	controller := true
	blockOwnerDeletion := true
	secret.OwnerReferences = []metav1.OwnerReference{{
		APIVersion:         infisicalv1alpha1.GroupVersion.String(),
		Kind:               "InfisicalUniversalAuth",
		Name:               auth.Name,
		UID:                auth.UID,
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}}
	return nil
}

func ownerReferenceMatches(owner *metav1.OwnerReference, auth *infisicalv1alpha1.InfisicalUniversalAuth) bool {
	return owner.APIVersion == infisicalv1alpha1.GroupVersion.String() && owner.Kind == "InfisicalUniversalAuth" && owner.Name == auth.Name && (owner.UID == "" || auth.UID == "" || owner.UID == auth.UID)
}

func clearUniversalAuthStatus(auth *infisicalv1alpha1.InfisicalUniversalAuth) {
	auth.Status.AuthID = ""
	auth.Status.IdentityID = ""
	auth.Status.ClientID = ""
	auth.Status.Config = infisicalv1alpha1.UniversalAuthConfigStatus{}
	auth.Status.ClientSecret = infisicalv1alpha1.UniversalAuthClientSecretStatus{}
}

func (r *InfisicalUniversalAuthReconciler) universalAuthError(ctx context.Context, auth *infisicalv1alpha1.InfisicalUniversalAuth, reason string, err error) (ctrl.Result, error) {
	before := auth.DeepCopy()
	auth.Status.ObservedGeneration = auth.Generation
	setCondition(&auth.Status.Conditions, auth.Generation, metav1.ConditionFalse, reason, statusErrorMessage(err))
	return ctrl.Result{RequeueAfter: retryFor(err)}, persistStatus(ctx, r.Client, auth, before)
}

func (r *InfisicalUniversalAuthReconciler) reconcileUniversalAuthDeletion(ctx context.Context, auth *infisicalv1alpha1.InfisicalUniversalAuth) (ctrl.Result, error) {
	if deletionPolicy(auth.Spec.DeletionPolicy) == infisicalv1alpha1.DeletionPolicyOrphan || auth.Status.IdentityID == "" {
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, auth)
	}
	apiClient, err := infisicalClientForConnection(ctx, r.Client, auth.Namespace, auth.Spec.ConnectionRef)
	if err != nil {
		return ctrl.Result{RequeueAfter: retryFor(err)}, err
	}
	if auth.Status.ClientSecret.ClientSecretID != "" {
		if err := apiClient.RevokeUniversalAuthClientSecret(ctx, auth.Status.IdentityID, auth.Status.ClientSecret.ClientSecretID); err != nil && !infisicalclient.IsNotFound(err) {
			return ctrl.Result{RequeueAfter: externalRetry}, err
		}
	}
	if auth.Status.AuthID != "" {
		if err := apiClient.DeleteUniversalAuth(ctx, auth.Status.IdentityID); err != nil && !infisicalclient.IsNotFound(err) {
			return ctrl.Result{RequeueAfter: externalRetry}, err
		}
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, auth)
}

func (r *InfisicalUniversalAuthReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infisicalv1alpha1.InfisicalUniversalAuth{}).
		Watches(&infisicalv1alpha1.InfisicalConnection{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var auths infisicalv1alpha1.InfisicalUniversalAuthList
			if err := mgr.GetClient().List(ctx, &auths, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range auths.Items {
				if auths.Items[i].Spec.ConnectionRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(&auths.Items[i])})
				}
			}
			return requests
		})).
		Watches(&infisicalv1alpha1.InfisicalIdentity{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var auths infisicalv1alpha1.InfisicalUniversalAuthList
			if err := mgr.GetClient().List(ctx, &auths, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range auths.Items {
				if auths.Items[i].Spec.IdentityRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(&auths.Items[i])})
				}
			}
			return requests
		})).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var auths infisicalv1alpha1.InfisicalUniversalAuthList
			if err := mgr.GetClient().List(ctx, &auths, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range auths.Items {
				auth := &auths.Items[i]
				if auth.Spec.ClientSecret.SecretRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(auth)})
				}
			}
			return requests
		})).
		Complete(r)
}
