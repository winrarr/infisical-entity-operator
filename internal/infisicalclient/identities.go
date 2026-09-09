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

package infisicalclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// IdentityMetadata is metadata attached to a machine identity.
type IdentityMetadata struct {
	ID    string `json:"id,omitempty"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Identity is a project-managed Infisical machine identity.
type Identity struct {
	ID                  string             `json:"id"`
	Name                string             `json:"name"`
	ProjectID           string             `json:"projectId"`
	HasDeleteProtection bool               `json:"hasDeleteProtection"`
	Metadata            []IdentityMetadata `json:"metadata"`
}

// IdentityMembershipRole is a role assigned to a project identity membership.
type IdentityMembershipRole struct {
	ID                   string `json:"id,omitempty"`
	Role                 string `json:"role,omitempty"`
	IsTemporary          bool   `json:"isTemporary,omitempty"`
	CustomRoleID         string `json:"customRoleId,omitempty"`
	CustomRoleName       string `json:"customRoleName,omitempty"`
	CustomRoleSlug       string `json:"customRoleSlug,omitempty"`
	TemporaryMode        string `json:"temporaryMode,omitempty"`
	TemporaryRange       string `json:"temporaryRange,omitempty"`
	TemporaryAccessStart string `json:"temporaryAccessStartTime,omitempty"`
	TemporaryAccessEnd   string `json:"temporaryAccessEndTime,omitempty"`
}

// Slug returns the stable built-in or custom role slug.
func (role IdentityMembershipRole) Slug() string {
	if role.CustomRoleSlug != "" {
		return role.CustomRoleSlug
	}
	return role.Role
}

// Name returns the most specific available role name.
func (role IdentityMembershipRole) Name() string {
	if role.CustomRoleName != "" {
		return role.CustomRoleName
	}
	return role.Role
}

// IdentityMembership is a project identity membership and its assigned roles.
type IdentityMembership struct {
	ID         string                   `json:"id"`
	ProjectID  string                   `json:"projectId"`
	IdentityID string                   `json:"identityId"`
	Roles      []IdentityMembershipRole `json:"roles"`
}

// IdentityMembershipRoleRequest is one permanent role assignment request.
type IdentityMembershipRoleRequest struct {
	Role        string `json:"role"`
	IsTemporary bool   `json:"isTemporary"`
}

// IdentityMembershipRequest is the supported permanent role assignment surface.
type IdentityMembershipRequest struct {
	Roles []IdentityMembershipRoleRequest `json:"roles"`
}

// CreateIdentityRequest is the supported identity creation surface.
type CreateIdentityRequest struct {
	Name                string             `json:"name"`
	HasDeleteProtection bool               `json:"hasDeleteProtection"`
	Metadata            []IdentityMetadata `json:"metadata,omitempty"`
}

// IdentityPatch contains mutable identity fields.
type IdentityPatch struct {
	Name                string              `json:"name,omitempty"`
	HasDeleteProtection *bool               `json:"hasDeleteProtection,omitempty"`
	Metadata            *[]IdentityMetadata `json:"metadata,omitempty"`
}

// CreateIdentity creates an identity inside a project.
func (c *Client) CreateIdentity(ctx context.Context, projectID string, request CreateIdentityRequest) (*Identity, error) {
	var response struct {
		Identity Identity `json:"identity"`
	}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/identities"
	if err := c.do(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return nil, err
	}
	if err := validateIdentity(&response.Identity); err != nil {
		return nil, err
	}
	return &response.Identity, nil
}

// GetIdentity retrieves a project-managed identity by ID.
func (c *Client) GetIdentity(ctx context.Context, projectID, identityID string) (*Identity, error) {
	var response struct {
		Identity Identity `json:"identity"`
	}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/identities/" + url.PathEscape(identityID)
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return nil, err
	}
	if err := validateIdentity(&response.Identity); err != nil {
		return nil, err
	}
	return &response.Identity, nil
}

// ListIdentities lists identities in a project.
func (c *Client) ListIdentities(ctx context.Context, projectID string) ([]Identity, error) {
	var response struct {
		Identities []Identity `json:"identities"`
	}
	query := url.Values{
		"limit":  []string{strconv.Itoa(1000)},
		"offset": []string{"0"},
	}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/identities"
	if err := c.do(ctx, http.MethodGet, path, query, nil, &response); err != nil {
		return nil, err
	}
	for i := range response.Identities {
		if err := validateIdentity(&response.Identities[i]); err != nil {
			return nil, err
		}
	}
	return response.Identities, nil
}

// FindIdentity matches an existing identity by name.
func (c *Client) FindIdentity(ctx context.Context, projectID, name string) (*Identity, error) {
	identities, err := c.ListIdentities(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for i := range identities {
		if identities[i].Name == name {
			return &identities[i], nil
		}
	}
	return nil, nil
}

// UpdateIdentity updates mutable identity fields.
func (c *Client) UpdateIdentity(ctx context.Context, projectID, identityID string, patch IdentityPatch) (*Identity, error) {
	var response struct {
		Identity Identity `json:"identity"`
	}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/identities/" + url.PathEscape(identityID)
	if err := c.do(ctx, http.MethodPatch, path, nil, patch, &response); err != nil {
		return nil, err
	}
	if err := validateIdentity(&response.Identity); err != nil {
		return nil, err
	}
	return &response.Identity, nil
}

// DeleteIdentity deletes a project-managed identity.
func (c *Client) DeleteIdentity(ctx context.Context, projectID, identityID string) error {
	path := "/v1/projects/" + url.PathEscape(projectID) + "/identities/" + url.PathEscape(identityID)
	return c.do(ctx, http.MethodDelete, path, nil, nil, nil)
}

// GetIdentityMembership retrieves an identity's project membership and roles.
func (c *Client) GetIdentityMembership(ctx context.Context, projectID, identityID string) (*IdentityMembership, error) {
	var response struct {
		IdentityMembership IdentityMembership `json:"identityMembership"`
	}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/memberships/identities/" + url.PathEscape(identityID)
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return nil, err
	}
	if err := validateIdentityMembership(&response.IdentityMembership); err != nil {
		return nil, err
	}
	return &response.IdentityMembership, nil
}

// CreateIdentityMembership creates a project identity membership with permanent roles.
func (c *Client) CreateIdentityMembership(ctx context.Context, projectID, identityID string, roleSlugs []string) (*IdentityMembership, error) {
	var response struct {
		IdentityMembership IdentityMembership `json:"identityMembership"`
	}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/memberships/identities/" + url.PathEscape(identityID)
	if err := c.do(ctx, http.MethodPost, path, nil, identityMembershipRequest(roleSlugs), &response); err != nil {
		return nil, err
	}
	if err := validateIdentityMembership(&response.IdentityMembership); err != nil {
		return nil, err
	}
	return &response.IdentityMembership, nil
}

// UpdateIdentityMembership replaces a project identity membership's roles.
func (c *Client) UpdateIdentityMembership(ctx context.Context, projectID, identityID string, roleSlugs []string) (*IdentityMembership, error) {
	var response struct {
		IdentityMembership IdentityMembership `json:"identityMembership"`
	}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/memberships/identities/" + url.PathEscape(identityID)
	if err := c.do(ctx, http.MethodPatch, path, nil, identityMembershipRequest(roleSlugs), &response); err != nil {
		return nil, err
	}
	if err := validateIdentityMembership(&response.IdentityMembership); err != nil {
		return nil, err
	}
	return &response.IdentityMembership, nil
}

func identityMembershipRequest(roleSlugs []string) IdentityMembershipRequest {
	roles := make([]IdentityMembershipRoleRequest, 0, len(roleSlugs))
	for _, slug := range roleSlugs {
		roles = append(roles, IdentityMembershipRoleRequest{Role: slug})
	}
	return IdentityMembershipRequest{Roles: roles}
}

func validateIdentity(identity *Identity) error {
	if identity.ID == "" {
		return fmt.Errorf("infisical API returned an identity without an ID")
	}
	return nil
}

func validateIdentityMembership(membership *IdentityMembership) error {
	if membership.ID == "" {
		return fmt.Errorf("infisical API returned an identity membership without an ID")
	}
	return nil
}
