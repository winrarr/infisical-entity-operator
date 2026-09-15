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
	"encoding/json"
	"net/http"
	"net/url"
)

// Organization is an Infisical top-level organization.
type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// CreateOrganizationRequest is the supported organization creation surface.
type CreateOrganizationRequest struct {
	Name string `json:"name"`
}

// CreateOrganization creates a top-level organization. Infisical requires a user JWT or API key
// for this operation; machine identity tokens cannot create organizations.
func (c *Client) CreateOrganization(ctx context.Context, request CreateOrganizationRequest) (*Organization, error) {
	var response struct {
		Organization Organization `json:"organization"`
	}
	if err := c.do(ctx, http.MethodPost, "/v2/organizations", nil, request, &response); err != nil {
		return nil, err
	}
	if err := validateOrganization(&response.Organization); err != nil {
		return nil, err
	}
	return &response.Organization, nil
}

// GetOrganization retrieves an organization by ID. Infisical currently exposes this endpoint
// to user JWTs, not machine identity access tokens.
func (c *Client) GetOrganization(ctx context.Context, id string) (*Organization, error) {
	var response struct {
		Organization Organization `json:"organization"`
	}
	path := "/v1/organization/" + url.PathEscape(id)
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return nil, err
	}
	if err := validateOrganization(&response.Organization); err != nil {
		return nil, err
	}
	return &response.Organization, nil
}

// ListOrganizations lists organizations visible to the user JWT in the connection.
func (c *Client) ListOrganizations(ctx context.Context) ([]Organization, error) {
	var response struct {
		Organizations []Organization `json:"organizations"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/organization", nil, nil, &response); err != nil {
		return nil, err
	}
	return response.Organizations, nil
}

// FindOrganization finds an explicit organization ID or, when no ID is supplied, an organization
// by name. An identity access token cannot use the user-only organization endpoints, so an
// explicit ID is verified through organization membership access, a non-empty workspace response,
// or, for older Infisical versions, a project visible to that credential.
func (c *Client) FindOrganization(ctx context.Context, id, name string) (*Organization, error) {
	if id != "" {
		organization, err := c.GetOrganization(ctx, id)
		if err == nil {
			return organization, nil
		}
		if !IsUnauthorized(err) {
			return nil, err
		}

		if membershipErr := c.CheckOrganizationMembershipAccess(ctx, id); membershipErr == nil {
			return &Organization{ID: id}, nil
		}
		if workspaces, workspaceErr := c.listOrganizationWorkspaces(ctx, id); workspaceErr == nil && len(workspaces) > 0 {
			return &Organization{ID: id}, nil
		}

		projects, listErr := c.ListProjects(ctx)
		if listErr != nil {
			return nil, err
		}
		for _, project := range projects {
			if project.OrganizationID == id {
				return &Organization{ID: id}, nil
			}
		}
		return nil, nil
	}

	organizations, err := c.ListOrganizations(ctx)
	if err != nil {
		return nil, err
	}
	for i := range organizations {
		if organizations[i].Name == name {
			return &organizations[i], nil
		}
	}
	return nil, nil
}

// CheckOrganizationMembershipAccess verifies that the current token can access an
// organization's membership resource. This endpoint supports machine identity access tokens
// and returns a forbidden response when the token is scoped to another organization.
func (c *Client) CheckOrganizationMembershipAccess(ctx context.Context, organizationID string) error {
	var response struct {
		Users []json.RawMessage `json:"users"`
	}
	path := "/v2/organizations/" + url.PathEscape(organizationID) + "/memberships"
	return c.do(ctx, http.MethodGet, path, nil, nil, &response)
}

func (c *Client) listOrganizationWorkspaces(ctx context.Context, organizationID string) ([]json.RawMessage, error) {
	var response struct {
		Workspaces []json.RawMessage `json:"workspaces"`
	}
	path := "/v2/organizations/" + url.PathEscape(organizationID) + "/workspaces"
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return nil, err
	}
	return response.Workspaces, nil
}

// DeleteOrganization deletes an organization by ID.
func (c *Client) DeleteOrganization(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/v2/organizations/"+url.PathEscape(id), nil, nil, nil)
}

func validateOrganization(organization *Organization) error {
	if organization.ID == "" {
		return &InvalidResponseError{Message: "infisical API returned an organization without an ID"}
	}
	return nil
}
