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
	"sort"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	infisicalv1alpha1 "github.com/winrarr/infisical-entity-operator/api/infisical/v1alpha1"
	"github.com/winrarr/infisical-entity-operator/internal/infisicalclient"
)

// InfisicalIdentityTemplateReconciler reconciles an Infisical identity authentication template.
type InfisicalIdentityTemplateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalidentitytemplates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalidentitytemplates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalidentitytemplates/finalizers,verbs=update
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups=infisical.infisical-operator.io,resources=infisicalorganizations,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *InfisicalIdentityTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var template infisicalv1alpha1.InfisicalIdentityTemplate
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
		return r.reconcileIdentityTemplateDeletion(ctx, &template)
	}
	before := template.DeepCopy()
	expectedOrganizationID, err := r.expectedIdentityTemplateOrganizationID(ctx, &template)
	if err != nil {
		return r.identityTemplateError(ctx, &template, "OrganizationNotReady", err)
	}

	apiClient, err := infisicalClientForConnection(ctx, r.Client, template.Namespace, template.Spec.ConnectionRef)
	if err != nil {
		return r.identityTemplateError(ctx, &template, "ConnectionNotReady", err)
	}
	fields, err := r.identityTemplateFields(ctx, &template)
	if err != nil {
		return r.identityTemplateError(ctx, &template, "SecretNotReady", err)
	}
	secretVersions, err := r.identityTemplateSecretResourceVersions(ctx, &template)
	if err != nil {
		return r.identityTemplateError(ctx, &template, "SecretNotReady", err)
	}

	if template.Status.TemplateID == "" && canAdopt(template.Spec.CreationPolicy) {
		adopted, findErr := apiClient.FindIdentityTemplate(ctx, identityTemplateName(&template))
		if findErr != nil {
			return r.identityTemplateError(ctx, &template, "ExternalReadFailed", findErr)
		}
		if adopted != nil {
			if err := validateIdentityTemplateOwnership(adopted, expectedOrganizationID, identityTemplateAuthMethod(&template)); err != nil {
				return r.identityTemplateError(ctx, &template, "ExternalTemplateMismatch", err)
			}
			r.setIdentityTemplateObservedState(&template, adopted)
			template.Status.SecretResourceVersions = secretVersions
		}
	}

	if template.Status.TemplateID == "" {
		if !canCreate(template.Spec.CreationPolicy) {
			return r.identityTemplateError(ctx, &template, "CreationNotAllowed", newDependencyError("identity template was not found and creationPolicy is Adopt"))
		}
		created, createErr := apiClient.CreateIdentityTemplate(ctx, infisicalclient.CreateIdentityTemplateRequest{
			Name:           identityTemplateName(&template),
			AuthMethod:     string(identityTemplateAuthMethod(&template)),
			TemplateFields: fields,
		})
		if createErr != nil {
			return r.identityTemplateError(ctx, &template, "ExternalCreateFailed", createErr)
		}
		if err := validateIdentityTemplateOwnership(created, expectedOrganizationID, identityTemplateAuthMethod(&template)); err != nil {
			r.setIdentityTemplateObservedState(&template, created)
			return r.identityTemplateError(ctx, &template, "ExternalTemplateMismatch", err)
		}
		r.setIdentityTemplateObservedState(&template, created)
		template.Status.SecretResourceVersions = secretVersions
	}

	current, err := apiClient.GetIdentityTemplate(ctx, template.Status.TemplateID)
	if err != nil {
		if infisicalclient.IsNotFound(err) {
			before := template.DeepCopy()
			template.Status.TemplateID = ""
			template.Status.Name = ""
			template.Status.OrganizationID = ""
			template.Status.LDAP = nil
			template.Status.Kubernetes = nil
			template.Status.OIDC = nil
			template.Status.SecretResourceVersions = nil
			template.Status.ObservedGeneration = template.Generation
			setCondition(&template.Status.Conditions, template.Generation, "False", "RemoteIdentityTemplateMissing", "the identity template no longer exists in Infisical; it will be recreated according to creationPolicy")
			return ctrl.Result{RequeueAfter: externalRetry}, persistStatus(ctx, r.Client, &template, before)
		}
		return r.identityTemplateError(ctx, &template, "ExternalReadFailed", err)
	}
	if err := validateIdentityTemplateOwnership(current, expectedOrganizationID, identityTemplateAuthMethod(&template)); err != nil {
		return r.identityTemplateError(ctx, &template, "ExternalTemplateMismatch", err)
	}

	secretVersionsChanged := !reflect.DeepEqual(template.Status.SecretResourceVersions, secretVersions)
	if identityTemplateNeedsUpdate(&template, current, fields) || secretVersionsChanged {
		updated, updateErr := apiClient.UpdateIdentityTemplate(ctx, current.ID, infisicalclient.IdentityTemplatePatch{
			Name:           identityTemplateName(&template),
			TemplateFields: fields,
		})
		if updateErr != nil {
			return r.identityTemplateError(ctx, &template, "ExternalUpdateFailed", updateErr)
		}
		current = updated
	}

	r.setIdentityTemplateObservedState(&template, current)
	template.Status.SecretResourceVersions = secretVersions
	setCondition(&template.Status.Conditions, template.Generation, "True", "Ready", "Infisical identity template is reconciled")
	return ctrl.Result{RequeueAfter: driftDetectionEvery}, persistStatus(ctx, r.Client, &template, before)
}

func identityTemplateName(template *infisicalv1alpha1.InfisicalIdentityTemplate) string {
	if template.Spec.TemplateName != "" {
		return template.Spec.TemplateName
	}
	return template.Name
}

func identityTemplateAuthMethod(template *infisicalv1alpha1.InfisicalIdentityTemplate) infisicalv1alpha1.IdentityTemplateAuthMethod {
	return template.Spec.AuthMethod
}

func (r *InfisicalIdentityTemplateReconciler) identityTemplateFields(ctx context.Context, template *infisicalv1alpha1.InfisicalIdentityTemplate) (infisicalclient.IdentityTemplateFields, error) {
	fields := infisicalclient.IdentityTemplateFields{}
	switch template.Spec.AuthMethod {
	case infisicalv1alpha1.IdentityTemplateAuthMethodLDAP:
		if template.Spec.LDAP == nil {
			return fields, fmt.Errorf("ldap template configuration is required")
		}
		bindPass, err := secretValueFromReference(ctx, r.Client, template.Namespace, template.Spec.LDAP.BindPasswordSecretRef, "password", "LDAP bind password")
		if err != nil {
			return fields, err
		}
		fields.URL = template.Spec.LDAP.URL
		fields.BindDN = template.Spec.LDAP.BindDN
		fields.BindPass = bindPass
		fields.SearchBase = template.Spec.LDAP.SearchBase
		if template.Spec.LDAP.CACertSecretRef != nil {
			fields.LDAPCACertificate, err = optionalSecretValueFromReference(ctx, r.Client, template.Namespace, template.Spec.LDAP.CACertSecretRef, "ca.crt", "LDAP CA certificate")
			if err != nil {
				return fields, err
			}
		}
	case infisicalv1alpha1.IdentityTemplateAuthMethodKubernetes:
		if template.Spec.Kubernetes == nil {
			return fields, fmt.Errorf("kubernetes template configuration is required")
		}
		configuration := template.Spec.Kubernetes
		fields.TokenReviewMode = string(configuration.TokenReviewMode)
		fields.KubernetesHost = configuration.KubernetesHost
		fields.VerifyTLSCertificate = configuration.VerifyTLSCertificate
		fields.GatewayID = configuration.GatewayID
		fields.GatewayPoolID = configuration.GatewayPoolID
		fields.AllowedAudience = configuration.AllowedAudience
		var err error
		fields.CACert, err = optionalSecretValueFromReference(ctx, r.Client, template.Namespace, configuration.CACertSecretRef, "ca.crt", "Kubernetes template CA certificate")
		if err != nil {
			return fields, err
		}
		fields.TokenReviewerJWT, err = optionalSecretValueFromReference(ctx, r.Client, template.Namespace, configuration.TokenReviewerJWTSecretRef, "token", "Kubernetes template reviewer JWT")
		if err != nil {
			return fields, err
		}
	case infisicalv1alpha1.IdentityTemplateAuthMethodOIDC:
		if template.Spec.OIDC == nil {
			return fields, fmt.Errorf("oidc template configuration is required")
		}
		configuration := template.Spec.OIDC
		fields.OIDCDiscoveryURL = configuration.OIDCDiscoveryURL
		fields.BoundIssuer = configuration.BoundIssuer
		fields.BoundAudiences = configuration.BoundAudiences
		var err error
		fields.CACert, err = optionalSecretValueFromReference(ctx, r.Client, template.Namespace, configuration.CACertSecretRef, "ca.crt", "OIDC template CA certificate")
		if err != nil {
			return fields, err
		}
	default:
		return fields, fmt.Errorf("unsupported identity template auth method %q", template.Spec.AuthMethod)
	}
	return fields, nil
}

func (r *InfisicalIdentityTemplateReconciler) identityTemplateSecretResourceVersions(ctx context.Context, template *infisicalv1alpha1.InfisicalIdentityTemplate) ([]string, error) {
	refs := make([]*infisicalv1alpha1.SecretKeyReference, 0, 3)
	switch template.Spec.AuthMethod {
	case infisicalv1alpha1.IdentityTemplateAuthMethodLDAP:
		if template.Spec.LDAP != nil {
			refs = append(refs, &template.Spec.LDAP.BindPasswordSecretRef, template.Spec.LDAP.CACertSecretRef)
		}
	case infisicalv1alpha1.IdentityTemplateAuthMethodKubernetes:
		if template.Spec.Kubernetes != nil {
			refs = append(refs, template.Spec.Kubernetes.CACertSecretRef, template.Spec.Kubernetes.TokenReviewerJWTSecretRef)
		}
	case infisicalv1alpha1.IdentityTemplateAuthMethodOIDC:
		if template.Spec.OIDC != nil {
			refs = append(refs, template.Spec.OIDC.CACertSecretRef)
		}
	}
	versions := make([]string, 0, len(refs))
	seen := map[string]struct{}{}
	for _, ref := range refs {
		if ref == nil || ref.Name == "" {
			continue
		}
		if _, ok := seen[ref.Name]; ok {
			continue
		}
		seen[ref.Name] = struct{}{}
		var secret corev1.Secret
		if err := r.Get(ctx, client.ObjectKey{Namespace: template.Namespace, Name: ref.Name}, &secret); err != nil {
			if apierrors.IsNotFound(err) {
				return nil, newDependencyError("identity template Secret %s/%s was not found", template.Namespace, ref.Name)
			}
			return nil, err
		}
		versions = append(versions, fmt.Sprintf("%s=%s", ref.Name, secret.ResourceVersion))
	}
	sort.Strings(versions)
	return versions, nil
}

func identityTemplateNeedsUpdate(template *infisicalv1alpha1.InfisicalIdentityTemplate, current *infisicalclient.IdentityTemplate, desired infisicalclient.IdentityTemplateFields) bool {
	if current.Name != identityTemplateName(template) || current.AuthMethod != string(identityTemplateAuthMethod(template)) {
		return true
	}
	switch template.Spec.AuthMethod {
	case infisicalv1alpha1.IdentityTemplateAuthMethodLDAP:
		configuration := template.Spec.LDAP
		if current.TemplateFields.URL != configuration.URL || current.TemplateFields.BindDN != configuration.BindDN || current.TemplateFields.SearchBase != configuration.SearchBase {
			return true
		}
		if configuration.CACertSecretRef != nil && current.TemplateFields.LDAPCACertificate != desired.LDAPCACertificate {
			return true
		}
		return !current.TemplateFields.HasBindPass
	case infisicalv1alpha1.IdentityTemplateAuthMethodKubernetes:
		configuration := template.Spec.Kubernetes
		if (configuration.TokenReviewMode != "" && current.TemplateFields.TokenReviewMode != desired.TokenReviewMode) ||
			(configuration.KubernetesHost != "" && current.TemplateFields.KubernetesHost != desired.KubernetesHost) ||
			(configuration.AllowedAudience != "" && current.TemplateFields.AllowedAudience != desired.AllowedAudience) ||
			(configuration.GatewayID != "" && current.TemplateFields.GatewayID != desired.GatewayID) ||
			(configuration.GatewayPoolID != "" && current.TemplateFields.GatewayPoolID != desired.GatewayPoolID) ||
			(configuration.VerifyTLSCertificate != nil && boolValue(current.TemplateFields.VerifyTLSCertificate, false) != *configuration.VerifyTLSCertificate) {
			return true
		}
		if configuration.CACertSecretRef != nil && current.TemplateFields.CACert != desired.CACert {
			return true
		}
		return configuration.TokenReviewerJWTSecretRef != nil && !current.TemplateFields.HasTokenReviewerJWT
	case infisicalv1alpha1.IdentityTemplateAuthMethodOIDC:
		configuration := template.Spec.OIDC
		if current.TemplateFields.OIDCDiscoveryURL != configuration.OIDCDiscoveryURL || current.TemplateFields.BoundIssuer != configuration.BoundIssuer || current.TemplateFields.BoundAudiences != configuration.BoundAudiences {
			return true
		}
		return configuration.CACertSecretRef != nil && current.TemplateFields.CACert != desired.CACert
	default:
		return true
	}
}

func validateIdentityTemplateOwnership(template *infisicalclient.IdentityTemplate, organizationID string, authMethod infisicalv1alpha1.IdentityTemplateAuthMethod) error {
	if template.OrganizationID != organizationID {
		return newDependencyError("infisical identity template belongs to organization %q, want %q", template.OrganizationID, organizationID)
	}
	if template.AuthMethod != string(authMethod) {
		return newDependencyError("infisical identity template uses auth method %q, want %q", template.AuthMethod, authMethod)
	}
	return nil
}

func (r *InfisicalIdentityTemplateReconciler) expectedIdentityTemplateOrganizationID(ctx context.Context, template *infisicalv1alpha1.InfisicalIdentityTemplate) (string, error) {
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

func (r *InfisicalIdentityTemplateReconciler) setIdentityTemplateObservedState(template *infisicalv1alpha1.InfisicalIdentityTemplate, observed *infisicalclient.IdentityTemplate) {
	template.Status.TemplateID = observed.ID
	template.Status.Name = observed.Name
	template.Status.AuthMethod = infisicalv1alpha1.IdentityTemplateAuthMethod(observed.AuthMethod)
	template.Status.OrganizationID = observed.OrganizationID
	template.Status.LDAP = nil
	template.Status.Kubernetes = nil
	template.Status.OIDC = nil
	switch observed.AuthMethod {
	case string(infisicalv1alpha1.IdentityTemplateAuthMethodLDAP):
		template.Status.LDAP = &infisicalv1alpha1.IdentityTemplateLDAPStatus{
			URL:              observed.TemplateFields.URL,
			BindDN:           observed.TemplateFields.BindDN,
			SearchBase:       observed.TemplateFields.SearchBase,
			HasCACertificate: observed.TemplateFields.LDAPCACertificate != "",
			HasBindPassword:  observed.TemplateFields.HasBindPass,
		}
	case string(infisicalv1alpha1.IdentityTemplateAuthMethodKubernetes):
		template.Status.Kubernetes = &infisicalv1alpha1.IdentityTemplateKubernetesStatus{
			TokenReviewMode:      infisicalv1alpha1.KubernetesTokenReviewMode(observed.TemplateFields.TokenReviewMode),
			KubernetesHost:       observed.TemplateFields.KubernetesHost,
			AllowedAudience:      observed.TemplateFields.AllowedAudience,
			GatewayID:            observed.TemplateFields.GatewayID,
			GatewayPoolID:        observed.TemplateFields.GatewayPoolID,
			VerifyTLSCertificate: boolValue(observed.TemplateFields.VerifyTLSCertificate, false),
			HasCACertificate:     observed.TemplateFields.CACert != "",
			HasTokenReviewerJWT:  observed.TemplateFields.HasTokenReviewerJWT,
		}
	case string(infisicalv1alpha1.IdentityTemplateAuthMethodOIDC):
		template.Status.OIDC = &infisicalv1alpha1.IdentityTemplateOIDCStatus{
			OIDCDiscoveryURL: observed.TemplateFields.OIDCDiscoveryURL,
			BoundIssuer:      observed.TemplateFields.BoundIssuer,
			BoundAudiences:   observed.TemplateFields.BoundAudiences,
			HasCACertificate: observed.TemplateFields.CACert != "",
		}
	}
	template.Status.ObservedGeneration = template.Generation
}

func (r *InfisicalIdentityTemplateReconciler) identityTemplateError(ctx context.Context, template *infisicalv1alpha1.InfisicalIdentityTemplate, reason string, err error) (ctrl.Result, error) {
	before := template.DeepCopy()
	template.Status.ObservedGeneration = template.Generation
	setCondition(&template.Status.Conditions, template.Generation, "False", reason, statusErrorMessage(err))
	return ctrl.Result{RequeueAfter: retryFor(err)}, persistStatus(ctx, r.Client, template, before)
}

func (r *InfisicalIdentityTemplateReconciler) reconcileIdentityTemplateDeletion(ctx context.Context, template *infisicalv1alpha1.InfisicalIdentityTemplate) (ctrl.Result, error) {
	if deletionPolicy(template.Spec.DeletionPolicy) == infisicalv1alpha1.DeletionPolicyOrphan || template.Status.TemplateID == "" {
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, template)
	}
	apiClient, err := infisicalClientForConnection(ctx, r.Client, template.Namespace, template.Spec.ConnectionRef)
	if err != nil {
		return ctrl.Result{RequeueAfter: retryFor(err)}, err
	}
	if err := apiClient.DeleteIdentityTemplate(ctx, template.Status.TemplateID); err != nil && !infisicalclient.IsNotFound(err) {
		return ctrl.Result{RequeueAfter: externalRetry}, err
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, template)
}

func (r *InfisicalIdentityTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infisicalv1alpha1.InfisicalIdentityTemplate{}).
		Watches(&infisicalv1alpha1.InfisicalConnection{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var templates infisicalv1alpha1.InfisicalIdentityTemplateList
			if err := mgr.GetClient().List(ctx, &templates, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range templates.Items {
				if templates.Items[i].Spec.ConnectionRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(&templates.Items[i])})
				}
			}
			return requests
		})).
		Watches(&infisicalv1alpha1.InfisicalOrganization{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var templates infisicalv1alpha1.InfisicalIdentityTemplateList
			if err := mgr.GetClient().List(ctx, &templates, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range templates.Items {
				if templates.Items[i].Spec.OrganizationRef.Name == object.GetName() {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(&templates.Items[i])})
				}
			}
			return requests
		})).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			var templates infisicalv1alpha1.InfisicalIdentityTemplateList
			if err := mgr.GetClient().List(ctx, &templates, client.InNamespace(object.GetNamespace())); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
			for i := range templates.Items {
				template := &templates.Items[i]
				if identityTemplateReferencesSecret(template, object.GetName()) {
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(template)})
				}
			}
			return requests
		})).
		Complete(r)
}

func identityTemplateReferencesSecret(template *infisicalv1alpha1.InfisicalIdentityTemplate, name string) bool {
	if template.Spec.AuthMethod == infisicalv1alpha1.IdentityTemplateAuthMethodLDAP && template.Spec.LDAP != nil {
		return template.Spec.LDAP.BindPasswordSecretRef.Name == name || (template.Spec.LDAP.CACertSecretRef != nil && template.Spec.LDAP.CACertSecretRef.Name == name)
	}
	if template.Spec.AuthMethod == infisicalv1alpha1.IdentityTemplateAuthMethodKubernetes && template.Spec.Kubernetes != nil {
		return (template.Spec.Kubernetes.CACertSecretRef != nil && template.Spec.Kubernetes.CACertSecretRef.Name == name) || (template.Spec.Kubernetes.TokenReviewerJWTSecretRef != nil && template.Spec.Kubernetes.TokenReviewerJWTSecretRef.Name == name)
	}
	return template.Spec.AuthMethod == infisicalv1alpha1.IdentityTemplateAuthMethodOIDC && template.Spec.OIDC != nil && template.Spec.OIDC.CACertSecretRef != nil && template.Spec.OIDC.CACertSecretRef.Name == name
}
