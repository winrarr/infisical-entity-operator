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

// CreationPolicy controls how an external entity is acquired.
// +kubebuilder:validation:Enum=Create;Adopt;CreateOrAdopt
type CreationPolicy string

const (
	CreationPolicyCreate        CreationPolicy = "Create"
	CreationPolicyAdopt         CreationPolicy = "Adopt"
	CreationPolicyCreateOrAdopt CreationPolicy = "CreateOrAdopt"
)

// DeletionPolicy controls what happens to the external entity on deletion.
// +kubebuilder:validation:Enum=Delete;Orphan
type DeletionPolicy string

const (
	DeletionPolicyDelete DeletionPolicy = "Delete"
	DeletionPolicyOrphan DeletionPolicy = "Orphan"
)

// ProjectType is an Infisical product type supported by the Infisical free plans.
// Secret Manager is available in the core free plan and Certificate Manager has
// a separate free plan with its documented certificate and CA limits.
// +kubebuilder:validation:Enum=secret-manager;cert-manager
type ProjectType string

const (
	ProjectTypeSecretManager ProjectType = "secret-manager"
	ProjectTypeCertManager   ProjectType = "cert-manager"
)

// SecretKeyReference identifies a key in a same-namespace Secret.
type SecretKeyReference struct {
	// Name is the Secret name.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Key is the Secret data key.
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key,omitempty"`
}

// UniversalAuthSecretReference references a same-namespace Secret containing
// an Infisical Universal Auth client ID and client secret.
type UniversalAuthSecretReference struct {
	// Name is the Secret name.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// ClientIDKey is the Secret key containing the client ID. It defaults to clientId.
	// +optional
	ClientIDKey string `json:"clientIDKey,omitempty"`
	// ClientSecretKey is the Secret key containing the client secret. It defaults to clientSecret.
	// +optional
	ClientSecretKey string `json:"clientSecretKey,omitempty"`
}

// InfisicalConnectionReference identifies a same-namespace connection.
type InfisicalConnectionReference struct {
	// Name is the InfisicalConnection name.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// LocalObjectReference identifies a same-namespace custom resource.
type LocalObjectReference struct {
	// Name is the referenced resource name.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// IdentityMetadata is a key/value pair attached to an Infisical identity.
type IdentityMetadata struct {
	// Key is the metadata key.
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`
	// Value is the metadata value.
	Value string `json:"value"`
}

// EnvironmentStatus describes an Infisical project environment.
type EnvironmentStatus struct {
	// ID is the Infisical environment identifier.
	ID string `json:"id"`
	// Name is the display name.
	Name string `json:"name"`
	// Slug is the stable environment slug.
	Slug string `json:"slug"`
}
