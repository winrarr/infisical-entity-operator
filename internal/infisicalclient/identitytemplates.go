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

// IdentityTemplateFields is the union of the currently documented identity-template fields.
// Only fields for the selected authentication method are sent by the controller.
type IdentityTemplateFields struct {
	URL                  string `json:"url,omitempty"`
	BindDN               string `json:"bindDN,omitempty"`
	BindPass             string `json:"bindPass,omitempty"`
	SearchBase           string `json:"searchBase,omitempty"`
	LDAPCACertificate    string `json:"ldapCaCertificate,omitempty"`
	TokenReviewMode      string `json:"tokenReviewMode,omitempty"`
	KubernetesHost       string `json:"kubernetesHost,omitempty"`
	CACert               string `json:"caCert,omitempty"`
	VerifyTLSCertificate *bool  `json:"verifyTlsCertificate,omitempty"`
	TokenReviewerJWT     string `json:"tokenReviewerJwt,omitempty"`
	GatewayID            string `json:"gatewayId,omitempty"`
	GatewayPoolID        string `json:"gatewayPoolId,omitempty"`
	AllowedAudience      string `json:"allowedAudience,omitempty"`
	OIDCDiscoveryURL     string `json:"oidcDiscoveryUrl,omitempty"`
	BoundIssuer          string `json:"boundIssuer,omitempty"`
	BoundAudiences       string `json:"boundAudiences,omitempty"`
	HasBindPass          bool   `json:"hasBindPass,omitempty"`
	HasTokenReviewerJWT  bool   `json:"hasTokenReviewerJwt,omitempty"`
}

// IdentityTemplate is an Infisical identity authentication template.
type IdentityTemplate struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	OrganizationID string                 `json:"orgId"`
	AuthMethod     string                 `json:"authMethod"`
	TemplateFields IdentityTemplateFields `json:"templateFields"`
}

// CreateIdentityTemplateRequest is the supported identity-template creation surface.
type CreateIdentityTemplateRequest struct {
	Name           string                 `json:"name"`
	AuthMethod     string                 `json:"authMethod"`
	TemplateFields IdentityTemplateFields `json:"templateFields"`
}

// IdentityTemplatePatch contains mutable identity-template fields.
type IdentityTemplatePatch struct {
	Name           string                 `json:"name,omitempty"`
	TemplateFields IdentityTemplateFields `json:"templateFields"`
}

// CreateIdentityTemplate creates an identity authentication template.
func (c *Client) CreateIdentityTemplate(ctx context.Context, request CreateIdentityTemplateRequest) (*IdentityTemplate, error) {
	var response IdentityTemplate
	if err := c.do(ctx, http.MethodPost, "/v1/identity-templates", nil, request, &response); err != nil {
		return nil, err
	}
	if err := validateIdentityTemplate(&response); err != nil {
		return nil, err
	}
	return &response, nil
}

// GetIdentityTemplate retrieves an identity authentication template by ID.
func (c *Client) GetIdentityTemplate(ctx context.Context, id string) (*IdentityTemplate, error) {
	var response IdentityTemplate
	path := "/v1/identity-templates/" + url.PathEscape(id)
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return nil, err
	}
	if err := validateIdentityTemplate(&response); err != nil {
		return nil, err
	}
	return &response, nil
}

// FindIdentityTemplate finds a template by its stable name.
func (c *Client) FindIdentityTemplate(ctx context.Context, name string) (*IdentityTemplate, error) {
	query := url.Values{paginationLimitParameter: {strconv.Itoa(100)}, "search": {name}}
	var response struct {
		Templates []IdentityTemplate `json:"templates"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/identity-templates/search", query, nil, &response); err != nil {
		return nil, err
	}
	for i := range response.Templates {
		if err := validateIdentityTemplate(&response.Templates[i]); err != nil {
			return nil, err
		}
		if response.Templates[i].Name == name {
			return &response.Templates[i], nil
		}
	}
	return nil, nil
}

// UpdateIdentityTemplate updates an identity authentication template.
func (c *Client) UpdateIdentityTemplate(ctx context.Context, id string, patch IdentityTemplatePatch) (*IdentityTemplate, error) {
	var response IdentityTemplate
	path := "/v1/identity-templates/" + url.PathEscape(id)
	if err := c.do(ctx, http.MethodPatch, path, nil, patch, &response); err != nil {
		return nil, err
	}
	if err := validateIdentityTemplate(&response); err != nil {
		return nil, err
	}
	return &response, nil
}

// DeleteIdentityTemplate deletes an identity authentication template by ID.
func (c *Client) DeleteIdentityTemplate(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/v1/identity-templates/"+url.PathEscape(id), nil, nil, nil)
}

func validateIdentityTemplate(template *IdentityTemplate) error {
	if template.ID == "" {
		return fmt.Errorf("infisical API returned an identity template without an ID")
	}
	if template.AuthMethod == "" {
		return fmt.Errorf("infisical API returned an identity template without an auth method")
	}
	return nil
}
