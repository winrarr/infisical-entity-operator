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

// KubernetesTokenReviewMode selects how Infisical validates Kubernetes service account tokens.
// +kubebuilder:validation:Enum=api;gateway
type KubernetesTokenReviewMode string

const (
	KubernetesTokenReviewModeAPI     KubernetesTokenReviewMode = "api"
	KubernetesTokenReviewModeGateway KubernetesTokenReviewMode = "gateway"
)

// KubernetesTrustedIP identifies an IP address or CIDR range allowed to use an access token.
type KubernetesTrustedIP struct {
	// IPAddress is an IP address or CIDR range.
	// +kubebuilder:validation:MinLength=1
	IPAddress string `json:"ipAddress"`
}

// +kubebuilder:validation:XValidation:rule="!has(oldSelf.connectionRef) || self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the InfisicalKubernetesAuth"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.identityRef) || self.identityRef == oldSelf.identityRef",message="identityRef is immutable; delete and recreate the InfisicalKubernetesAuth"

// InfisicalKubernetesAuthSpec defines the desired state of Infisical Kubernetes Auth.
type InfisicalKubernetesAuthSpec struct {
	// ConnectionRef selects the Infisical API connection.
	ConnectionRef InfisicalConnectionReference `json:"connectionRef"`

	// IdentityRef references the InfisicalIdentity receiving this auth method.
	IdentityRef LocalObjectReference `json:"identityRef"`

	// KubernetesHost is the Kubernetes API server URL Infisical uses for token review.
	// +optional
	// +kubebuilder:validation:MaxLength=255
	KubernetesHost string `json:"kubernetesHost,omitempty"`

	// AllowedNamespaces lists the Kubernetes namespaces trusted to authenticate.
	// +kubebuilder:validation:MinItems=1
	// +listType=atomic
	AllowedNamespaces []string `json:"allowedNamespaces"`

	// AllowedNames lists the service account names trusted to authenticate.
	// +kubebuilder:validation:MinItems=1
	// +listType=atomic
	AllowedNames []string `json:"allowedNames"`

	// AllowedAudience is the optional audience required in the service account token.
	// +optional
	// +kubebuilder:validation:MaxLength=1000
	AllowedAudience string `json:"allowedAudience,omitempty"`

	// CACertSecretRef references a Secret containing the PEM-encoded Kubernetes API CA certificate.
	// +optional
	CACertSecretRef *SecretKeyReference `json:"caCertSecretRef,omitempty"`

	// VerifyTLSCertificate controls Kubernetes API server certificate verification.
	// +optional
	VerifyTLSCertificate *bool `json:"verifyTLSCertificate,omitempty"`

	// TokenReviewerJWTSecretRef references a Secret containing a token for the Kubernetes TokenReview API.
	// If omitted, Infisical can use the authenticating workload token when configured for that mode.
	// +optional
	TokenReviewerJWTSecretRef *SecretKeyReference `json:"tokenReviewerJWTSecretRef,omitempty"`

	// TokenReviewMode selects the API-server or gateway token review path.
	// +optional
	// +kubebuilder:default=api
	// +kubebuilder:validation:Enum=api;gateway
	TokenReviewMode KubernetesTokenReviewMode `json:"tokenReviewMode,omitempty"`

	// GatewayID selects an Infisical gateway for token review when gateway mode is used.
	// +optional
	GatewayID string `json:"gatewayID,omitempty"`

	// GatewayPoolID selects an Infisical gateway pool for token review when gateway mode is used.
	// +optional
	GatewayPoolID string `json:"gatewayPoolID,omitempty"`

	// AccessTokenTrustedIPs limits where issued access tokens may be used.
	// +optional
	// +listType=atomic
	AccessTokenTrustedIPs []KubernetesTrustedIP `json:"accessTokenTrustedIPs,omitempty"`

	// AccessTokenTTL is the access token lifetime in seconds.
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=315360000
	AccessTokenTTL *int64 `json:"accessTokenTTL,omitempty"`

	// AccessTokenMaxTTL is the maximum access token lifetime in seconds.
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=315360000
	AccessTokenMaxTTL *int64 `json:"accessTokenMaxTTL,omitempty"`

	// AccessTokenNumUsesLimit limits access token uses. Zero means unlimited.
	// +optional
	// +kubebuilder:validation:Minimum=0
	AccessTokenNumUsesLimit *int64 `json:"accessTokenNumUsesLimit,omitempty"`

	// CreationPolicy controls whether the operator attaches or adopts Kubernetes Auth.
	// +optional
	// +kubebuilder:default=Create
	// +kubebuilder:validation:Enum=Create;Adopt;CreateOrAdopt
	CreationPolicy CreationPolicy `json:"creationPolicy,omitempty"`

	// DeletionPolicy controls whether the remote auth method is removed with this resource.
	// +optional
	// +kubebuilder:default=Orphan
	// +kubebuilder:validation:Enum=Delete;Orphan
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
}

// InfisicalKubernetesAuthStatus defines the observed state of Infisical Kubernetes Auth.
type InfisicalKubernetesAuthStatus struct {
	// AuthID is the immutable Infisical auth method identifier once attached or adopted.
	AuthID string `json:"authID,omitempty"`

	// IdentityID is the observed target identity identifier.
	IdentityID string `json:"identityID,omitempty"`

	// KubernetesHost is the observed Kubernetes API server URL.
	KubernetesHost string `json:"kubernetesHost,omitempty"`

	// AllowedNamespaces is the observed namespace allowlist.
	// +listType=atomic
	AllowedNamespaces []string `json:"allowedNamespaces,omitempty"`

	// AllowedNames is the observed service account allowlist.
	// +listType=atomic
	AllowedNames []string `json:"allowedNames,omitempty"`

	// AllowedAudience is the observed token audience.
	AllowedAudience string `json:"allowedAudience,omitempty"`

	// TokenReviewMode is the observed token review mode.
	TokenReviewMode KubernetesTokenReviewMode `json:"tokenReviewMode,omitempty"`

	// GatewayID is the observed gateway identifier.
	GatewayID string `json:"gatewayID,omitempty"`

	// GatewayPoolID is the observed gateway pool identifier.
	GatewayPoolID string `json:"gatewayPoolID,omitempty"`

	// VerifyTLSCertificate reports whether the Kubernetes API certificate is verified.
	VerifyTLSCertificate bool `json:"verifyTLSCertificate,omitempty"`

	// HasCACertificate reports whether a CA certificate is configured without exposing it.
	HasCACertificate bool `json:"hasCACertificate,omitempty"`

	// HasTokenReviewerJWT reports whether a reviewer JWT is configured without exposing it.
	HasTokenReviewerJWT bool `json:"hasTokenReviewerJWT,omitempty"`

	// AccessTokenTrustedIPs is the observed access token IP allowlist.
	// +listType=atomic
	AccessTokenTrustedIPs []KubernetesTrustedIP `json:"accessTokenTrustedIPs,omitempty"`

	// AccessTokenTTL is the observed access token lifetime in seconds.
	AccessTokenTTL int64 `json:"accessTokenTTL,omitempty"`

	// AccessTokenMaxTTL is the observed maximum access token lifetime in seconds.
	AccessTokenMaxTTL int64 `json:"accessTokenMaxTTL,omitempty"`

	// AccessTokenNumUsesLimit is the observed access token use limit.
	AccessTokenNumUsesLimit int64 `json:"accessTokenNumUsesLimit,omitempty"`

	// ObservedGeneration is the most recent generation reflected in status.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of Kubernetes Auth.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:printcolumn:name="Auth ID",type="string",JSONPath=".status.authID"
// +kubebuilder:printcolumn:name="Identity ID",type="string",JSONPath=".status.identityID"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].reason"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// InfisicalKubernetesAuth is the Schema for the infisicalkubernetesauths API.
type InfisicalKubernetesAuth struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// Spec defines the desired state of InfisicalKubernetesAuth.
	// +required
	Spec InfisicalKubernetesAuthSpec `json:"spec"`

	// Status defines the observed state of InfisicalKubernetesAuth.
	// +optional
	Status InfisicalKubernetesAuthStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// InfisicalKubernetesAuthList contains a list of InfisicalKubernetesAuth.
type InfisicalKubernetesAuthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []InfisicalKubernetesAuth `json:"items"`
}
