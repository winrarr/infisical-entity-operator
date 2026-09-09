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
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	infisicalv1alpha1 "github.com/winrarr/infisical-entity-operator/api/infisical/v1alpha1"
	"github.com/winrarr/infisical-entity-operator/internal/infisicalclient"
)

// InfisicalKubernetesAuthReconciler reconciles an Infisical Kubernetes Auth method.
type InfisicalKubernetesAuthReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalkubernetesauths,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalkubernetesauths/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalkubernetesauths/finalizers,verbs=update
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalidentities,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *InfisicalKubernetesAuthReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var auth infisicalv1alpha1.InfisicalKubernetesAuth
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
		return r.reconcileKubernetesAuthDeletion(ctx, &auth)
	}
	before := auth.Status

	var identity infisicalv1alpha1.InfisicalIdentity
	if err := r.Get(ctx, client.ObjectKey{Namespace: auth.Namespace, Name: auth.Spec.IdentityRef.Name}, &identity); err != nil {
		if apierrors.IsNotFound(err) {
			return r.kubernetesAuthError(ctx, &auth, "IdentityNotReady", newDependencyError("InfisicalIdentity %s/%s was not found", auth.Namespace, auth.Spec.IdentityRef.Name))
		}
		return r.kubernetesAuthError(ctx, &auth, "IdentityReadFailed", err)
	}
	if identity.Status.IdentityID == "" {
		return r.kubernetesAuthError(ctx, &auth, "IdentityNotReady", newDependencyError("InfisicalIdentity %s/%s has no observed Infisical identity ID", identity.Namespace, identity.Name))
	}

	apiClient, err := infisicalClientForConnection(ctx, r.Client, auth.Namespace, auth.Spec.ConnectionRef)
	if err != nil {
		return r.kubernetesAuthError(ctx, &auth, "ConnectionNotReady", err)
	}

	if auth.Status.AuthID == "" && canAdopt(auth.Spec.CreationPolicy) {
		adopted, getErr := apiClient.GetKubernetesAuth(ctx, identity.Status.IdentityID)
		if getErr != nil && !infisicalclient.IsNotFound(getErr) {
			return r.kubernetesAuthError(ctx, &auth, "ExternalReadFailed", getErr)
		}
		if adopted != nil {
			r.setKubernetesAuthObservedState(&auth, adopted, identity.Status.IdentityID)
		}
	}

	caCert, tokenReviewerJWT, err := r.kubernetesAuthSecrets(ctx, &auth)
	if err != nil {
		return r.kubernetesAuthError(ctx, &auth, "SecretNotReady", err)
	}

	if auth.Status.AuthID == "" {
		if !canCreate(auth.Spec.CreationPolicy) {
			return r.kubernetesAuthError(ctx, &auth, "CreationNotAllowed", newDependencyError("Kubernetes Auth was not found and creationPolicy is Adopt"))
		}
		created, createErr := apiClient.AttachKubernetesAuth(ctx, identity.Status.IdentityID, kubernetesAuthRequestFrom(&auth, caCert, tokenReviewerJWT))
		if createErr != nil {
			return r.kubernetesAuthError(ctx, &auth, "ExternalCreateFailed", createErr)
		}
		r.setKubernetesAuthObservedState(&auth, created, identity.Status.IdentityID)
	}

	current, err := apiClient.GetKubernetesAuth(ctx, identity.Status.IdentityID)
	if err != nil {
		if infisicalclient.IsNotFound(err) {
			before := auth.Status
			auth.Status.AuthID = ""
			auth.Status.ObservedGeneration = auth.Generation
			setCondition(&auth.Status.Conditions, auth.Generation, "False", "RemoteKubernetesAuthMissing", "the Kubernetes Auth method no longer exists in Infisical; it will be recreated according to creationPolicy")
			return ctrl.Result{RequeueAfter: externalRetry}, persistStatus(ctx, r.Client, &auth, before, auth.Status)
		}
		return r.kubernetesAuthError(ctx, &auth, "ExternalReadFailed", err)
	}

	if kubernetesAuthNeedsUpdate(&auth, current, caCert, tokenReviewerJWT) {
		updated, updateErr := apiClient.UpdateKubernetesAuth(ctx, identity.Status.IdentityID, kubernetesAuthPatchFrom(&auth, caCert, tokenReviewerJWT))
		if updateErr != nil {
			return r.kubernetesAuthError(ctx, &auth, "ExternalUpdateFailed", updateErr)
		}
		current = updated
	}

	r.setKubernetesAuthObservedState(&auth, current, identity.Status.IdentityID)
	setCondition(&auth.Status.Conditions, auth.Generation, "True", "Ready", "Infisical Kubernetes Auth is reconciled")
	return ctrl.Result{RequeueAfter: driftDetectionEvery}, persistStatus(ctx, r.Client, &auth, before, auth.Status)
}

func (r *InfisicalKubernetesAuthReconciler) kubernetesAuthSecrets(ctx context.Context, auth *infisicalv1alpha1.InfisicalKubernetesAuth) (string, string, error) {
	caCert, err := optionalSecretValueFromReference(ctx, r.Client, auth.Namespace, auth.Spec.CACertSecretRef, "ca.crt", "Kubernetes CA certificate")
	if err != nil {
		return "", "", err
	}
	tokenReviewerJWT, err := optionalSecretValueFromReference(ctx, r.Client, auth.Namespace, auth.Spec.TokenReviewerJWTSecretRef, "token", "Kubernetes token reviewer JWT")
	if err != nil {
		return "", "", err
	}
	return caCert, tokenReviewerJWT, nil
}

func kubernetesAuthRequestFrom(auth *infisicalv1alpha1.InfisicalKubernetesAuth, caCert, tokenReviewerJWT string) infisicalclient.CreateKubernetesAuthRequest {
	return infisicalclient.CreateKubernetesAuthRequest{
		KubernetesHost:          auth.Spec.KubernetesHost,
		CACert:                  caCert,
		VerifyTLSCertificate:    auth.Spec.VerifyTLSCertificate,
		TokenReviewerJWT:        tokenReviewerJWT,
		TokenReviewMode:         string(kubernetesAuthMode(auth.Spec.TokenReviewMode)),
		AllowedNamespaces:       joinCSV(auth.Spec.AllowedNamespaces),
		AllowedNames:            joinCSV(auth.Spec.AllowedNames),
		AllowedAudience:         auth.Spec.AllowedAudience,
		GatewayID:               auth.Spec.GatewayID,
		GatewayPoolID:           auth.Spec.GatewayPoolID,
		AccessTokenTrustedIPs:   kubernetesTrustedIPsFrom(auth.Spec.AccessTokenTrustedIPs),
		AccessTokenTTL:          auth.Spec.AccessTokenTTL,
		AccessTokenMaxTTL:       auth.Spec.AccessTokenMaxTTL,
		AccessTokenNumUsesLimit: auth.Spec.AccessTokenNumUsesLimit,
	}
}

func kubernetesAuthPatchFrom(auth *infisicalv1alpha1.InfisicalKubernetesAuth, caCert, tokenReviewerJWT string) infisicalclient.KubernetesAuthPatch {
	mode := string(kubernetesAuthMode(auth.Spec.TokenReviewMode))
	allowedNamespaces := joinCSV(auth.Spec.AllowedNamespaces)
	allowedNames := joinCSV(auth.Spec.AllowedNames)
	patch := infisicalclient.KubernetesAuthPatch{
		AllowedNamespaces:       &allowedNamespaces,
		AllowedNames:            &allowedNames,
		TokenReviewMode:         &mode,
		AccessTokenTTL:          auth.Spec.AccessTokenTTL,
		AccessTokenMaxTTL:       auth.Spec.AccessTokenMaxTTL,
		AccessTokenNumUsesLimit: auth.Spec.AccessTokenNumUsesLimit,
	}
	if auth.Spec.KubernetesHost != "" {
		patch.KubernetesHost = &auth.Spec.KubernetesHost
	}
	if auth.Spec.CACertSecretRef != nil {
		patch.CACert = &caCert
	}
	if auth.Spec.VerifyTLSCertificate != nil {
		patch.VerifyTLSCertificate = auth.Spec.VerifyTLSCertificate
	}
	if auth.Spec.TokenReviewerJWTSecretRef != nil {
		patch.TokenReviewerJWT = &tokenReviewerJWT
	}
	if auth.Spec.AllowedAudience != "" {
		patch.AllowedAudience = &auth.Spec.AllowedAudience
	}
	if auth.Spec.GatewayID != "" {
		patch.GatewayID = &auth.Spec.GatewayID
	}
	if auth.Spec.GatewayPoolID != "" {
		patch.GatewayPoolID = &auth.Spec.GatewayPoolID
	}
	if auth.Spec.AccessTokenTrustedIPs != nil {
		trustedIPs := kubernetesTrustedIPsFrom(auth.Spec.AccessTokenTrustedIPs)
		patch.AccessTokenTrustedIPs = &trustedIPs
	}
	return patch
}

func kubernetesAuthMode(mode infisicalv1alpha1.KubernetesTokenReviewMode) infisicalv1alpha1.KubernetesTokenReviewMode {
	if mode == "" {
		return infisicalv1alpha1.KubernetesTokenReviewModeAPI
	}
	return mode
}

func kubernetesTrustedIPsFrom(ips []infisicalv1alpha1.KubernetesTrustedIP) []infisicalclient.TrustedIP {
	if ips == nil {
		return nil
	}
	result := make([]infisicalclient.TrustedIP, 0, len(ips))
	for _, ip := range ips {
		result = append(result, infisicalclient.TrustedIP{IPAddress: ip.IPAddress})
	}
	return result
}

func kubernetesTrustedIPsTo(ips []infisicalclient.TrustedIP) []infisicalv1alpha1.KubernetesTrustedIP {
	if ips == nil {
		return nil
	}
	result := make([]infisicalv1alpha1.KubernetesTrustedIP, 0, len(ips))
	for _, ip := range ips {
		result = append(result, infisicalv1alpha1.KubernetesTrustedIP{IPAddress: ip.IPAddress})
	}
	return result
}

func kubernetesAuthNeedsUpdate(auth *infisicalv1alpha1.InfisicalKubernetesAuth, current *infisicalclient.KubernetesAuth, caCert, tokenReviewerJWT string) bool {
	if current.AllowedNamespaces != joinCSV(auth.Spec.AllowedNamespaces) || current.AllowedNames != joinCSV(auth.Spec.AllowedNames) {
		return true
	}
	if auth.Spec.KubernetesHost != "" && current.KubernetesHost != auth.Spec.KubernetesHost {
		return true
	}
	if auth.Spec.AllowedAudience != "" && current.AllowedAudience != auth.Spec.AllowedAudience {
		return true
	}
	if auth.Spec.TokenReviewMode != "" && current.TokenReviewMode != string(kubernetesAuthMode(auth.Spec.TokenReviewMode)) {
		return true
	}
	if auth.Spec.GatewayID != "" && current.GatewayID != auth.Spec.GatewayID {
		return true
	}
	if auth.Spec.GatewayPoolID != "" && current.GatewayPoolID != auth.Spec.GatewayPoolID {
		return true
	}
	if auth.Spec.VerifyTLSCertificate != nil && current.VerifyTLSCertificate != *auth.Spec.VerifyTLSCertificate {
		return true
	}
	if auth.Spec.CACertSecretRef != nil && current.CACert != caCert {
		return true
	}
	if auth.Spec.TokenReviewerJWTSecretRef != nil && current.TokenReviewerJWT != tokenReviewerJWT {
		return true
	}
	if auth.Spec.AccessTokenTrustedIPs != nil && !reflect.DeepEqual(current.AccessTokenTrustedIPs, kubernetesTrustedIPsFrom(auth.Spec.AccessTokenTrustedIPs)) {
		return true
	}
	if auth.Spec.AccessTokenTTL != nil && current.AccessTokenTTL != *auth.Spec.AccessTokenTTL {
		return true
	}
	if auth.Spec.AccessTokenMaxTTL != nil && current.AccessTokenMaxTTL != *auth.Spec.AccessTokenMaxTTL {
		return true
	}
	return auth.Spec.AccessTokenNumUsesLimit != nil && current.AccessTokenNumUsesLimit != *auth.Spec.AccessTokenNumUsesLimit
}

func (r *InfisicalKubernetesAuthReconciler) setKubernetesAuthObservedState(auth *infisicalv1alpha1.InfisicalKubernetesAuth, observed *infisicalclient.KubernetesAuth, identityID string) {
	auth.Status.AuthID = observed.ID
	auth.Status.IdentityID = identityID
	auth.Status.KubernetesHost = observed.KubernetesHost
	auth.Status.AllowedNamespaces = splitCSV(observed.AllowedNamespaces)
	auth.Status.AllowedNames = splitCSV(observed.AllowedNames)
	auth.Status.AllowedAudience = observed.AllowedAudience
	auth.Status.TokenReviewMode = infisicalv1alpha1.KubernetesTokenReviewMode(observed.TokenReviewMode)
	auth.Status.GatewayID = observed.GatewayID
	auth.Status.GatewayPoolID = observed.GatewayPoolID
	auth.Status.VerifyTLSCertificate = observed.VerifyTLSCertificate
	auth.Status.HasCACertificate = observed.CACert != ""
	auth.Status.HasTokenReviewerJWT = observed.TokenReviewerJWT != ""
	auth.Status.AccessTokenTrustedIPs = kubernetesTrustedIPsTo(observed.AccessTokenTrustedIPs)
	auth.Status.AccessTokenTTL = observed.AccessTokenTTL
	auth.Status.AccessTokenMaxTTL = observed.AccessTokenMaxTTL
	auth.Status.AccessTokenNumUsesLimit = observed.AccessTokenNumUsesLimit
	auth.Status.ObservedGeneration = auth.Generation
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func sortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func joinCSV(values []string) string {
	return strings.Join(sortedStrings(values), ",")
}

func (r *InfisicalKubernetesAuthReconciler) kubernetesAuthError(ctx context.Context, auth *infisicalv1alpha1.InfisicalKubernetesAuth, reason string, err error) (ctrl.Result, error) {
	before := auth.Status
	auth.Status.ObservedGeneration = auth.Generation
	setCondition(&auth.Status.Conditions, auth.Generation, "False", reason, statusErrorMessage(err))
	return ctrl.Result{RequeueAfter: retryFor(err)}, persistStatus(ctx, r.Client, auth, before, auth.Status)
}

func (r *InfisicalKubernetesAuthReconciler) reconcileKubernetesAuthDeletion(ctx context.Context, auth *infisicalv1alpha1.InfisicalKubernetesAuth) (ctrl.Result, error) {
	if deletionPolicy(auth.Spec.DeletionPolicy) == infisicalv1alpha1.DeletionPolicyOrphan || auth.Status.IdentityID == "" {
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, auth)
	}

	apiClient, err := infisicalClientForConnection(ctx, r.Client, auth.Namespace, auth.Spec.ConnectionRef)
	if err != nil {
		return ctrl.Result{RequeueAfter: retryFor(err)}, err
	}
	if err := apiClient.DeleteKubernetesAuth(ctx, auth.Status.IdentityID); err != nil && !infisicalclient.IsNotFound(err) {
		return ctrl.Result{RequeueAfter: externalRetry}, err
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, auth)
}

func (r *InfisicalKubernetesAuthReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infisicalv1alpha1.InfisicalKubernetesAuth{}).
		Watches(&infisicalv1alpha1.InfisicalConnection{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var auths infisicalv1alpha1.InfisicalKubernetesAuthList
			if err := mgr.GetClient().List(ctx, &auths, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range auths.Items {
				auth := &auths.Items[i]
				if auth.Spec.ConnectionRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(auth)})
				}
			}
			return requests
		})).
		Watches(&infisicalv1alpha1.InfisicalIdentity{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var auths infisicalv1alpha1.InfisicalKubernetesAuthList
			if err := mgr.GetClient().List(ctx, &auths, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range auths.Items {
				auth := &auths.Items[i]
				if auth.Spec.IdentityRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(auth)})
				}
			}
			return requests
		})).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var auths infisicalv1alpha1.InfisicalKubernetesAuthList
			if err := mgr.GetClient().List(ctx, &auths, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range auths.Items {
				auth := &auths.Items[i]
				var connection infisicalv1alpha1.InfisicalConnection
				if err := mgr.GetClient().Get(ctx, client.ObjectKey{Namespace: auth.Namespace, Name: auth.Spec.ConnectionRef.Name}, &connection); err != nil {
					continue
				}
				if connection.Spec.AuthSecretRef.Name == object.GetName() || (auth.Spec.CACertSecretRef != nil && auth.Spec.CACertSecretRef.Name == object.GetName()) || (auth.Spec.TokenReviewerJWTSecretRef != nil && auth.Spec.TokenReviewerJWTSecretRef.Name == object.GetName()) {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(auth)})
				}
			}
			return requests
		})).
		Complete(r)
}
