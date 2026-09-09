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

// Environment is an Infisical project environment.
type Environment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Project is the Infisical project representation used by the operator.
type Project struct {
	ID                  string        `json:"id"`
	Name                string        `json:"name"`
	Type                string        `json:"type"`
	Slug                string        `json:"slug"`
	OrganizationID      string        `json:"orgId"`
	Description         string        `json:"description"`
	HasDeleteProtection bool          `json:"hasDeleteProtection"`
	Environments        []Environment `json:"environments"`
}

// CreateProjectRequest is the supported project creation surface.
type CreateProjectRequest struct {
	ProjectName             string `json:"projectName"`
	ProjectDescription      string `json:"projectDescription,omitempty"`
	Slug                    string `json:"slug,omitempty"`
	Template                string `json:"template"`
	Type                    string `json:"type"`
	ShouldCreateDefaultEnvs bool   `json:"shouldCreateDefaultEnvs"`
	HasDeleteProtection     bool   `json:"hasDeleteProtection"`
}

// ProjectPatch contains mutable project fields.
type ProjectPatch struct {
	Name                string `json:"name,omitempty"`
	Description         string `json:"description,omitempty"`
	Slug                string `json:"slug,omitempty"`
	HasDeleteProtection *bool  `json:"hasDeleteProtection,omitempty"`
}

// CreateProject creates a project.
func (c *Client) CreateProject(ctx context.Context, request CreateProjectRequest) (*Project, error) {
	var response struct {
		Project Project `json:"project"`
	}
	if err := c.do(ctx, http.MethodPost, "/v1/projects", nil, request, &response); err != nil {
		return nil, err
	}
	if err := validateProject(&response.Project); err != nil {
		return nil, err
	}
	return &response.Project, nil
}

// GetProject retrieves a project by ID.
func (c *Client) GetProject(ctx context.Context, id string) (*Project, error) {
	var response struct {
		Project Project `json:"project"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/projects/"+url.PathEscape(id), nil, nil, &response); err != nil {
		return nil, err
	}
	if err := validateProject(&response.Project); err != nil {
		return nil, err
	}
	return &response.Project, nil
}

// GetProjectBySlug retrieves a project by its stable slug.
func (c *Client) GetProjectBySlug(ctx context.Context, slug string) (*Project, error) {
	var project Project
	if err := c.do(ctx, http.MethodGet, "/v1/projects/slug/"+url.PathEscape(slug), nil, nil, &project); err != nil {
		return nil, err
	}
	if err := validateProject(&project); err != nil {
		return nil, err
	}
	return &project, nil
}

// ListProjects lists projects visible to the token.
func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	var response struct {
		Projects []Project `json:"projects"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/projects", nil, nil, &response); err != nil {
		return nil, err
	}
	for i := range response.Projects {
		if err := validateProject(&response.Projects[i]); err != nil {
			return nil, err
		}
	}
	return response.Projects, nil
}

// FindProject matches an existing project by slug first, then by name.
func (c *Client) FindProject(ctx context.Context, name, slug string) (*Project, error) {
	if slug != "" {
		project, err := c.GetProjectBySlug(ctx, slug)
		if err == nil {
			return project, nil
		}
		if !IsNotFound(err) {
			return nil, err
		}
	}

	projects, err := c.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	for i := range projects {
		if slug != "" && projects[i].Slug == slug {
			return &projects[i], nil
		}
	}
	for i := range projects {
		if projects[i].Name == name {
			return &projects[i], nil
		}
	}
	return nil, nil
}

// UpdateProject updates mutable project fields.
func (c *Client) UpdateProject(ctx context.Context, id string, patch ProjectPatch) (*Project, error) {
	var response struct {
		Project Project `json:"project"`
	}
	if err := c.do(ctx, http.MethodPatch, "/v1/projects/"+url.PathEscape(id), nil, patch, &response); err != nil {
		return nil, err
	}
	if err := validateProject(&response.Project); err != nil {
		return nil, err
	}
	return &response.Project, nil
}

// DeleteProject deletes a project by ID.
func (c *Client) DeleteProject(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/v1/projects/"+url.PathEscape(id), nil, nil, nil)
}

func validateProject(project *Project) error {
	if project.ID == "" {
		return fmt.Errorf("infisical API returned a project without an ID")
	}
	return nil
}
