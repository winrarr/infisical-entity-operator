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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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
