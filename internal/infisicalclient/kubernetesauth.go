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

// TrustedIP identifies an IP address or CIDR range.
type TrustedIP struct {
	IPAddress string `json:"ipAddress"`
}

// KubernetesAuth is an Infisical Kubernetes Auth configuration.
type KubernetesAuth struct {
	ID                      string      `json:"id"`
	IdentityID              string      `json:"identityId"`
	KubernetesHost          string      `json:"kubernetesHost"`
	AllowedNamespaces       string      `json:"allowedNamespaces"`
	AllowedNames            string      `json:"allowedNames"`
	AllowedAudience         string      `json:"allowedAudience"`
	TokenReviewMode         string      `json:"tokenReviewMode"`
	GatewayID               string      `json:"gatewayId"`
	GatewayPoolID           string      `json:"gatewayPoolId"`
	VerifyTLSCertificate    bool        `json:"verifyTlsCertificate"`
	CACert                  string      `json:"caCert"`
	TokenReviewerJWT        string      `json:"tokenReviewerJwt"`
	AccessTokenTrustedIPs   []TrustedIP `json:"accessTokenTrustedIps"`
	AccessTokenTTL          int64       `json:"accessTokenTTL"`
	AccessTokenMaxTTL       int64       `json:"accessTokenMaxTTL"`
	AccessTokenNumUsesLimit int64       `json:"accessTokenNumUsesLimit"`
}

// CreateKubernetesAuthRequest is the supported Kubernetes Auth attach surface.
type CreateKubernetesAuthRequest struct {
	KubernetesHost          string      `json:"kubernetesHost,omitempty"`
	CACert                  string      `json:"caCert,omitempty"`
	VerifyTLSCertificate    *bool       `json:"verifyTlsCertificate,omitempty"`
	TokenReviewerJWT        string      `json:"tokenReviewerJwt,omitempty"`
	TokenReviewMode         string      `json:"tokenReviewMode,omitempty"`
	AllowedNamespaces       string      `json:"allowedNamespaces"`
	AllowedNames            string      `json:"allowedNames"`
	AllowedAudience         string      `json:"allowedAudience,omitempty"`
	GatewayID               string      `json:"gatewayId,omitempty"`
	GatewayPoolID           string      `json:"gatewayPoolId,omitempty"`
	AccessTokenTrustedIPs   []TrustedIP `json:"accessTokenTrustedIps,omitempty"`
	AccessTokenTTL          *int64      `json:"accessTokenTTL,omitempty"`
	AccessTokenMaxTTL       *int64      `json:"accessTokenMaxTTL,omitempty"`
	AccessTokenNumUsesLimit *int64      `json:"accessTokenNumUsesLimit,omitempty"`
}

// KubernetesAuthPatch contains optional mutable Kubernetes Auth fields.
type KubernetesAuthPatch struct {
	KubernetesHost          *string      `json:"kubernetesHost,omitempty"`
	CACert                  *string      `json:"caCert,omitempty"`
	VerifyTLSCertificate    *bool        `json:"verifyTlsCertificate,omitempty"`
	TokenReviewerJWT        *string      `json:"tokenReviewerJwt,omitempty"`
	TokenReviewMode         *string      `json:"tokenReviewMode,omitempty"`
	AllowedNamespaces       *string      `json:"allowedNamespaces,omitempty"`
	AllowedNames            *string      `json:"allowedNames,omitempty"`
	AllowedAudience         *string      `json:"allowedAudience,omitempty"`
	GatewayID               *string      `json:"gatewayId,omitempty"`
	GatewayPoolID           *string      `json:"gatewayPoolId,omitempty"`
	AccessTokenTrustedIPs   *[]TrustedIP `json:"accessTokenTrustedIps,omitempty"`
	AccessTokenTTL          *int64       `json:"accessTokenTTL,omitempty"`
	AccessTokenMaxTTL       *int64       `json:"accessTokenMaxTTL,omitempty"`
	AccessTokenNumUsesLimit *int64       `json:"accessTokenNumUsesLimit,omitempty"`
}

// GetKubernetesAuth retrieves Kubernetes Auth for an identity.
func (c *Client) GetKubernetesAuth(ctx context.Context, identityID string) (*KubernetesAuth, error) {
	var response struct {
		Auth KubernetesAuth `json:"identityKubernetesAuth"`
	}
	path := "/v1/auth/kubernetes-auth/identities/" + url.PathEscape(identityID)
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return nil, err
	}
	if err := validateKubernetesAuth(&response.Auth); err != nil {
		return nil, err
	}
	return &response.Auth, nil
}

// AttachKubernetesAuth attaches Kubernetes Auth to an identity.
func (c *Client) AttachKubernetesAuth(ctx context.Context, identityID string, request CreateKubernetesAuthRequest) (*KubernetesAuth, error) {
	var response struct {
		Auth KubernetesAuth `json:"identityKubernetesAuth"`
	}
	path := "/v1/auth/kubernetes-auth/identities/" + url.PathEscape(identityID)
	if err := c.do(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return nil, err
	}
	if err := validateKubernetesAuth(&response.Auth); err != nil {
		return nil, err
	}
	return &response.Auth, nil
}

// UpdateKubernetesAuth updates Kubernetes Auth on an identity.
func (c *Client) UpdateKubernetesAuth(ctx context.Context, identityID string, patch KubernetesAuthPatch) (*KubernetesAuth, error) {
	var response struct {
		Auth KubernetesAuth `json:"identityKubernetesAuth"`
	}
	path := "/v1/auth/kubernetes-auth/identities/" + url.PathEscape(identityID)
	if err := c.do(ctx, http.MethodPatch, path, nil, patch, &response); err != nil {
		return nil, err
	}
	if err := validateKubernetesAuth(&response.Auth); err != nil {
		return nil, err
	}
	return &response.Auth, nil
}

// DeleteKubernetesAuth removes Kubernetes Auth from an identity.
func (c *Client) DeleteKubernetesAuth(ctx context.Context, identityID string) error {
	path := "/v1/auth/kubernetes-auth/identities/" + url.PathEscape(identityID)
	return c.do(ctx, http.MethodDelete, path, nil, nil, nil)
}

func validateKubernetesAuth(auth *KubernetesAuth) error {
	if auth.ID == "" {
		return fmt.Errorf("infisical API returned Kubernetes Auth without an ID")
	}
	return nil
}
