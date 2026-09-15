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

// ProjectTemplateEnvironment describes an environment created when a project uses a template.
type ProjectTemplateEnvironment struct {
	// Name is the environment display name.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Slug is the stable environment slug.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	Slug string `json:"slug"`
	// Position controls the environment order in the project.
	// +kubebuilder:validation:Minimum=1
	Position int32 `json:"position"`
}

// ProjectTemplateUser assigns roles to a user added to projects created from a template.
type ProjectTemplateUser struct {
	// Username identifies the Infisical user, normally by username or email.
	// +kubebuilder:validation:MinLength=1
	Username string `json:"username"`
	// Roles contains role slugs assigned to the user.
	// +kubebuilder:validation:MinItems=1
	// +listType=atomic
	Roles []string `json:"roles"`
}

// ProjectTemplateGroup assigns roles to a group added to projects created from a template.
type ProjectTemplateGroup struct {
	// GroupSlug identifies the Infisical group.
	// +kubebuilder:validation:MinLength=1
	GroupSlug string `json:"groupSlug"`
	// Roles contains role slugs assigned to the group.
	// +kubebuilder:validation:MinItems=1
	// +listType=atomic
	Roles []string `json:"roles"`
}

// ProjectTemplateIdentity assigns roles to an organization-owned identity added to projects.
type ProjectTemplateIdentity struct {
	// IdentityID is the Infisical machine identity identifier.
	// +kubebuilder:validation:Format=uuid
	IdentityID string `json:"identityID"`
	// Roles contains role slugs assigned to the identity.
	// +kubebuilder:validation:MinItems=1
	// +listType=atomic
	Roles []string `json:"roles"`
}

// ProjectTemplateManagedIdentity creates a project-owned identity and assigns roles to it.
type ProjectTemplateManagedIdentity struct {
	// Name is the project-owned identity name.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Roles contains role slugs assigned to the identity.
	// +kubebuilder:validation:MinItems=1
	// +listType=atomic
	Roles []string `json:"roles"`
}

// +kubebuilder:validation:XValidation:rule="!has(oldSelf.connectionRef) || self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the InfisicalProjectTemplate"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.organizationRef) || self.organizationRef == oldSelf.organizationRef",message="organizationRef is immutable; delete and recreate the InfisicalProjectTemplate"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.type) || self.type == oldSelf.type",message="type is immutable; delete and recreate the InfisicalProjectTemplate"

// InfisicalProjectTemplateSpec defines the desired state of an Infisical project template.
type InfisicalProjectTemplateSpec struct {
	// ConnectionRef selects the Infisical API connection.
	ConnectionRef InfisicalConnectionReference `json:"connectionRef"`

	// OrganizationRef optionally identifies the organization that owns this template.
	// +optional
	OrganizationRef *LocalObjectReference `json:"organizationRef,omitempty"`

	// TemplateName is the Infisical template name. If omitted, metadata.name is used.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	TemplateName string `json:"templateName,omitempty"`

	// Description is an optional template description.
	// +optional
	// +kubebuilder:validation:MaxLength=256
	Description string `json:"description,omitempty"`

	// Type selects the Infisical product type for projects created from this template.
	// +optional
	// +kubebuilder:default="secret-manager"
	// +kubebuilder:validation:Enum=secret-manager;cert-manager;kms;secret-scanning;pam;agent-vault
	Type ProjectType `json:"type,omitempty"`

	// Roles contains the custom project roles created by this template.
	// +optional
	Roles []ProjectTemplateRole `json:"roles,omitempty"`

	// Environments contains the environments created by this template.
	// +optional
	Environments []ProjectTemplateEnvironment `json:"environments,omitempty"`

	// Users contains users automatically added to projects created from this template.
	// +optional
	Users []ProjectTemplateUser `json:"users,omitempty"`

	// Groups contains groups automatically added to projects created from this template.
	// Group support depends on the Infisical plan and group configuration.
	// +optional
	Groups []ProjectTemplateGroup `json:"groups,omitempty"`

	// Identities contains organization-owned identities automatically added to projects.
	// +optional
	Identities []ProjectTemplateIdentity `json:"identities,omitempty"`

	// ProjectManagedIdentities contains project-owned identities created from this template.
	// +optional
	ProjectManagedIdentities []ProjectTemplateManagedIdentity `json:"projectManagedIdentities,omitempty"`

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

// ProjectTemplateRole describes one custom role in a project template.
type ProjectTemplateRole struct {
	// Name is the role display name.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Slug is the stable role slug.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	Slug string `json:"slug"`
	// Permissions contains the role permission rules.
	// +optional
	Permissions []ProjectRolePermission `json:"permissions,omitempty"`
}

// InfisicalProjectTemplateStatus defines the observed state of InfisicalProjectTemplate.
type InfisicalProjectTemplateStatus struct {
	// TemplateID is the immutable Infisical identifier once created or adopted.
	TemplateID string `json:"templateID,omitempty"`
	// Name is the observed Infisical template name.
	Name string `json:"name,omitempty"`
	// Description is the observed template description.
	Description string `json:"description,omitempty"`
	// Type is the observed project type.
	Type ProjectType `json:"type,omitempty"`
	// OrganizationID is the organization that owns the template.
	OrganizationID string `json:"organizationID,omitempty"`
	// ObservedGeneration is the most recent generation reflected in status.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions represent the latest available observations of the template.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:printcolumn:name="Template ID",type="string",JSONPath=".status.templateID"
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".status.type"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].reason"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// InfisicalProjectTemplate is the Schema for the infisicalprojecttemplates API.
type InfisicalProjectTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// Spec defines the desired state of InfisicalProjectTemplate.
	// +required
	Spec InfisicalProjectTemplateSpec `json:"spec"`

	// Status defines the observed state of InfisicalProjectTemplate.
	// +optional
	Status InfisicalProjectTemplateStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// InfisicalProjectTemplateList contains a list of InfisicalProjectTemplate.
type InfisicalProjectTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []InfisicalProjectTemplate `json:"items"`
}
