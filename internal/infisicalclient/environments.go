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

// CreateEnvironmentRequest is the supported environment creation surface.
type CreateEnvironmentRequest struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Position *int32 `json:"position,omitempty"`
}

// EnvironmentPatch contains mutable environment fields.
type EnvironmentPatch struct {
	Name     string `json:"name,omitempty"`
	Position *int32 `json:"position,omitempty"`
}

// GetEnvironmentByID retrieves an environment by ID within a project.
func (c *Client) GetEnvironmentByID(ctx context.Context, projectID, environmentID string) (*Environment, error) {
	var response struct {
		Environment Environment `json:"environment"`
	}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/environments/" + url.PathEscape(environmentID)
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return nil, err
	}
	if err := validateEnvironment(&response.Environment); err != nil {
		return nil, err
	}
	return &response.Environment, nil
}

// GetEnvironmentBySlug retrieves an environment by its stable slug.
func (c *Client) GetEnvironmentBySlug(ctx context.Context, projectID, slug string) (*Environment, error) {
	var response struct {
		Environment Environment `json:"environment"`
	}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/environments/slug/" + url.PathEscape(slug)
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return nil, err
	}
	if err := validateEnvironment(&response.Environment); err != nil {
		return nil, err
	}
	return &response.Environment, nil
}

// CreateEnvironment creates an environment in a project.
func (c *Client) CreateEnvironment(ctx context.Context, projectID string, request CreateEnvironmentRequest) (*Environment, error) {
	var response struct {
		Environment Environment `json:"environment"`
	}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/environments"
	if err := c.do(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return nil, err
	}
	if err := validateEnvironment(&response.Environment); err != nil {
		return nil, err
	}
	return &response.Environment, nil
}

// UpdateEnvironment updates mutable environment fields.
func (c *Client) UpdateEnvironment(ctx context.Context, projectID, environmentID string, patch EnvironmentPatch) (*Environment, error) {
	var response struct {
		Environment Environment `json:"environment"`
	}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/environments/" + url.PathEscape(environmentID)
	if err := c.do(ctx, http.MethodPatch, path, nil, patch, &response); err != nil {
		return nil, err
	}
	if err := validateEnvironment(&response.Environment); err != nil {
		return nil, err
	}
	return &response.Environment, nil
}

// DeleteEnvironment soft-deletes an environment by ID.
func (c *Client) DeleteEnvironment(ctx context.Context, projectID, environmentID string) error {
	path := "/v1/projects/" + url.PathEscape(projectID) + "/environments/" + url.PathEscape(environmentID)
	return c.do(ctx, http.MethodDelete, path, nil, nil, nil)
}

// RestoreEnvironment restores a soft-deleted environment by ID.
func (c *Client) RestoreEnvironment(ctx context.Context, projectID, environmentID string) (*Environment, error) {
	var response struct {
		Environment Environment `json:"environment"`
	}
	path := "/v1/projects/" + url.PathEscape(projectID) + "/environments/" + url.PathEscape(environmentID) + "/restore"
	if err := c.do(ctx, http.MethodPost, path, nil, nil, &response); err != nil {
		return nil, err
	}
	if err := validateEnvironment(&response.Environment); err != nil {
		return nil, err
	}
	return &response.Environment, nil
}

func validateEnvironment(environment *Environment) error {
	if environment.ID == "" {
		return fmt.Errorf("infisical API returned an environment without an ID")
	}
	return nil
}
