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

// +kubebuilder:validation:XValidation:rule="!has(oldSelf.connectionRef) || self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the InfisicalEnvironment"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.projectRef) || self.projectRef == oldSelf.projectRef",message="projectRef is immutable; delete and recreate the InfisicalEnvironment"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.slug) || self.slug == oldSelf.slug",message="slug is immutable; delete and recreate the InfisicalEnvironment"

// InfisicalEnvironmentSpec defines the desired state of InfisicalEnvironment.
type InfisicalEnvironmentSpec struct {
	// ConnectionRef selects the Infisical API connection.
	ConnectionRef InfisicalConnectionReference `json:"connectionRef"`

	// ProjectRef references the InfisicalProject that owns this environment.
	ProjectRef LocalObjectReference `json:"projectRef"`

	// EnvironmentName is the Infisical environment display name. If omitted, metadata.name is used.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	EnvironmentName string `json:"environmentName,omitempty"`

	// Slug is the stable Infisical environment slug. If omitted, metadata.name is used.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	Slug string `json:"slug,omitempty"`

	// Position controls the environment order. Lower values appear first.
	// +optional
	// +kubebuilder:validation:Minimum=1
	Position *int32 `json:"position,omitempty"`

	// CreationPolicy controls whether the operator creates or adopts an environment.
	// +optional
	// +kubebuilder:default=Create
	// +kubebuilder:validation:Enum=Create;Adopt;CreateOrAdopt
	CreationPolicy CreationPolicy `json:"creationPolicy,omitempty"`

	// DeletionPolicy controls whether the external environment is deleted with this resource.
	// +optional
	// +kubebuilder:default=Orphan
	// +kubebuilder:validation:Enum=Delete;Orphan
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
}

// InfisicalEnvironmentStatus defines the observed state of InfisicalEnvironment.
type InfisicalEnvironmentStatus struct {
	// EnvironmentID is the immutable Infisical identifier once created or adopted.
	EnvironmentID string `json:"environmentID,omitempty"`

	// ProjectID is the observed owning project identifier.
	ProjectID string `json:"projectID,omitempty"`

	// Name is the observed Infisical environment display name.
	Name string `json:"name,omitempty"`

	// Slug is the observed Infisical environment slug.
	Slug string `json:"slug,omitempty"`

	// Position is the observed environment order.
	Position int32 `json:"position,omitempty"`

	// ObservedGeneration is the most recent generation reflected in status.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of the environment.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:printcolumn:name="Environment ID",type="string",JSONPath=".status.environmentID"
// +kubebuilder:printcolumn:name="Project ID",type="string",JSONPath=".status.projectID"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].reason"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// InfisicalEnvironment is the Schema for the infisicalenvironments API.
type InfisicalEnvironment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// Spec defines the desired state of InfisicalEnvironment.
	// +required
	Spec InfisicalEnvironmentSpec `json:"spec"`

	// Status defines the observed state of InfisicalEnvironment.
	// +optional
	Status InfisicalEnvironmentStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// InfisicalEnvironmentList contains a list of InfisicalEnvironment.
type InfisicalEnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []InfisicalEnvironment `json:"items"`
}
