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

// UniversalAuthTrustedIP is an IP address or CIDR range allowed by Universal Auth.
type UniversalAuthTrustedIP struct {
	IPAddress string `json:"ipAddress"`
}

// UniversalAuthConfig is the non-secret Universal Auth configuration returned by Infisical.
type UniversalAuthConfig struct {
	ID                         string                   `json:"id"`
	ClientID                   string                   `json:"clientId"`
	IdentityID                 string                   `json:"identityId"`
	ClientSecretTrustedIPs     []UniversalAuthTrustedIP `json:"clientSecretTrustedIps"`
	AccessTokenTrustedIPs      []UniversalAuthTrustedIP `json:"accessTokenTrustedIps"`
	AccessTokenTTL             int64                    `json:"accessTokenTTL"`
	AccessTokenMaxTTL          int64                    `json:"accessTokenMaxTTL"`
	AccessTokenNumUsesLimit    int64                    `json:"accessTokenNumUsesLimit"`
	AccessTokenPeriod          int64                    `json:"accessTokenPeriod"`
	LockoutEnabled             bool                     `json:"lockoutEnabled"`
	LockoutThreshold           int64                    `json:"lockoutThreshold"`
	LockoutDurationSeconds     int64                    `json:"lockoutDurationSeconds"`
	LockoutCounterResetSeconds int64                    `json:"lockoutCounterResetSeconds"`
}

// UniversalAuthConfigRequest is the supported Universal Auth configuration surface.
type UniversalAuthConfigRequest struct {
	ClientSecretTrustedIPs     []UniversalAuthTrustedIP `json:"clientSecretTrustedIps,omitempty"`
	AccessTokenTrustedIPs      []UniversalAuthTrustedIP `json:"accessTokenTrustedIps,omitempty"`
	AccessTokenTTL             *int64                   `json:"accessTokenTTL,omitempty"`
	AccessTokenMaxTTL          *int64                   `json:"accessTokenMaxTTL,omitempty"`
	AccessTokenNumUsesLimit    *int64                   `json:"accessTokenNumUsesLimit,omitempty"`
	AccessTokenPeriod          *int64                   `json:"accessTokenPeriod,omitempty"`
	LockoutEnabled             *bool                    `json:"lockoutEnabled,omitempty"`
	LockoutThreshold           *int64                   `json:"lockoutThreshold,omitempty"`
	LockoutDurationSeconds     *int64                   `json:"lockoutDurationSeconds,omitempty"`
	LockoutCounterResetSeconds *int64                   `json:"lockoutCounterResetSeconds,omitempty"`
}

// UniversalAuthClientSecret is the metadata for a Universal Auth client secret.
// The secret value is only returned by CreateUniversalAuthClientSecret and is never stored here.
type UniversalAuthClientSecret struct {
	ID                       string `json:"id"`
	Description              string `json:"description"`
	ClientSecretPrefix       string `json:"clientSecretPrefix"`
	ClientSecretNumUses      int64  `json:"clientSecretNumUses"`
	ClientSecretNumUsesLimit int64  `json:"clientSecretNumUsesLimit"`
	ClientSecretTTL          int64  `json:"clientSecretTTL"`
	IdentityUniversalAuthID  string `json:"identityUAId"`
	IsClientSecretRevoked    bool   `json:"isClientSecretRevoked"`
}

// CreateUniversalAuthClientSecretRequest declares a new remote client secret.
type CreateUniversalAuthClientSecretRequest struct {
	Description  string `json:"description,omitempty"`
	NumUsesLimit *int64 `json:"numUsesLimit,omitempty"`
	TTL          *int64 `json:"ttl,omitempty"`
}

// CreateUniversalAuthClientSecretResponse contains the one-time client secret value and metadata.
type CreateUniversalAuthClientSecretResponse struct {
	ClientSecret     string                    `json:"clientSecret"`
	ClientSecretData UniversalAuthClientSecret `json:"clientSecretData"`
}

// AttachUniversalAuth attaches Universal Auth to a machine identity.
func (c *Client) AttachUniversalAuth(ctx context.Context, identityID string, request UniversalAuthConfigRequest) (*UniversalAuthConfig, error) {
	return c.writeUniversalAuth(ctx, http.MethodPost, identityID, request)
}

// GetUniversalAuth retrieves Universal Auth attached to a machine identity.
func (c *Client) GetUniversalAuth(ctx context.Context, identityID string) (*UniversalAuthConfig, error) {
	var response struct {
		IdentityUniversalAuth UniversalAuthConfig `json:"identityUniversalAuth"`
	}
	if err := c.do(ctx, http.MethodGet, universalAuthPath(identityID), nil, nil, &response); err != nil {
		return nil, err
	}
	if err := validateUniversalAuth(&response.IdentityUniversalAuth); err != nil {
		return nil, err
	}
	return &response.IdentityUniversalAuth, nil
}

// UpdateUniversalAuth updates the mutable Universal Auth configuration.
func (c *Client) UpdateUniversalAuth(ctx context.Context, identityID string, request UniversalAuthConfigRequest) (*UniversalAuthConfig, error) {
	return c.writeUniversalAuth(ctx, http.MethodPatch, identityID, request)
}

func (c *Client) writeUniversalAuth(ctx context.Context, method, identityID string, request UniversalAuthConfigRequest) (*UniversalAuthConfig, error) {
	var response struct {
		IdentityUniversalAuth UniversalAuthConfig `json:"identityUniversalAuth"`
	}
	if err := c.do(ctx, method, universalAuthPath(identityID), nil, request, &response); err != nil {
		return nil, err
	}
	if err := validateUniversalAuth(&response.IdentityUniversalAuth); err != nil {
		return nil, err
	}
	return &response.IdentityUniversalAuth, nil
}

// DeleteUniversalAuth removes Universal Auth from a machine identity.
func (c *Client) DeleteUniversalAuth(ctx context.Context, identityID string) error {
	return c.do(ctx, http.MethodDelete, universalAuthPath(identityID), nil, nil, nil)
}

// CreateUniversalAuthClientSecret creates a client secret. The value is returned only once.
func (c *Client) CreateUniversalAuthClientSecret(ctx context.Context, identityID string, request CreateUniversalAuthClientSecretRequest) (*CreateUniversalAuthClientSecretResponse, error) {
	var response CreateUniversalAuthClientSecretResponse
	if err := c.do(ctx, http.MethodPost, universalAuthPath(identityID)+"/client-secrets", nil, request, &response); err != nil {
		return nil, err
	}
	if response.ClientSecret == "" || response.ClientSecretData.ID == "" {
		return nil, fmt.Errorf("infisical API returned an incomplete Universal Auth client secret")
	}
	return &response, nil
}

// ListUniversalAuthClientSecrets lists client-secret metadata without secret values.
func (c *Client) ListUniversalAuthClientSecrets(ctx context.Context, identityID string) ([]UniversalAuthClientSecret, error) {
	var response struct {
		ClientSecretData []UniversalAuthClientSecret `json:"clientSecretData"`
	}
	if err := c.do(ctx, http.MethodGet, universalAuthPath(identityID)+"/client-secrets", nil, nil, &response); err != nil {
		return nil, err
	}
	return response.ClientSecretData, nil
}

// GetUniversalAuthClientSecret retrieves client-secret metadata without its value.
func (c *Client) GetUniversalAuthClientSecret(ctx context.Context, identityID, clientSecretID string) (*UniversalAuthClientSecret, error) {
	var response struct {
		ClientSecretData UniversalAuthClientSecret `json:"clientSecretData"`
	}
	path := universalAuthPath(identityID) + "/client-secrets/" + url.PathEscape(clientSecretID)
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return nil, err
	}
	if response.ClientSecretData.ID == "" {
		return nil, fmt.Errorf("infisical API returned a Universal Auth client secret without an ID")
	}
	return &response.ClientSecretData, nil
}

// RevokeUniversalAuthClientSecret revokes a client secret.
func (c *Client) RevokeUniversalAuthClientSecret(ctx context.Context, identityID, clientSecretID string) error {
	path := universalAuthPath(identityID) + "/client-secrets/" + url.PathEscape(clientSecretID) + "/revoke"
	return c.do(ctx, http.MethodPost, path, nil, nil, nil)
}

func universalAuthPath(identityID string) string {
	return "/v1/auth/universal-auth/identities/" + url.PathEscape(identityID)
}

func validateUniversalAuth(config *UniversalAuthConfig) error {
	if config.ID == "" {
		return fmt.Errorf("infisical API returned Universal Auth without an ID")
	}
	if config.ClientID == "" {
		return fmt.Errorf("infisical API returned Universal Auth without a client ID")
	}
	if config.IdentityID == "" {
		return fmt.Errorf("infisical API returned Universal Auth without an identity ID")
	}
	return nil
}
