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

// UniversalAuthTrustedIP identifies an IP address or CIDR range allowed by Universal Auth.
type UniversalAuthTrustedIP struct {
	// IPAddress is an IP address or CIDR range.
	// +kubebuilder:validation:MinLength=1
	IPAddress string `json:"ipAddress"`
}

// UniversalAuthConfigSpec contains the optional Infisical Universal Auth configuration.
// Omitted fields retain Infisical's server-side defaults and are not managed by the operator.
type UniversalAuthConfigSpec struct {
	// ClientSecretTrustedIPs limits where client secrets may be exchanged for access tokens.
	// +optional
	// +listType=atomic
	ClientSecretTrustedIPs []UniversalAuthTrustedIP `json:"clientSecretTrustedIPs,omitempty"`

	// AccessTokenTrustedIPs limits where issued access tokens may be used.
	// +optional
	// +listType=atomic
	AccessTokenTrustedIPs []UniversalAuthTrustedIP `json:"accessTokenTrustedIPs,omitempty"`

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

	// AccessTokenPeriod is the renewable access-token period in seconds. Zero disables periodic renewal.
	// +optional
	// +kubebuilder:validation:Minimum=0
	AccessTokenPeriod *int64 `json:"accessTokenPeriod,omitempty"`

	// LockoutEnabled enables lockout after failed Universal Auth logins.
	// +optional
	LockoutEnabled *bool `json:"lockoutEnabled,omitempty"`

	// LockoutThreshold is the number of failed login attempts before lockout.
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=30
	LockoutThreshold *int64 `json:"lockoutThreshold,omitempty"`

	// LockoutDurationSeconds is how long a locked identity remains locked.
	// +optional
	// +kubebuilder:validation:Minimum=30
	// +kubebuilder:validation:Maximum=86400
	LockoutDurationSeconds *int64 `json:"lockoutDurationSeconds,omitempty"`

	// LockoutCounterResetSeconds is how long until a failed-login counter resets.
	// +optional
	// +kubebuilder:validation:Minimum=5
	// +kubebuilder:validation:Maximum=3600
	LockoutCounterResetSeconds *int64 `json:"lockoutCounterResetSeconds,omitempty"`
}

// UniversalAuthClientSecretSpec declares the Kubernetes Secret to publish and its remote secret lifetime.
type UniversalAuthClientSecretSpec struct {
	// SecretRef identifies the same-namespace Secret that will contain clientId and clientSecret.
	SecretRef UniversalAuthSecretReference `json:"secretRef"`

	// Description is the Infisical client secret description.
	// +optional
	// +kubebuilder:validation:MaxLength=255
	Description string `json:"description,omitempty"`

	// NumUsesLimit limits client-secret exchanges. Zero means unlimited.
	// +optional
	// +kubebuilder:validation:Minimum=0
	NumUsesLimit *int64 `json:"numUsesLimit,omitempty"`

	// TTL is the client secret lifetime in seconds. Zero means no expiry.
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=315360000
	TTL *int64 `json:"ttl,omitempty"`

	// RotationNonce forces a new remote client secret when changed.
	// +optional
	RotationNonce string `json:"rotationNonce,omitempty"`
}

// InfisicalUniversalAuthSpec defines the desired state of an Infisical Universal Auth method.
type InfisicalUniversalAuthSpec struct {
	// ConnectionRef selects the Infisical API connection used to manage the auth method.
	ConnectionRef InfisicalConnectionReference `json:"connectionRef"`

	// IdentityRef references the same-namespace machine identity receiving Universal Auth.
	IdentityRef LocalObjectReference `json:"identityRef"`

	// Config contains optional Universal Auth settings.
	// +optional
	Config *UniversalAuthConfigSpec `json:"config,omitempty"`

	// ClientSecret declares the remote client secret and Kubernetes Secret publication.
	ClientSecret UniversalAuthClientSecretSpec `json:"clientSecret"`

	// CreationPolicy controls whether the operator attaches or adopts Universal Auth.
	// +optional
	// +kubebuilder:default=Create
	// +kubebuilder:validation:Enum=Create;Adopt;CreateOrAdopt
	CreationPolicy CreationPolicy `json:"creationPolicy,omitempty"`

	// DeletionPolicy controls whether the remote Universal Auth configuration is removed.
	// +optional
	// +kubebuilder:default=Orphan
	// +kubebuilder:validation:Enum=Delete;Orphan
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
}

// UniversalAuthConfigStatus contains non-secret observed Universal Auth settings.
type UniversalAuthConfigStatus struct {
	ClientSecretTrustedIPs     []UniversalAuthTrustedIP `json:"clientSecretTrustedIPs,omitempty"`
	AccessTokenTrustedIPs      []UniversalAuthTrustedIP `json:"accessTokenTrustedIPs,omitempty"`
	AccessTokenTTL             int64                    `json:"accessTokenTTL,omitempty"`
	AccessTokenMaxTTL          int64                    `json:"accessTokenMaxTTL,omitempty"`
	AccessTokenNumUsesLimit    int64                    `json:"accessTokenNumUsesLimit,omitempty"`
	AccessTokenPeriod          int64                    `json:"accessTokenPeriod,omitempty"`
	LockoutEnabled             bool                     `json:"lockoutEnabled,omitempty"`
	LockoutThreshold           int64                    `json:"lockoutThreshold,omitempty"`
	LockoutDurationSeconds     int64                    `json:"lockoutDurationSeconds,omitempty"`
	LockoutCounterResetSeconds int64                    `json:"lockoutCounterResetSeconds,omitempty"`
}

// UniversalAuthClientSecretStatus contains non-secret observed client-secret metadata.
type UniversalAuthClientSecretStatus struct {
	SecretName         string `json:"secretName,omitempty"`
	ClientSecretID     string `json:"clientSecretID,omitempty"`
	ClientSecretPrefix string `json:"clientSecretPrefix,omitempty"`
	LastRotationNonce  string `json:"lastRotationNonce,omitempty"`
	// LastRotatedAt records when the currently published client secret was created.
	// +optional
	LastRotatedAt *metav1.Time `json:"lastRotatedAt,omitempty"`
}

// InfisicalUniversalAuthStatus defines the observed state of an Infisical Universal Auth method.
type InfisicalUniversalAuthStatus struct {
	// AuthID is the immutable Infisical Universal Auth configuration identifier.
	AuthID string `json:"authID,omitempty"`
	// IdentityID is the observed target machine identity identifier.
	IdentityID string `json:"identityID,omitempty"`
	// ClientID is safe to expose and is required for token exchange.
	ClientID           string                          `json:"clientID,omitempty"`
	Config             UniversalAuthConfigStatus       `json:"config,omitzero"`
	ClientSecret       UniversalAuthClientSecretStatus `json:"clientSecret,omitzero"`
	ObservedGeneration int64                           `json:"observedGeneration,omitempty"`
	// Conditions represent the latest available observations.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:printcolumn:name="Auth ID",type="string",JSONPath=".status.authID"
// +kubebuilder:printcolumn:name="Client ID",type="string",JSONPath=".status.clientID"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].reason"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// InfisicalUniversalAuth is the Schema for the infisicaluniversalauths API.
type InfisicalUniversalAuth struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              InfisicalUniversalAuthSpec   `json:"spec"`
	Status            InfisicalUniversalAuthStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// InfisicalUniversalAuthList contains a list of InfisicalUniversalAuth.
type InfisicalUniversalAuthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []InfisicalUniversalAuth `json:"items"`
}
