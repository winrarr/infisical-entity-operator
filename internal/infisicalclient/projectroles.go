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
	"fmt"
	"net/http"
	"net/url"
)

// ProjectRoleActions preserves Infisical's string-or-array action representation.
// Requests are sent as arrays; responses accept both forms returned by deployed API versions.
type ProjectRoleActions []string

func (actions *ProjectRoleActions) UnmarshalJSON(data []byte) error {
	var values []string
	if len(data) > 0 && data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		values = []string{value}
	} else if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	*actions = values
	return nil
}

func (actions ProjectRoleActions) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string(actions))
}

// ProjectRoleStringCondition describes an Infisical string comparison.
type ProjectRoleStringCondition struct {
	Eq   string   `json:"$eq,omitempty"`
	Ne   string   `json:"$ne,omitempty"`
	In   []string `json:"$in,omitempty"`
	Glob string   `json:"$glob,omitempty"`
}

// ProjectRoleSecretTagsCondition describes secret tag matching.
type ProjectRoleSecretTagsCondition struct {
	In  []string `json:"$in,omitempty"`
	All []string `json:"$all,omitempty"`
}

// ProjectRoleConditions limits a permission to matching project resources.
type ProjectRoleConditions struct {
	Environment *ProjectRoleStringCondition     `json:"environment,omitempty"`
	SecretPath  *ProjectRoleStringCondition     `json:"secretPath,omitempty"`
	SecretName  *ProjectRoleStringCondition     `json:"secretName,omitempty"`
	SecretTags  *ProjectRoleSecretTagsCondition `json:"secretTags,omitempty"`
	EventType   *ProjectRoleStringCondition     `json:"eventType,omitempty"`
}

// ProjectRolePermission defines one role permission rule.
type ProjectRolePermission struct {
	Subject    string                 `json:"subject"`
	Action     ProjectRoleActions     `json:"action"`
	Inverted   bool                   `json:"inverted,omitempty"`
	Conditions *ProjectRoleConditions `json:"conditions,omitempty"`
}

// ProjectRole is an Infisical project role.
type ProjectRole struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Slug        string                  `json:"slug"`
	Description string                  `json:"description"`
	ProjectID   string                  `json:"projectId"`
	Permissions []ProjectRolePermission `json:"permissions"`
}

// CreateProjectRoleRequest is the supported project role creation surface.
type CreateProjectRoleRequest struct {
	Slug        string                  `json:"slug"`
	Name        string                  `json:"name"`
	Description string                  `json:"description,omitempty"`
	Permissions []ProjectRolePermission `json:"permissions"`
}

// ProjectRolePatch contains mutable project role fields.
type ProjectRolePatch struct {
	Name        string                  `json:"name,omitempty"`
	Description string                  `json:"description"`
	Permissions []ProjectRolePermission `json:"permissions,omitempty"`
}

// GetProjectRoleByID retrieves a project role by ID.
func (c *Client) GetProjectRoleByID(ctx context.Context, projectID, roleID string) (*ProjectRole, error) {
	var response struct {
		Role ProjectRole `json:"role"`
	}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/roles/" + url.PathEscape(roleID)
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return nil, err
	}
	if err := validateProjectRole(&response.Role); err != nil {
		return nil, err
	}
	return &response.Role, nil
}

// GetProjectRoleBySlug retrieves a project role by its stable slug.
func (c *Client) GetProjectRoleBySlug(ctx context.Context, projectID, slug string) (*ProjectRole, error) {
	var response struct {
		Role ProjectRole `json:"role"`
	}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/roles/slug/" + url.PathEscape(slug)
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return nil, err
	}
	if err := validateProjectRole(&response.Role); err != nil {
		return nil, err
	}
	return &response.Role, nil
}

// CreateProjectRole creates a custom role in a project.
func (c *Client) CreateProjectRole(ctx context.Context, projectID string, request CreateProjectRoleRequest) (*ProjectRole, error) {
	var response struct {
		Role ProjectRole `json:"role"`
	}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/roles"
	if err := c.do(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return nil, err
	}
	if err := validateProjectRole(&response.Role); err != nil {
		return nil, err
	}
	return &response.Role, nil
}

// UpdateProjectRole updates a custom role in a project.
func (c *Client) UpdateProjectRole(ctx context.Context, projectID, roleID string, patch ProjectRolePatch) (*ProjectRole, error) {
	var response struct {
		Role ProjectRole `json:"role"`
	}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/roles/" + url.PathEscape(roleID)
	if err := c.do(ctx, http.MethodPatch, path, nil, patch, &response); err != nil {
		return nil, err
	}
	if err := validateProjectRole(&response.Role); err != nil {
		return nil, err
	}
	return &response.Role, nil
}

// DeleteProjectRole deletes a custom role from a project.
func (c *Client) DeleteProjectRole(ctx context.Context, projectID, roleID string) error {
	path := "/v1/projects/" + url.PathEscape(projectID) + "/roles/" + url.PathEscape(roleID)
	return c.do(ctx, http.MethodDelete, path, nil, nil, nil)
}

func validateProjectRole(role *ProjectRole) error {
	if role.ID == "" {
		return fmt.Errorf("infisical API returned a project role without an ID")
	}
	return nil
}
