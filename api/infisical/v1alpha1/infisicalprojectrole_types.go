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

// ProjectRoleStringCondition describes a string condition using Infisical's comparison operators.
type ProjectRoleStringCondition struct {
	// Eq matches an exact value.
	// +optional
	Eq string `json:"$eq,omitempty"`
	// Ne excludes an exact value.
	// +optional
	Ne string `json:"$ne,omitempty"`
	// In matches one of the listed values.
	// +optional
	// +listType=atomic
	In []string `json:"$in,omitempty"`
	// Glob matches a glob pattern.
	// +optional
	Glob string `json:"$glob,omitempty"`
}

// ProjectRoleSecretTagsCondition describes secret tag matching.
type ProjectRoleSecretTagsCondition struct {
	// In matches any listed tag.
	// +optional
	// +listType=atomic
	In []string `json:"$in,omitempty"`
	// All requires all listed tags.
	// +optional
	// +listType=atomic
	All []string `json:"$all,omitempty"`
}

// ProjectRoleConditions limits a permission to matching project resources.
type ProjectRoleConditions struct {
	// Environment limits the environment.
	// +optional
	Environment *ProjectRoleStringCondition `json:"environment,omitempty"`
	// SecretPath limits the secret path.
	// +optional
	SecretPath *ProjectRoleStringCondition `json:"secretPath,omitempty"`
	// SecretName limits the secret name.
	// +optional
	SecretName *ProjectRoleStringCondition `json:"secretName,omitempty"`
	// SecretTags limits secret tags.
	// +optional
	SecretTags *ProjectRoleSecretTagsCondition `json:"secretTags,omitempty"`
	// EventType limits audit or event permissions.
	// +optional
	EventType *ProjectRoleStringCondition `json:"eventType,omitempty"`
}

// ProjectRolePermission defines one subject/action permission rule.
type ProjectRolePermission struct {
	// Subject identifies the Infisical resource subject.
	// +kubebuilder:validation:MinLength=1
	Subject string `json:"subject"`
	// Action lists one or more operations allowed on the subject.
	// +kubebuilder:validation:MinItems=1
	// +listType=atomic
	Action []string `json:"action"`
	// Inverted turns the rule into a deny rule when true.
	// +optional
	Inverted *bool `json:"inverted,omitempty"`
	// Conditions limits the rule to matching resources.
	// +optional
	Conditions *ProjectRoleConditions `json:"conditions,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="!has(oldSelf.connectionRef) || self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the InfisicalProjectRole"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.projectRef) || self.projectRef == oldSelf.projectRef",message="projectRef is immutable; delete and recreate the InfisicalProjectRole"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.slug) || self.slug == oldSelf.slug",message="slug is immutable; delete and recreate the InfisicalProjectRole"

// InfisicalProjectRoleSpec defines the desired state of InfisicalProjectRole.
type InfisicalProjectRoleSpec struct {
	// ConnectionRef selects the Infisical API connection.
	ConnectionRef InfisicalConnectionReference `json:"connectionRef"`

	// ProjectRef references the InfisicalProject that owns this role.
	ProjectRef LocalObjectReference `json:"projectRef"`

	// RoleName is the Infisical role display name. If omitted, metadata.name is used.
	// +optional
	// +kubebuilder:validation:MinLength=1
	RoleName string `json:"roleName,omitempty"`

	// Slug is the stable Infisical role slug. If omitted, metadata.name is used.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	Slug string `json:"slug,omitempty"`

	// Description is an optional role description.
	// +optional
	// +kubebuilder:validation:MaxLength=1024
	Description string `json:"description,omitempty"`

	// Permissions contains the role permission rules.
	// +kubebuilder:validation:MinItems=1
	Permissions []ProjectRolePermission `json:"permissions"`

	// CreationPolicy controls whether the operator creates or adopts a role.
	// +optional
	// +kubebuilder:default=Create
	// +kubebuilder:validation:Enum=Create;Adopt;CreateOrAdopt
	CreationPolicy CreationPolicy `json:"creationPolicy,omitempty"`

	// DeletionPolicy controls whether the external role is deleted with this resource.
	// +optional
	// +kubebuilder:default=Orphan
	// +kubebuilder:validation:Enum=Delete;Orphan
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
}

// InfisicalProjectRoleStatus defines the observed state of InfisicalProjectRole.
type InfisicalProjectRoleStatus struct {
	// RoleID is the immutable Infisical identifier once created or adopted.
	RoleID string `json:"roleID,omitempty"`

	// ProjectID is the observed owning project identifier.
	ProjectID string `json:"projectID,omitempty"`

	// Name is the observed Infisical role display name.
	Name string `json:"name,omitempty"`

	// Slug is the observed Infisical role slug.
	Slug string `json:"slug,omitempty"`

	// Description is the observed Infisical role description.
	Description string `json:"description,omitempty"`

	// Permissions contains the observed permission rules.
	// +optional
	Permissions []ProjectRolePermission `json:"permissions,omitempty"`

	// ObservedGeneration is the most recent generation reflected in status.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of the role.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:printcolumn:name="Role ID",type="string",JSONPath=".status.roleID"
// +kubebuilder:printcolumn:name="Project ID",type="string",JSONPath=".status.projectID"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].reason"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// InfisicalProjectRole is the Schema for the infisicalprojectroles API.
type InfisicalProjectRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// Spec defines the desired state of InfisicalProjectRole.
	// +required
	Spec InfisicalProjectRoleSpec `json:"spec"`

	// Status defines the observed state of InfisicalProjectRole.
	// +optional
	Status InfisicalProjectRoleStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// InfisicalProjectRoleList contains a list of InfisicalProjectRole.
type InfisicalProjectRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []InfisicalProjectRole `json:"items"`
}
