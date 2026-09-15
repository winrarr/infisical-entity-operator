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

// +kubebuilder:validation:XValidation:rule="!has(oldSelf.connectionRef) || self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the InfisicalOrganization"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.organizationID) || self.organizationID == oldSelf.organizationID",message="organizationID is immutable; delete and recreate the InfisicalOrganization"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.organizationName) || self.organizationName == oldSelf.organizationName",message="organizationName is immutable; delete and recreate the InfisicalOrganization"

// InfisicalOrganizationSpec defines the desired state of an Infisical organization.
type InfisicalOrganizationSpec struct {
	// ConnectionRef selects the Infisical API connection used for organization lifecycle.
	// Creating or deleting an organization requires a user JWT or API key; machine identity
	// tokens can adopt an explicit organization after the platform has provisioned it.
	ConnectionRef InfisicalConnectionReference `json:"connectionRef"`

	// OrganizationID identifies an existing organization to adopt. It is also the explicit
	// boundary used by tenant operators, which may not be able to call Infisical's user-only
	// organization lookup endpoint.
	// +optional
	// +kubebuilder:validation:MinLength=1
	OrganizationID string `json:"organizationID,omitempty"`

	// OrganizationName is the Infisical display name. If omitted, metadata.name is used.
	// It is required when creating an organization and is used for name-based adoption.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	OrganizationName string `json:"organizationName,omitempty"`

	// CreationPolicy controls whether the operator creates or adopts an organization.
	// +optional
	// +kubebuilder:default=Create
	// +kubebuilder:validation:Enum=Create;Adopt;CreateOrAdopt
	CreationPolicy CreationPolicy `json:"creationPolicy,omitempty"`

	// DeletionPolicy controls whether the external organization is deleted with this resource.
	// +optional
	// +kubebuilder:default=Orphan
	// +kubebuilder:validation:Enum=Delete;Orphan
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
}

// InfisicalOrganizationStatus defines the observed state of InfisicalOrganization.
type InfisicalOrganizationStatus struct {
	// OrganizationID is the immutable Infisical identifier once created or adopted.
	OrganizationID string `json:"organizationID,omitempty"`

	// OrganizationName is the observed Infisical display name.
	OrganizationName string `json:"organizationName,omitempty"`

	// Slug is the observed Infisical organization slug.
	Slug string `json:"slug,omitempty"`

	// ObservedGeneration is the most recent generation reflected in status.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of the organization.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:printcolumn:name="Organization ID",type="string",JSONPath=".status.organizationID"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].reason"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// InfisicalOrganization is the Schema for the infisicalorganizations API.
type InfisicalOrganization struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// Spec defines the desired state of InfisicalOrganization.
	// +required
	Spec InfisicalOrganizationSpec `json:"spec"`

	// Status defines the observed state of InfisicalOrganization.
	// +optional
	Status InfisicalOrganizationStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// InfisicalOrganizationList contains a list of InfisicalOrganization.
type InfisicalOrganizationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []InfisicalOrganization `json:"items"`
}
