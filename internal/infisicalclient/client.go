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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxErrorBodySize = 1 << 20

// Client is a small, typed client for the Infisical API used by the operator.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	token      string
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

// New validates an Infisical API URL and returns a client using the supplied token.
func New(baseURL, token string, timeout time.Duration) (*Client, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("infisical bearer token is empty")
	}

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
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		baseURL: parsed,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		token: token,
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
	requestURL := *c.baseURL
	requestURL.Path = strings.TrimRight(c.baseURL.Path, "/") + "/" + strings.TrimLeft(path, "/")
	requestURL.RawQuery = query.Encode()

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
	req.Header.Set("Authorization", "Bearer "+c.token)
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
