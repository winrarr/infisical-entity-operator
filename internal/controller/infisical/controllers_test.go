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

package infisical

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	infisicalv1alpha1 "github.com/winrarr/infisical-entity-operator/api/infisical/v1alpha1"
)

const (
	testNamespace = "default"
	testProject   = "demo"
	testProjectID = "project-1"
)

func testClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	testScheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(testScheme); err != nil {
		t.Fatal(err)
	}
	if err := infisicalv1alpha1.AddToScheme(testScheme); err != nil {
		t.Fatal(err)
	}
	return fake.NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(objects...).
		WithStatusSubresource(&infisicalv1alpha1.InfisicalConnection{}, &infisicalv1alpha1.InfisicalProject{}, &infisicalv1alpha1.InfisicalIdentity{}).
		Build()
}

func connectionAndSecret(serverURL string) (*infisicalv1alpha1.InfisicalConnection, *corev1.Secret) {
	return &infisicalv1alpha1.InfisicalConnection{
		ObjectMeta: metav1.ObjectMeta{Name: "infisical", Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalConnectionSpec{
			HostAPI: serverURL + "/api",
			AuthSecretRef: infisicalv1alpha1.SecretKeyReference{
				Name: "infisical-token",
				Key:  "token",
			},
		},
	}, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "infisical-token", Namespace: testNamespace},
		Data:       map[string][]byte{"token": []byte("test-token")},
	}
}

func TestConnectionReconcilerReportsReachability(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/projects" || request.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(writer, "unexpected request", http.StatusBadRequest)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"projects":[]}`))
	}))
	defer server.Close()

	connection, secret := connectionAndSecret(server.URL)
	kubeClient := testClient(t, connection, secret)
	reconciler := &InfisicalConnectionReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: connection.Name, Namespace: connection.Namespace}}); err != nil {
		t.Fatalf("reconcile connection: %v", err)
	}

	var observed infisicalv1alpha1.InfisicalConnection
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(connection), &observed); err != nil {
		t.Fatalf("get connection: %v", err)
	}
	if len(observed.Status.Conditions) != 1 || observed.Status.Conditions[0].Status != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True, got %#v", observed.Status.Conditions)
	}
}

func TestProjectReconcilerCreatesAndUpdatesProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/api/v1/projects":
			_, _ = writer.Write([]byte(`{"project":{"id":"project-1","name":"demo","slug":"demo-project","orgId":"org-1","environments":[]}}`))
		case request.Method == http.MethodPatch && request.URL.Path == "/api/v1/projects/project-1":
			_, _ = writer.Write([]byte(`{"project":{"id":"project-1","name":"demo","slug":"demo-project","orgId":"org-1","description":"updated","environments":[{"id":"env-1","name":"Production","slug":"prod"}]}}`))
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/projects/project-1":
			_, _ = writer.Write([]byte(`{"project":{"id":"project-1","name":"demo","slug":"demo-project","orgId":"org-1","environments":[{"id":"env-1","name":"Production","slug":"prod"}]}}`))
		default:
			http.Error(writer, fmt.Sprintf("unexpected %s %s", request.Method, request.URL.Path), http.StatusNotFound)
		}
	}))
	defer server.Close()

	connection, secret := connectionAndSecret(server.URL)
	project := &infisicalv1alpha1.InfisicalProject{
		ObjectMeta: metav1.ObjectMeta{Name: testProject, Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalProjectSpec{
			ConnectionRef: infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			ProjectName:   testProject,
		},
	}
	kubeClient := testClient(t, connection, secret, project)
	reconciler := &InfisicalProjectReconciler{Client: kubeClient}
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(project)}

	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile project finalizer: %v", err)
	}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile project: %v", err)
	}

	var observed infisicalv1alpha1.InfisicalProject
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(project), &observed); err != nil {
		t.Fatalf("get project: %v", err)
	}
	if observed.Status.ProjectID != testProjectID || observed.Status.OrganizationID != "org-1" {
		t.Fatalf("unexpected project status: %#v", observed.Status)
	}
	if len(observed.Status.Environments) != 1 || observed.Status.Environments[0].Slug != "prod" {
		t.Fatalf("unexpected environment status: %#v", observed.Status.Environments)
	}
	if len(observed.Status.Conditions) != 1 || observed.Status.Conditions[0].Status != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True, got %#v", observed.Status.Conditions)
	}
	if len(observed.Finalizers) != 0 {
		t.Fatalf("expected no finalizer for default orphan policy, got %#v", observed.Finalizers)
	}

	observed.Spec.Description = "updated"
	if err := kubeClient.Update(context.Background(), &observed); err != nil {
		t.Fatalf("update project spec: %v", err)
	}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile project drift: %v", err)
	}
}

func TestProjectReconcilerAddsDeleteFinalizer(t *testing.T) {
	connection := &infisicalv1alpha1.InfisicalConnection{
		ObjectMeta: metav1.ObjectMeta{Name: "infisical", Namespace: testNamespace},
	}
	project := &infisicalv1alpha1.InfisicalProject{
		ObjectMeta: metav1.ObjectMeta{Name: testProject, Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalProjectSpec{
			ConnectionRef:  infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			DeletionPolicy: infisicalv1alpha1.DeletionPolicyDelete,
		},
	}
	kubeClient := testClient(t, connection, project)
	reconciler := &InfisicalProjectReconciler{Client: kubeClient}
	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(project)}); err != nil {
		t.Fatalf("reconcile project finalizer: %v", err)
	}

	var observed infisicalv1alpha1.InfisicalProject
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(project), &observed); err != nil {
		t.Fatalf("get project: %v", err)
	}
	if len(observed.Finalizers) != 1 {
		t.Fatalf("expected delete finalizer, got %#v", observed.Finalizers)
	}
}

func TestIdentityReconcilerWaitsForProjectThenCreatesIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/api/v1/projects/project-1/identities":
			_, _ = writer.Write([]byte(`{"identity":{"id":"identity-1","name":"workload","projectId":"project-1"}}`))
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/projects/project-1/identities/identity-1":
			_, _ = writer.Write([]byte(`{"identity":{"id":"identity-1","name":"workload","projectId":"project-1"}}`))
		default:
			http.Error(writer, fmt.Sprintf("unexpected %s %s", request.Method, request.URL.Path), http.StatusNotFound)
		}
	}))
	defer server.Close()

	connection, secret := connectionAndSecret(server.URL)
	project := &infisicalv1alpha1.InfisicalProject{
		ObjectMeta: metav1.ObjectMeta{Name: testProject, Namespace: testNamespace},
		Status:     infisicalv1alpha1.InfisicalProjectStatus{ProjectID: testProjectID},
	}
	identity := &infisicalv1alpha1.InfisicalIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: "workload", Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalIdentitySpec{
			ConnectionRef: infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			ProjectRef:    infisicalv1alpha1.LocalObjectReference{Name: project.Name},
		},
	}
	kubeClient := testClient(t, connection, secret, project, identity)
	reconciler := &InfisicalIdentityReconciler{Client: kubeClient}
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(identity)}

	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile identity finalizer: %v", err)
	}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile identity: %v", err)
	}

	var observed infisicalv1alpha1.InfisicalIdentity
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(identity), &observed); err != nil {
		t.Fatalf("get identity: %v", err)
	}
	if observed.Status.IdentityID != "identity-1" || observed.Status.ProjectID != testProjectID {
		t.Fatalf("unexpected identity status: %#v", observed.Status)
	}
	if len(observed.Status.Conditions) != 1 || observed.Status.Conditions[0].Status != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True, got %#v", observed.Status.Conditions)
	}
	if len(observed.Finalizers) != 0 {
		t.Fatalf("expected no finalizer for default orphan policy, got %#v", observed.Finalizers)
	}
}
