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
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const (
	testOrganizationID                 = "org-1"
	testOrganizationMemberRole         = "member"
	testOrganizationIdentityPath       = "/api/v1/identities/identity-1"
	testOrganizationByIDPath           = "/api/v1/organization/org-1"
	testTenantName                     = "tenant"
	testIdentityID                     = "identity-1"
	testUniversalAuthClientSecretID    = "client-secret-1"
	testUniversalAuthClientSecretValue = "secret-1"
	testUniversalAuthIdentityPath      = "/api/v1/auth/universal-auth/identities/identity-1"
)

func TestClientUsesAPIPathAndBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/projects/project-1" {
			t.Errorf("unexpected path: %s", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer secret-token" {
			t.Errorf("unexpected authorization header: %s", request.Header.Get("Authorization"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"project":{"id":"project-1","name":"demo"}}`))
	}))
	defer server.Close()

	client, err := New(server.URL+"/api", "secret-token", time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	project, err := client.GetProject(context.Background(), "project-1")
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if project.ID != "project-1" || project.Name != "demo" {
		t.Fatalf("unexpected project: %#v", project)
	}
}

func TestUniversalAuthClientExchangesAndCachesToken(t *testing.T) {
	claims := base64.RawURLEncoding.EncodeToString([]byte(`{"identityId":"identity-1"}`))
	accessToken := "header." + claims + ".signature"
	loginRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/api/v1/auth/universal-auth/login":
			loginRequests++
			if request.Header.Get("Authorization") != "" {
				t.Errorf("Universal Auth login unexpectedly used a bearer token")
			}
			var body struct {
				ClientID         string `json:"clientId"`
				ClientSecret     string `json:"clientSecret"`
				OrganizationSlug string `json:"organizationSlug"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode Universal Auth login: %v", err)
			}
			if body.ClientID != "client-1" || body.ClientSecret != "secret-1" || body.OrganizationSlug != "tenant" {
				t.Errorf("unexpected Universal Auth login: %#v", body)
			}
			_, _ = writer.Write([]byte(`{"accessToken":"` + accessToken + `","expiresIn":3600,"accessTokenMaxTTL":3600,"tokenType":"Bearer"}`))
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/projects":
			if request.Header.Get("Authorization") != "Bearer "+accessToken {
				t.Errorf("unexpected cached bearer token: %s", request.Header.Get("Authorization"))
			}
			_, _ = writer.Write([]byte(`{"projects":[]}`))
		default:
			http.Error(writer, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewWithUniversalAuth(server.URL+"/api", "client-1", "secret-1", "tenant", time.Second)
	if err != nil {
		t.Fatalf("new Universal Auth client: %v", err)
	}
	if err := client.Check(context.Background()); err != nil {
		t.Fatalf("check Universal Auth client: %v", err)
	}
	if err := client.Check(context.Background()); err != nil {
		t.Fatalf("check cached Universal Auth client: %v", err)
	}
	if loginRequests != 1 {
		t.Fatalf("expected one token exchange, got %d", loginRequests)
	}
	if got, err := client.TokenIdentityIDContext(context.Background()); err != nil || got != testIdentityID {
		t.Fatalf("unexpected Universal Auth identity ID: %q, %v", got, err)
	}
}

func TestUniversalAuthClientUsesLifecycleEndpoints(t *testing.T) {
	configResponse := `{"identityUniversalAuth":{"id":"ua-1","clientId":"client-1","identityId":"identity-1"}}`
	responses := map[string]string{
		http.MethodPost + " " + testUniversalAuthIdentityPath:                                            configResponse,
		http.MethodGet + " " + testUniversalAuthIdentityPath:                                             configResponse,
		http.MethodPatch + " " + testUniversalAuthIdentityPath:                                           configResponse,
		http.MethodPost + " " + testUniversalAuthIdentityPath + "/client-secrets":                        `{"clientSecret":"secret-1","clientSecretData":{"id":"client-secret-1","identityUAId":"ua-1"}}`,
		http.MethodGet + " " + testUniversalAuthIdentityPath + "/client-secrets":                         `{"clientSecretData":[{"id":"client-secret-1","identityUAId":"ua-1"}]}`,
		http.MethodGet + " " + testUniversalAuthIdentityPath + "/client-secrets/client-secret-1":         `{"clientSecretData":{"id":"client-secret-1","identityUAId":"ua-1"}}`,
		http.MethodDelete + " " + testUniversalAuthIdentityPath:                                          "",
		http.MethodPost + " " + testUniversalAuthIdentityPath + "/client-secrets/client-secret-1/revoke": "",
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		response, ok := responses[request.Method+" "+request.URL.Path]
		if !ok {
			http.Error(writer, "unexpected request", http.StatusNotFound)
			return
		}
		if response != "" {
			_, _ = writer.Write([]byte(response))
		}
	}))
	defer server.Close()

	client, err := New(server.URL+"/api", "secret-token", time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	configRequest := UniversalAuthConfigRequest{}
	if config, err := client.AttachUniversalAuth(context.Background(), testIdentityID, configRequest); err != nil || config.ID != "ua-1" {
		t.Fatalf("attach Universal Auth: %#v, %v", config, err)
	}
	if config, err := client.GetUniversalAuth(context.Background(), testIdentityID); err != nil || config.IdentityID != testIdentityID {
		t.Fatalf("get Universal Auth: %#v, %v", config, err)
	}
	if _, err := client.UpdateUniversalAuth(context.Background(), testIdentityID, configRequest); err != nil {
		t.Fatalf("update Universal Auth: %v", err)
	}
	created, err := client.CreateUniversalAuthClientSecret(context.Background(), testIdentityID, CreateUniversalAuthClientSecretRequest{})
	if err != nil || created.ClientSecret != testUniversalAuthClientSecretValue {
		t.Fatalf("create Universal Auth client secret: %#v, %v", created, err)
	}
	if secrets, err := client.ListUniversalAuthClientSecrets(context.Background(), testIdentityID); err != nil || len(secrets) != 1 {
		t.Fatalf("list Universal Auth client secrets: %#v, %v", secrets, err)
	}
	if secret, err := client.GetUniversalAuthClientSecret(context.Background(), testIdentityID, testUniversalAuthClientSecretID); err != nil || secret.ID != testUniversalAuthClientSecretID {
		t.Fatalf("get Universal Auth client secret: %#v, %v", secret, err)
	}
	if err := client.RevokeUniversalAuthClientSecret(context.Background(), testIdentityID, testUniversalAuthClientSecretID); err != nil {
		t.Fatalf("revoke Universal Auth client secret: %v", err)
	}
	if err := client.DeleteUniversalAuth(context.Background(), testIdentityID); err != nil {
		t.Fatalf("delete Universal Auth: %v", err)
	}
}

func TestClientClassifiesNotFound(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	client, err := New(server.URL, "secret-token", time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	_, err = client.GetProject(context.Background(), "missing")
	if !IsNotFound(err) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestNewRejectsInvalidURLAndEmptyToken(t *testing.T) {
	if _, err := New("localhost:8080/api", "secret-token", time.Second); err == nil {
		t.Fatal("expected invalid URL error")
	}
	if _, err := New("https://app.infisical.com/api", "", time.Second); err == nil {
		t.Fatal("expected empty token error")
	}
}

func TestOrganizationClientUsesOrganizationEndpoints(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/api/v2/organizations":
			requests++
			_, _ = writer.Write([]byte(`{"organization":{"id":"org-1","name":"tenant","slug":"tenant"}}`))
		case request.Method == http.MethodGet && request.URL.Path == testOrganizationByIDPath:
			requests++
			_, _ = writer.Write([]byte(`{"organization":{"id":"org-1","name":"tenant","slug":"tenant"}}`))
		case request.Method == http.MethodDelete && request.URL.Path == "/api/v2/organizations/org-1":
			requests++
		default:
			http.Error(writer, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := New(server.URL+"/api", "secret-token", time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	created, err := client.CreateOrganization(context.Background(), CreateOrganizationRequest{Name: testTenantName})
	if err != nil || created.ID != testOrganizationID {
		t.Fatalf("create organization: %#v, %v", created, err)
	}
	found, err := client.GetOrganization(context.Background(), testOrganizationID)
	if err != nil || found.Slug != testTenantName {
		t.Fatalf("get organization: %#v, %v", found, err)
	}
	if err := client.DeleteOrganization(context.Background(), testOrganizationID); err != nil {
		t.Fatalf("delete organization: %v", err)
	}
	if requests != 3 {
		t.Fatalf("expected three organization requests, got %d", requests)
	}
}

func TestFindOrganizationUsesMachineIdentityWorkspaceAccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.Method == http.MethodGet && request.URL.Path == testOrganizationByIDPath {
			http.Error(writer, "user JWT required", http.StatusForbidden)
			return
		}
		if request.Method == http.MethodGet && request.URL.Path == "/api/v2/organizations/org-1/memberships" {
			_, _ = writer.Write([]byte(`{"users":[]}`))
			return
		}
		if request.Method == http.MethodGet && request.URL.Path == "/api/v2/organizations/org-1/workspaces" {
			_, _ = writer.Write([]byte(`{"workspaces":[]}`))
			return
		}
		http.Error(writer, "unexpected request", http.StatusNotFound)
	}))
	defer server.Close()

	client, err := New(server.URL+"/api", "secret-token", time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	organization, err := client.FindOrganization(context.Background(), testOrganizationID, "")
	if err != nil || organization == nil || organization.ID != testOrganizationID {
		t.Fatalf("find organization from workspaces: %#v, %v", organization, err)
	}
}

func TestFindOrganizationFallsBackToVisibleProjects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.Method == http.MethodGet && request.URL.Path == testOrganizationByIDPath {
			http.Error(writer, "user JWT required", http.StatusForbidden)
			return
		}
		if request.Method == http.MethodGet && request.URL.Path == "/api/v2/organizations/org-1/memberships" {
			http.Error(writer, "organization membership endpoint unavailable", http.StatusNotFound)
			return
		}
		if request.Method == http.MethodGet && request.URL.Path == "/api/v2/organizations/org-1/workspaces" {
			http.Error(writer, "workspace endpoint unavailable", http.StatusNotFound)
			return
		}
		if request.Method == http.MethodGet && request.URL.Path == "/api/v1/projects" {
			_, _ = writer.Write([]byte(`{"projects":[{"id":"project-1","orgId":"org-1"}]}`))
			return
		}
		http.Error(writer, "unexpected request", http.StatusNotFound)
	}))
	defer server.Close()

	client, err := New(server.URL+"/api", "secret-token", time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	organization, err := client.FindOrganization(context.Background(), testOrganizationID, "")
	if err != nil || organization == nil || organization.ID != testOrganizationID {
		t.Fatalf("find organization from projects: %#v, %v", organization, err)
	}
}

func TestTokenIdentityIDReadsMachineIdentityClaim(t *testing.T) {
	claims := base64.RawURLEncoding.EncodeToString([]byte(`{"identityId":"identity-1"}`))
	client, err := New("https://infisical.example", "header."+claims+".signature", time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if got := client.TokenIdentityID(); got != "identity-1" {
		t.Fatalf("expected identity ID, got %q", got)
	}

	apiKeyClient, err := New("https://infisical.example", "api-key", time.Second)
	if err != nil {
		t.Fatalf("new API key client: %v", err)
	}
	if got := apiKeyClient.TokenIdentityID(); got != "" {
		t.Fatalf("expected no identity ID for API key, got %q", got)
	}
}

func TestEnsureIdentityProjectMembership(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.Method == http.MethodGet && request.URL.Path == "/api/v1/projects/project-1/memberships/identities/identity-1" {
			requests++
			http.Error(writer, "membership not found", http.StatusNotFound)
			return
		}
		if request.Method == http.MethodPost && request.URL.Path == "/api/v1/projects/project-1/memberships/identities/identity-1" {
			requests++
			var body IdentityMembershipRequest
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode membership request: %v", err)
			}
			if len(body.Roles) != 1 || body.Roles[0].Role != "admin" || body.Roles[0].IsTemporary {
				t.Errorf("unexpected membership request: %#v", body)
			}
			_, _ = writer.Write([]byte(`{"identityMembership":{"id":"membership-1","projectId":"project-1","identityId":"identity-1"}}`))
			return
		}
		http.Error(writer, "unexpected request", http.StatusNotFound)
	}))
	defer server.Close()

	client, err := New(server.URL+"/api", "secret-token", time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if err := client.EnsureIdentityProjectMembership(context.Background(), "project-1", "identity-1", []string{"admin"}); err != nil {
		t.Fatalf("ensure membership: %v", err)
	}
	if requests != 2 {
		t.Fatalf("expected membership read and create, got %d requests", requests)
	}
}

func TestOrganizationIdentityClientUsesOrganizationEndpoints(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/api/v1/identities":
			requests++
			var body CreateOrganizationIdentityRequest
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode organization identity request: %v", err)
			}
			if body.OrganizationID != testOrganizationID || body.Role != testOrganizationMemberRole {
				t.Errorf("unexpected organization identity request: %#v", body)
			}
			_, _ = writer.Write([]byte(`{"identity":{"id":"identity-1","name":"tenant","orgId":"org-1","role":"member"}}`))
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/identities":
			requests++
			if request.URL.Query().Get("orgId") != testOrganizationID {
				t.Errorf("unexpected organization query: %s", request.URL.RawQuery)
			}
			_, _ = writer.Write([]byte(`{"identities":[{"id":"membership-1","identityId":"identity-1","role":"no-access","orgId":"org-1","identity":{"id":"identity-1","name":"tenant","orgId":"org-1","hasDeleteProtection":false}}]}`))
		case request.Method == http.MethodGet && request.URL.Path == testOrganizationIdentityPath:
			requests++
			_, _ = writer.Write([]byte(`{"identity":{"id":"membership-1","identityId":"identity-1","orgId":"org-1","role":"no-access","identity":{"id":"identity-1","name":"tenant","orgId":"org-1","hasDeleteProtection":false}}}`))
		case request.Method == http.MethodPatch && request.URL.Path == testOrganizationIdentityPath:
			requests++
			_, _ = writer.Write([]byte(`{"identity":{"id":"membership-1","identityId":"identity-1","orgId":"org-1","role":"member","identity":{"id":"identity-1","name":"tenant-renamed","orgId":"org-1","hasDeleteProtection":false}}}`))
		case request.Method == http.MethodDelete && request.URL.Path == testOrganizationIdentityPath:
			requests++
		default:
			http.Error(writer, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := New(server.URL+"/api", "secret-token", time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	created, err := client.CreateOrganizationIdentity(context.Background(), testOrganizationID, CreateOrganizationIdentityRequest{Name: testTenantName, Role: testOrganizationMemberRole})
	if err != nil {
		t.Fatalf("create organization identity: %v", err)
	}
	if created.OrganizationID != testOrganizationID || created.OrganizationRole != testOrganizationMemberRole {
		t.Fatalf("unexpected created identity: %#v", created)
	}
	found, err := client.FindOrganizationIdentity(context.Background(), testOrganizationID, testTenantName)
	if err != nil {
		t.Fatalf("find organization identity: %v", err)
	}
	if found == nil || found.ID != "identity-1" || found.OrganizationRole != "no-access" {
		t.Fatalf("unexpected found identity: %#v", found)
	}
	observed, err := client.GetOrganizationIdentity(context.Background(), "identity-1")
	if err != nil {
		t.Fatalf("get organization identity: %v", err)
	}
	if observed.OrganizationID != testOrganizationID {
		t.Fatalf("unexpected observed identity: %#v", observed)
	}
	updated, err := client.UpdateOrganizationIdentity(context.Background(), "identity-1", IdentityPatch{Name: "tenant-renamed"})
	if err != nil {
		t.Fatalf("update organization identity: %v", err)
	}
	if updated.Name != "tenant-renamed" {
		t.Fatalf("unexpected updated identity: %#v", updated)
	}
	if err := client.DeleteOrganizationIdentity(context.Background(), "identity-1"); err != nil {
		t.Fatalf("delete organization identity: %v", err)
	}
	if requests != 5 {
		t.Fatalf("expected five organization identity requests, got %d", requests)
	}
}
