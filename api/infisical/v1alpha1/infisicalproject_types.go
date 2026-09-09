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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:XValidation:rule="!has(oldSelf.connectionRef) || self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the InfisicalProject"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.shouldCreateDefaultEnvs) || self.shouldCreateDefaultEnvs == oldSelf.shouldCreateDefaultEnvs",message="shouldCreateDefaultEnvs is immutable; delete and recreate the InfisicalProject"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.type) || self.type == oldSelf.type",message="type is immutable; delete and recreate the InfisicalProject"

// InfisicalProjectSpec defines the desired state of InfisicalProject
type InfisicalProjectSpec struct {
	// ConnectionRef selects the Infisical API connection.
	ConnectionRef InfisicalConnectionReference `json:"connectionRef"`

	// ProjectName is the Infisical project name. If omitted, metadata.name is used.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	ProjectName string `json:"projectName,omitempty"`

	// Description is an optional project description.
	// +optional
	// +kubebuilder:validation:MaxLength=1024
	Description string `json:"description,omitempty"`

	// Slug is an optional unique project slug.
	// +optional
	// +kubebuilder:validation:MinLength=5
	// +kubebuilder:validation:MaxLength=64
	Slug string `json:"slug,omitempty"`

	// Type selects the Infisical product type.
	// +optional
	// +kubebuilder:default="secret-manager"
	// +kubebuilder:validation:Enum=secret-manager;cert-manager;kms;ssh;secret-scanning;pam;ai
	Type ProjectType `json:"type,omitempty"`

	// ShouldCreateDefaultEnvs controls whether Infisical creates its default environments.
	// It is only used during project creation.
	// +optional
	// +kubebuilder:default=true
	ShouldCreateDefaultEnvs *bool `json:"shouldCreateDefaultEnvs,omitempty"`

	// HasDeleteProtection configures Infisical-side delete protection.
	// +optional
	// +kubebuilder:default=false
	HasDeleteProtection *bool `json:"hasDeleteProtection,omitempty"`

	// CreationPolicy controls whether the operator creates or adopts a project.
	// +optional
	// +kubebuilder:default=Create
	// +kubebuilder:validation:Enum=Create;Adopt;CreateOrAdopt
	CreationPolicy CreationPolicy `json:"creationPolicy,omitempty"`

	// DeletionPolicy controls whether the external project is deleted with this resource.
	// +optional
	// +kubebuilder:default=Orphan
	// +kubebuilder:validation:Enum=Delete;Orphan
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
}

// InfisicalProjectStatus defines the observed state of InfisicalProject.
type InfisicalProjectStatus struct {
	// ProjectID is the immutable Infisical identifier once created or adopted.
	ProjectID string `json:"projectID,omitempty"`

	// Slug is the observed Infisical project slug.
	Slug string `json:"slug,omitempty"`

	// OrganizationID is the organization that owns the project.
	OrganizationID string `json:"organizationID,omitempty"`

	// Environments contains the environments returned by Infisical.
	// +listType=map
	// +listMapKey=id
	// +optional
	Environments []EnvironmentStatus `json:"environments,omitempty"`

	// ObservedGeneration is the most recent generation reflected in status.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of the project.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:printcolumn:name="Project ID",type="string",JSONPath=".status.projectID"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].reason"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// InfisicalProject is the Schema for the infisicalprojects API
type InfisicalProject struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of InfisicalProject
	// +required
	Spec InfisicalProjectSpec `json:"spec"`

	// status defines the observed state of InfisicalProject
	// +optional
	Status InfisicalProjectStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// InfisicalProjectList contains a list of InfisicalProject
type InfisicalProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []InfisicalProject `json:"items"`
}
