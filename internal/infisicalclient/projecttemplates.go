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
)

// ProjectTemplateEnvironment is an environment created from a project template.
type ProjectTemplateEnvironment struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Position int32  `json:"position"`
}

// ProjectTemplateUser assigns roles to a user added by a project template.
type ProjectTemplateUser struct {
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}

// ProjectTemplateGroup assigns roles to a group added by a project template.
type ProjectTemplateGroup struct {
	GroupSlug string   `json:"groupSlug"`
	Roles     []string `json:"roles"`
}

// ProjectTemplateIdentity assigns roles to an organization identity added by a template.
type ProjectTemplateIdentity struct {
	IdentityID string   `json:"identityId"`
	Roles      []string `json:"roles"`
}

// ProjectTemplateManagedIdentity describes a project-owned identity created by a template.
type ProjectTemplateManagedIdentity struct {
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
}

// ProjectTemplateRole describes a custom role created by a project template.
type ProjectTemplateRole struct {
	Name        string                  `json:"name"`
	Slug        string                  `json:"slug"`
	Permissions []ProjectRolePermission `json:"permissions"`
}

// ProjectTemplate is an Infisical project template.
type ProjectTemplate struct {
	ID                       string                           `json:"id"`
	Name                     string                           `json:"name"`
	Description              string                           `json:"description"`
	Roles                    []ProjectTemplateRole            `json:"roles"`
	Environments             []ProjectTemplateEnvironment     `json:"environments"`
	OrganizationID           string                           `json:"orgId"`
	Type                     string                           `json:"type"`
	Users                    []ProjectTemplateUser            `json:"users"`
	Groups                   []ProjectTemplateGroup           `json:"groups"`
	Identities               []ProjectTemplateIdentity        `json:"identities"`
	ProjectManagedIdentities []ProjectTemplateManagedIdentity `json:"projectManagedIdentities"`
}

// CreateProjectTemplateRequest is the supported project template creation surface.
type CreateProjectTemplateRequest struct {
	Name                     string                           `json:"name"`
	Description              string                           `json:"description,omitempty"`
	Type                     string                           `json:"type"`
	Roles                    []ProjectTemplateRole            `json:"roles,omitempty"`
	Environments             []ProjectTemplateEnvironment     `json:"environments,omitempty"`
	Users                    []ProjectTemplateUser            `json:"users,omitempty"`
	Groups                   []ProjectTemplateGroup           `json:"groups,omitempty"`
	Identities               []ProjectTemplateIdentity        `json:"identities,omitempty"`
	ProjectManagedIdentities []ProjectTemplateManagedIdentity `json:"projectManagedIdentities,omitempty"`
}

// ProjectTemplatePatch contains mutable project template fields.
type ProjectTemplatePatch struct {
	Name                     string                           `json:"name,omitempty"`
	Description              string                           `json:"description"`
	Roles                    []ProjectTemplateRole            `json:"roles"`
	Environments             []ProjectTemplateEnvironment     `json:"environments"`
	Users                    []ProjectTemplateUser            `json:"users"`
	Groups                   []ProjectTemplateGroup           `json:"groups"`
	Identities               []ProjectTemplateIdentity        `json:"identities"`
	ProjectManagedIdentities []ProjectTemplateManagedIdentity `json:"projectManagedIdentities"`
}

// CreateProjectTemplate creates a project template.
func (c *Client) CreateProjectTemplate(ctx context.Context, request CreateProjectTemplateRequest) (*ProjectTemplate, error) {
	var response struct {
		Template ProjectTemplate `json:"projectTemplate"`
	}
	if err := c.do(ctx, http.MethodPost, "/v1/project-templates", nil, request, &response); err != nil {
		return nil, err
	}
	if err := validateProjectTemplate(&response.Template); err != nil {
		return nil, err
	}
	return &response.Template, nil
}

// GetProjectTemplate retrieves a template by ID.
func (c *Client) GetProjectTemplate(ctx context.Context, id string) (*ProjectTemplate, error) {
	var response struct {
		Template ProjectTemplate `json:"projectTemplate"`
	}
	path := "/v1/project-templates/" + url.PathEscape(id)
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return nil, err
	}
	if err := validateProjectTemplate(&response.Template); err != nil {
		return nil, err
	}
	return &response.Template, nil
}

// ListProjectTemplates lists templates visible to the token.
func (c *Client) ListProjectTemplates(ctx context.Context) ([]ProjectTemplate, error) {
	var response struct {
		Templates []ProjectTemplate `json:"projectTemplates"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/project-templates", nil, nil, &response); err != nil {
		return nil, err
	}
	for i := range response.Templates {
		if err := validateProjectTemplate(&response.Templates[i]); err != nil {
			return nil, err
		}
	}
	return response.Templates, nil
}

// FindProjectTemplate finds a template by its stable name.
func (c *Client) FindProjectTemplate(ctx context.Context, name string) (*ProjectTemplate, error) {
	templates, err := c.ListProjectTemplates(ctx)
	if err != nil {
		return nil, err
	}
	for i := range templates {
		if templates[i].Name == name {
			return &templates[i], nil
		}
	}
	return nil, nil
}

// UpdateProjectTemplate updates mutable project template fields.
func (c *Client) UpdateProjectTemplate(ctx context.Context, id string, patch ProjectTemplatePatch) (*ProjectTemplate, error) {
	var response struct {
		Template ProjectTemplate `json:"projectTemplate"`
	}
	path := "/v1/project-templates/" + url.PathEscape(id)
	if err := c.do(ctx, http.MethodPatch, path, nil, patch, &response); err != nil {
		return nil, err
	}
	if err := validateProjectTemplate(&response.Template); err != nil {
		return nil, err
	}
	return &response.Template, nil
}

// DeleteProjectTemplate deletes a project template by ID.
func (c *Client) DeleteProjectTemplate(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/v1/project-templates/"+url.PathEscape(id), nil, nil, nil)
}

func validateProjectTemplate(template *ProjectTemplate) error {
	if template.ID == "" {
		return fmt.Errorf("infisical API returned a project template without an ID")
	}
	return nil
}
