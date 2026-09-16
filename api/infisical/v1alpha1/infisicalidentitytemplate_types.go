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

package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// IdentityTemplateAuthMethod selects the Infisical authentication method configured by a template.
// +kubebuilder:validation:Enum=ldap;kubernetes;oidc
type IdentityTemplateAuthMethod string

const (
	IdentityTemplateAuthMethodLDAP       IdentityTemplateAuthMethod = "ldap"
	IdentityTemplateAuthMethodKubernetes IdentityTemplateAuthMethod = "kubernetes"
	IdentityTemplateAuthMethodOIDC       IdentityTemplateAuthMethod = "oidc"
)

// IdentityTemplateKubernetesSpec defines Kubernetes Auth template fields.
type IdentityTemplateKubernetesSpec struct {
	// TokenReviewMode selects the API-server or gateway token review path.
	// +optional
	// +kubebuilder:validation:Enum=api;gateway
	TokenReviewMode KubernetesTokenReviewMode `json:"tokenReviewMode,omitempty"`
	// KubernetesHost is the Kubernetes API server URL used for token review.
	// +optional
	// +kubebuilder:validation:MaxLength=255
	KubernetesHost string `json:"kubernetesHost,omitempty"`
	// CACertSecretRef references a Secret containing the PEM-encoded API CA certificate.
	// +optional
	CACertSecretRef *SecretKeyReference `json:"caCertSecretRef,omitempty"`
	// VerifyTLSCertificate controls API server certificate verification.
	// +optional
	VerifyTLSCertificate *bool `json:"verifyTLSCertificate,omitempty"`
	// TokenReviewerJWTSecretRef references a Secret containing a TokenReview JWT.
	// +optional
	TokenReviewerJWTSecretRef *SecretKeyReference `json:"tokenReviewerJWTSecretRef,omitempty"`
	// GatewayID selects an Infisical gateway.
	// +optional
	// +kubebuilder:validation:Format=uuid
	GatewayID string `json:"gatewayID,omitempty"`
	// GatewayPoolID selects an Infisical gateway pool.
	// +optional
	// +kubebuilder:validation:Format=uuid
	GatewayPoolID string `json:"gatewayPoolID,omitempty"`
	// AllowedAudience restricts the audience claim on authenticating tokens.
	// +optional
	// +kubebuilder:validation:MaxLength=1000
	AllowedAudience string `json:"allowedAudience,omitempty"`
}

// IdentityTemplateOIDCSpec defines OIDC Auth template fields.
type IdentityTemplateOIDCSpec struct {
	// OIDCDiscoveryURL is the identity provider discovery URL.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=2048
	// +kubebuilder:validation:Format=uri
	OIDCDiscoveryURL string `json:"oidcDiscoveryURL"`
	// BoundIssuer is the expected JWT issuer.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=2048
	BoundIssuer string `json:"boundIssuer"`
	// BoundAudiences is the comma-separated audience list.
	// +optional
	// +kubebuilder:validation:MaxLength=2048
	BoundAudiences string `json:"boundAudiences,omitempty"`
	// CACertSecretRef references a Secret containing the identity provider CA certificate.
	// +optional
	CACertSecretRef *SecretKeyReference `json:"caCertSecretRef,omitempty"`
}

// IdentityTemplateLDAPSpec defines LDAP Auth template fields.
type IdentityTemplateLDAPSpec struct {
	// URL is the LDAP server URL.
	// +kubebuilder:validation:MinLength=1
	URL string `json:"url"`
	// BindDN is the LDAP bind distinguished name.
	// +kubebuilder:validation:MinLength=1
	BindDN string `json:"bindDN"`
	// BindPasswordSecretRef references a Secret containing the LDAP bind password.
	BindPasswordSecretRef SecretKeyReference `json:"bindPasswordSecretRef"`
	// SearchBase is the LDAP search base.
	// +kubebuilder:validation:MinLength=1
	SearchBase string `json:"searchBase"`
	// CACertSecretRef references a Secret containing the LDAP CA certificate.
	// +optional
	CACertSecretRef *SecretKeyReference `json:"caCertSecretRef,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="(self.authMethod == 'ldap' && has(self.ldap) && !has(self.kubernetes) && !has(self.oidc)) || (self.authMethod == 'kubernetes' && has(self.kubernetes) && !has(self.ldap) && !has(self.oidc)) || (self.authMethod == 'oidc' && has(self.oidc) && !has(self.ldap) && !has(self.kubernetes))",message="exactly one auth method configuration must match authMethod"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.connectionRef) || self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the InfisicalIdentityTemplate"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.organizationRef) || self.organizationRef == oldSelf.organizationRef",message="organizationRef is immutable; delete and recreate the InfisicalIdentityTemplate"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.authMethod) || self.authMethod == oldSelf.authMethod",message="authMethod is immutable; delete and recreate the InfisicalIdentityTemplate"

// InfisicalIdentityTemplateSpec defines an organization-owned identity authentication template.
type InfisicalIdentityTemplateSpec struct {
	// ConnectionRef selects the Infisical API connection.
	ConnectionRef InfisicalConnectionReference `json:"connectionRef"`
	// OrganizationRef identifies the organization that owns this template.
	OrganizationRef LocalObjectReference `json:"organizationRef"`
	// TemplateName is the Infisical template name. If omitted, metadata.name is used.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	TemplateName string `json:"templateName,omitempty"`
	// AuthMethod selects the template authentication method.
	AuthMethod IdentityTemplateAuthMethod `json:"authMethod"`
	// LDAP contains LDAP template settings when authMethod is ldap.
	// +optional
	LDAP *IdentityTemplateLDAPSpec `json:"ldap,omitempty"`
	// Kubernetes contains Kubernetes Auth template settings when authMethod is kubernetes.
	// +optional
	Kubernetes *IdentityTemplateKubernetesSpec `json:"kubernetes,omitempty"`
	// OIDC contains OIDC template settings when authMethod is oidc.
	// +optional
	OIDC *IdentityTemplateOIDCSpec `json:"oidc,omitempty"`
	// CreationPolicy controls whether the operator creates or adopts a template.
	// +optional
	// +kubebuilder:default=Create
	// +kubebuilder:validation:Enum=Create;Adopt;CreateOrAdopt
	CreationPolicy CreationPolicy `json:"creationPolicy,omitempty"`
	// DeletionPolicy controls whether the external template is deleted with this resource.
	// +optional
	// +kubebuilder:default=Orphan
	// +kubebuilder:validation:Enum=Delete;Orphan
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
}

// IdentityTemplateKubernetesStatus exposes non-secret observed Kubernetes Auth template fields.
type IdentityTemplateKubernetesStatus struct {
	TokenReviewMode      KubernetesTokenReviewMode `json:"tokenReviewMode,omitempty"`
	KubernetesHost       string                    `json:"kubernetesHost,omitempty"`
	AllowedAudience      string                    `json:"allowedAudience,omitempty"`
	GatewayID            string                    `json:"gatewayID,omitempty"`
	GatewayPoolID        string                    `json:"gatewayPoolID,omitempty"`
	VerifyTLSCertificate bool                      `json:"verifyTLSCertificate,omitempty"`
	HasCACertificate     bool                      `json:"hasCACertificate,omitempty"`
	HasTokenReviewerJWT  bool                      `json:"hasTokenReviewerJWT,omitempty"`
}

// IdentityTemplateOIDCStatus exposes non-secret observed OIDC template fields.
type IdentityTemplateOIDCStatus struct {
	OIDCDiscoveryURL string `json:"oidcDiscoveryURL,omitempty"`
	BoundIssuer      string `json:"boundIssuer,omitempty"`
	BoundAudiences   string `json:"boundAudiences,omitempty"`
	HasCACertificate bool   `json:"hasCACertificate,omitempty"`
}

// IdentityTemplateLDAPStatus exposes non-secret observed LDAP template fields.
type IdentityTemplateLDAPStatus struct {
	URL              string `json:"url,omitempty"`
	BindDN           string `json:"bindDN,omitempty"`
	SearchBase       string `json:"searchBase,omitempty"`
	HasCACertificate bool   `json:"hasCACertificate,omitempty"`
	HasBindPassword  bool   `json:"hasBindPassword,omitempty"`
}

// InfisicalIdentityTemplateStatus defines the observed state of an identity template.
type InfisicalIdentityTemplateStatus struct {
	TemplateID     string                            `json:"templateID,omitempty"`
	Name           string                            `json:"name,omitempty"`
	AuthMethod     IdentityTemplateAuthMethod        `json:"authMethod,omitempty"`
	OrganizationID string                            `json:"organizationID,omitempty"`
	LDAP           *IdentityTemplateLDAPStatus       `json:"ldap,omitempty"`
	Kubernetes     *IdentityTemplateKubernetesStatus `json:"kubernetes,omitempty"`
	OIDC           *IdentityTemplateOIDCStatus       `json:"oidc,omitempty"`
	// SecretResourceVersions identifies the referenced Secret versions represented in the remote template.
	// It contains no Secret data and is used to detect updates to write-only template fields.
	// +optional
	// +listType=atomic
	SecretResourceVersions []string `json:"secretResourceVersions,omitempty"`
	ObservedGeneration     int64    `json:"observedGeneration,omitempty"`
	// Conditions represent the latest available observations of the template.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:printcolumn:name="Template ID",type="string",JSONPath=".status.templateID"
// +kubebuilder:printcolumn:name="Auth Method",type="string",JSONPath=".status.authMethod"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].reason"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// InfisicalIdentityTemplate is the Schema for the infisicalidentitytemplates API.
type InfisicalIdentityTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              InfisicalIdentityTemplateSpec   `json:"spec"`
	Status            InfisicalIdentityTemplateStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// InfisicalIdentityTemplateList contains a list of InfisicalIdentityTemplate.
type InfisicalIdentityTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []InfisicalIdentityTemplate `json:"items"`
}
