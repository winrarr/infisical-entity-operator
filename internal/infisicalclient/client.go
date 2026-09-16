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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	maxErrorBodySize         = 1 << 20
	paginationLimitParameter = "limit"
)

// Client is a small, typed client for the Infisical API used by the operator.
type Client struct {
	baseURL       *url.URL
	httpClient    *http.Client
	token         string
	universalAuth *universalAuthCredentials
	tokenMu       sync.Mutex
}

type universalAuthCredentials struct {
	clientID         string
	clientSecret     string
	organizationSlug string
	token            string
	expiresAt        time.Time
}

// HTTPError represents a non-successful Infisical API response.
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("Infisical API returned HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("Infisical API returned HTTP %d: %s", e.StatusCode, e.Body)
}

// IsNotFound reports whether err is an Infisical 404 response.
func IsNotFound(err error) bool {
	var httpErr *HTTPError
	return errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound
}

// IsUnauthorized reports whether err is an authentication or authorization response.
func IsUnauthorized(err error) bool {
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		return false
	}
	return httpErr.StatusCode == http.StatusUnauthorized || httpErr.StatusCode == http.StatusForbidden
}

// InvalidResponseError reports a successful response that does not satisfy the client's contract.
type InvalidResponseError struct {
	Message string
}

func (e *InvalidResponseError) Error() string { return e.Message }

// TokenIdentityID returns the machine identity ID in a JWT access token. It returns an
// empty string for user tokens, API keys, and tokens without the standard identityId claim.
// The value is only used as an API resource identifier; Infisical still authorizes every
// request using the original token.
func (c *Client) TokenIdentityID() string {
	return tokenIdentityID(c.token)
}

// TokenIdentityIDContext returns the machine identity ID in the current access token.
// Universal Auth clients obtain a token lazily, so this method performs the exchange when
// necessary. It is intended for controllers that need the caller identity as an API ID.
func (c *Client) TokenIdentityIDContext(ctx context.Context) (string, error) {
	token, err := c.bearerToken(ctx)
	if err != nil {
		return "", err
	}
	return tokenIdentityID(token), nil
}

func tokenIdentityID(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}
	claims, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var payload struct {
		IdentityID string `json:"identityId"`
	}
	if err := json.Unmarshal(claims, &payload); err != nil {
		return ""
	}
	return payload.IdentityID
}

// New validates an Infisical API URL and returns a client using the supplied token.
func New(baseURL, token string, timeout time.Duration) (*Client, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("infisical bearer token is empty")
	}

	parsed, err := parseBaseURL(baseURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		baseURL:    parsed,
		httpClient: newHTTPClient(timeout),
		token:      token,
	}, nil
}

// NewWithUniversalAuth validates an Infisical API URL and returns a client that exchanges
// the supplied Universal Auth credentials for cached short-lived bearer tokens on demand.
func NewWithUniversalAuth(baseURL, clientID, clientSecret, organizationSlug string, timeout time.Duration) (*Client, error) {
	if strings.TrimSpace(clientID) == "" {
		return nil, errors.New("infisical Universal Auth client ID is empty")
	}
	if strings.TrimSpace(clientSecret) == "" {
		return nil, errors.New("infisical Universal Auth client secret is empty")
	}
	parsed, err := parseBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	return &Client{
		baseURL:    parsed,
		httpClient: newHTTPClient(timeout),
		universalAuth: &universalAuthCredentials{
			clientID:         clientID,
			clientSecret:     clientSecret,
			organizationSlug: organizationSlug,
		},
	}, nil
}

// Check authenticates a request and verifies that the configured API is usable.
func (c *Client) Check(ctx context.Context) error {
	var result struct {
		Projects []Project `json:"projects"`
	}
	return c.do(ctx, http.MethodGet, "/v1/projects", nil, nil, &result)
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body, target any) error {
	token, err := c.bearerToken(ctx)
	if err != nil {
		return err
	}
	return c.doWithToken(ctx, token, method, path, query, body, target)
}

func (c *Client) bearerToken(ctx context.Context) (string, error) {
	if c.universalAuth == nil {
		return c.token, nil
	}

	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	if c.universalAuth.token != "" && time.Now().Before(c.universalAuth.expiresAt) {
		return c.universalAuth.token, nil
	}

	var response struct {
		AccessToken string  `json:"accessToken"`
		ExpiresIn   float64 `json:"expiresIn"`
		TokenType   string  `json:"tokenType"`
	}
	request := struct {
		ClientID         string `json:"clientId"`
		ClientSecret     string `json:"clientSecret"`
		OrganizationSlug string `json:"organizationSlug,omitempty"`
	}{
		ClientID:         c.universalAuth.clientID,
		ClientSecret:     c.universalAuth.clientSecret,
		OrganizationSlug: c.universalAuth.organizationSlug,
	}
	if err := c.doWithToken(ctx, "", http.MethodPost, "/v1/auth/universal-auth/login", nil, request, &response); err != nil {
		return "", fmt.Errorf("exchange Infisical Universal Auth credentials: %w", err)
	}
	if strings.TrimSpace(response.AccessToken) == "" {
		return "", errors.New("infisical Universal Auth login returned an empty access token")
	}
	if response.TokenType != "" && !strings.EqualFold(response.TokenType, "Bearer") {
		return "", fmt.Errorf("infisical Universal Auth login returned unsupported token type %q", response.TokenType)
	}
	c.universalAuth.token = response.AccessToken
	// Refresh before the server-side expiry so a long request does not start with an
	// already-expired token. Infisical returns seconds as a number in its OpenAPI schema.
	validFor := time.Duration(response.ExpiresIn * float64(time.Second))
	refreshSkew := 30 * time.Second
	if validFor <= refreshSkew {
		refreshSkew = validFor / 2
	}
	if refreshSkew < 0 {
		refreshSkew = 0
	}
	c.universalAuth.expiresAt = time.Now().Add(validFor - refreshSkew)
	return c.universalAuth.token, nil
}

func (c *Client) doWithToken(ctx context.Context, token, method, path string, query url.Values, body, target any) error {
	requestURL := *c.baseURL
	requestURL.Path = strings.TrimRight(c.baseURL.Path, "/") + "/" + strings.TrimLeft(path, "/")
	if query != nil {
		requestURL.RawQuery = query.Encode()
	}

	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode Infisical API request: %w", err)
		}
		requestBody = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), requestBody)
	if err != nil {
		return fmt.Errorf("create Infisical API request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call Infisical API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodySize))
		if readErr != nil {
			return fmt.Errorf("read Infisical API error response: %w", readErr)
		}
		return &HTTPError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(bodyBytes))}
	}
	if target == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode Infisical API response: %w", err)
	}
	return nil
}

func parseBaseURL(baseURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil {
		return nil, fmt.Errorf("parse Infisical API URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("infisical API URL must use http or https, got %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return nil, errors.New("infisical API URL has no host")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("infisical API URL must not contain a query or fragment")
	}
	return parsed, nil
}

func newHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &http.Client{Timeout: timeout}
}
