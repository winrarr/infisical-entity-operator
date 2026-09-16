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

// IdentityScope selects the Infisical ownership boundary for a machine identity.
// Project is the backward-compatible default.
// +kubebuilder:validation:Enum=Project;Organization
type IdentityScope string

const (
	// IdentityScopeProject creates a project-managed machine identity.
	IdentityScopeProject IdentityScope = "Project"
	// IdentityScopeOrganization creates an organization-managed machine identity.
	IdentityScopeOrganization IdentityScope = "Organization"
)

// +kubebuilder:validation:XValidation:rule="!has(oldSelf.connectionRef) || self.connectionRef == oldSelf.connectionRef",message="connectionRef is immutable; delete and recreate the InfisicalIdentity"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.scope) || self.scope == oldSelf.scope",message="scope is immutable; delete and recreate the InfisicalIdentity"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.projectRef) || self.projectRef == oldSelf.projectRef",message="projectRef is immutable; delete and recreate the InfisicalIdentity"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.organizationRef) || self.organizationRef == oldSelf.organizationRef",message="organizationRef is immutable; delete and recreate the InfisicalIdentity"
// +kubebuilder:validation:XValidation:rule="self.scope == 'Organization' ? has(self.organizationRef) && (!has(self.projectRef) || size(self.projectRef.name) == 0) : has(self.projectRef) && size(self.projectRef.name) > 0 && !has(self.organizationRef)",message="Project scope requires projectRef and Organization scope requires organizationRef"
// +kubebuilder:validation:XValidation:rule="self.scope == 'Organization' ? (!has(self.roleSlugs) || size(self.roleSlugs) == 0) : (!has(self.projectRoleBindings) || size(self.projectRoleBindings) == 0)",message="roleSlugs is only valid for Project scope and projectRoleBindings is only valid for Organization scope"
// +kubebuilder:validation:XValidation:rule="self.scope == 'Organization' || !has(self.organizationRole)",message="organizationRole is only valid for Organization scope"
// InfisicalIdentitySpec defines the desired state of InfisicalIdentity
type InfisicalIdentitySpec struct {
	// ConnectionRef selects the Infisical API connection.
	ConnectionRef InfisicalConnectionReference `json:"connectionRef"`

	// Scope selects whether Infisical manages this identity at project or organization scope.
	// It defaults to Project. Organization-scoped identities are useful as tenant principals;
	// use organizationRole: admin when the tenant should create its own projects.
	// +optional
	// +kubebuilder:default=Project
	Scope IdentityScope `json:"scope,omitempty"`

	// ProjectRef references the InfisicalProject resource that owns a project-scoped identity.
	// +optional
	ProjectRef *LocalObjectReference `json:"projectRef,omitempty"`

	// OrganizationRef references an InfisicalOrganization resource whose observed organization
	// owns an organization-scoped identity. The project itself need not be listed in projectRoleBindings.
	// +optional
	OrganizationRef *LocalObjectReference `json:"organizationRef,omitempty"`

	// IdentityName is the Infisical identity name. If omitted, metadata.name is used.
	// +optional
	// +kubebuilder:validation:MinLength=1
	IdentityName string `json:"identityName,omitempty"`

	// HasDeleteProtection configures Infisical-side delete protection.
	// +optional
	// +kubebuilder:default=false
	HasDeleteProtection *bool `json:"hasDeleteProtection,omitempty"`

	// Metadata contains optional key/value metadata attached to the identity.
	// +optional
	// +listType=map
	// +listMapKey=key
	Metadata []IdentityMetadata `json:"metadata,omitempty"`

	// RoleSlugs declares the permanent built-in project role slugs assigned to the identity.
	// When omitted, the existing project membership is not managed. Use no-access
	// explicitly when the identity should have no project permissions.
	// +optional
	// +listType=atomic
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:Items:Enum=admin;member;viewer;no-access
	RoleSlugs []string `json:"roleSlugs,omitempty"`

	// OrganizationRole is the built-in Infisical organization role for an organization-scoped identity.
	// Leave it empty to use Infisical's least-privilege no-access role. Project-scoped identities
	// must omit this field.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Enum=admin;member;no-access
	OrganizationRole string `json:"organizationRole,omitempty"`

	// ProjectRoleBindings grants an organization-scoped identity roles in selected projects.
	// A binding manages that project's complete permanent role list; omitted projects are left
	// unmanaged. Every referenced project must belong to the organization from organizationRef.
	// +optional
	// +listType=atomic
	ProjectRoleBindings []IdentityProjectRoleBinding `json:"projectRoleBindings,omitempty"`

	// CreationPolicy controls whether the operator creates or adopts an identity.
	// +optional
	// +kubebuilder:default=Create
	// +kubebuilder:validation:Enum=Create;Adopt;CreateOrAdopt
	CreationPolicy CreationPolicy `json:"creationPolicy,omitempty"`

	// DeletionPolicy controls whether the external identity is deleted with this resource.
	// +optional
	// +kubebuilder:default=Orphan
	// +kubebuilder:validation:Enum=Delete;Orphan
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
}

// InfisicalIdentityStatus defines the observed state of InfisicalIdentity.
type InfisicalIdentityStatus struct {
	// IdentityID is the immutable Infisical identifier once created or adopted.
	IdentityID string `json:"identityID,omitempty"`

	// ProjectID is the observed owning project identifier.
	ProjectID string `json:"projectID,omitempty"`

	// OrganizationID is the observed owning organization identifier for an organization-scoped identity.
	OrganizationID string `json:"organizationID,omitempty"`

	// OrganizationRole is the observed organization role for an organization-scoped identity.
	OrganizationRole string `json:"organizationRole,omitempty"`

	// MembershipID is the Infisical project membership identifier when role management is enabled.
	MembershipID string `json:"membershipID,omitempty"`

	// Roles contains the observed project roles when role management is enabled.
	// +optional
	// +listType=atomic
	Roles []IdentityRoleStatus `json:"roles,omitempty"`

	// ProjectMemberships contains observed memberships managed through projectRoleBindings.
	// +optional
	// +listType=atomic
	ProjectMemberships []IdentityProjectMembershipStatus `json:"projectMemberships,omitempty"`

	// ObservedGeneration is the most recent generation reflected in status.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of the identity.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// IdentityProjectRoleBinding declares the permanent project roles for an organization-scoped identity.
type IdentityProjectRoleBinding struct {
	// ProjectRef references the project receiving this identity's roles.
	ProjectRef LocalObjectReference `json:"projectRef"`

	// RoleSlugs is the complete permanent role set for this project membership.
	// +listType=atomic
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:Items:Enum=admin;member;viewer;no-access
	RoleSlugs []string `json:"roleSlugs"`
}

// IdentityProjectMembershipStatus describes an observed project membership for an organization-scoped identity.
type IdentityProjectMembershipStatus struct {
	// ProjectID is the Infisical project identifier.
	ProjectID string `json:"projectID,omitempty"`

	// MembershipID is the Infisical project membership identifier.
	MembershipID string `json:"membershipID,omitempty"`

	// Roles contains the observed permanent or temporary roles.
	// +listType=atomic
	Roles []IdentityRoleStatus `json:"roles,omitempty"`
}

// IdentityRoleStatus describes an observed permanent or temporary project role.
type IdentityRoleStatus struct {
	// RoleID is the Infisical role assignment identifier.
	RoleID string `json:"roleID,omitempty"`

	// Slug is the observed project role slug.
	Slug string `json:"slug,omitempty"`

	// Name is the observed project role name when available.
	Name string `json:"name,omitempty"`

	// IsTemporary reports whether Infisical marked the assignment as temporary.
	IsTemporary bool `json:"isTemporary,omitempty"`
}

// +kubebuilder:printcolumn:name="Identity ID",type="string",JSONPath=".status.identityID"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].reason"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// InfisicalIdentity is the Schema for the infisicalidentities API
type InfisicalIdentity struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of InfisicalIdentity
	// +required
	Spec InfisicalIdentitySpec `json:"spec"`

	// status defines the observed state of InfisicalIdentity
	// +optional
	Status InfisicalIdentityStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// InfisicalIdentityList contains a list of InfisicalIdentity
type InfisicalIdentityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []InfisicalIdentity `json:"items"`
}
