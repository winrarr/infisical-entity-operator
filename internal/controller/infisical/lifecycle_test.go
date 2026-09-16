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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infisicalv1alpha1 "github.com/winrarr/infisical-entity-operator/api/infisical/v1alpha1"
)

func TestConnectionReconcilerRecoversAfterExternalFailure(t *testing.T) {
	healthy := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != testProjectsPath {
			http.Error(writer, fmt.Sprintf("unexpected %s %s", request.Method, request.URL.Path), http.StatusNotFound)
			return
		}
		if !healthy {
			http.Error(writer, "temporary outage", http.StatusServiceUnavailable)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"projects":[]}`))
	}))
	defer server.Close()

	connection, secret := connectionAndSecret(server.URL)
	kubeClient := testClient(t, connection, secret)
	reconciler := &InfisicalConnectionReconciler{Client: kubeClient}
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(connection)}

	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile unavailable connection: %v", err)
	}
	var observed infisicalv1alpha1.InfisicalConnection
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(connection), &observed); err != nil {
		t.Fatalf("get unavailable connection: %v", err)
	}
	if conditionReason(observed.Status.Conditions, readyCondition) != "ConnectionUnavailable" {
		t.Fatalf("expected external failure condition, got %#v", observed.Status.Conditions)
	}
	assertKstatusStates(t, observed.Status.Conditions, metav1.ConditionFalse, metav1.ConditionTrue, metav1.ConditionFalse)

	healthy = true
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile recovered connection: %v", err)
	}
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(connection), &observed); err != nil {
		t.Fatalf("get recovered connection: %v", err)
	}
	assertKstatusStates(t, observed.Status.Conditions, metav1.ConditionTrue, metav1.ConditionFalse, metav1.ConditionFalse)
}

func TestProjectReconcilerAdoptsBySlug(t *testing.T) {
	createRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/projects/slug/demo":
			_, _ = writer.Write([]byte(`{"id":"project-1","name":"demo","slug":"demo","environments":[]}`))
		case request.Method == http.MethodGet && request.URL.Path == testProjectByIDPath:
			_, _ = writer.Write([]byte(`{"project":{"id":"project-1","name":"demo","slug":"demo","environments":[]}}`))
		case request.Method == http.MethodPost && request.URL.Path == testProjectsPath:
			createRequests++
			http.Error(writer, "adoption unexpectedly created a project", http.StatusInternalServerError)
		default:
			http.Error(writer, fmt.Sprintf("unexpected %s %s", request.Method, request.URL.Path), http.StatusNotFound)
		}
	}))
	defer server.Close()

	connection, secret := connectionAndSecret(server.URL)
	project := &infisicalv1alpha1.InfisicalProject{
		ObjectMeta: metav1.ObjectMeta{Name: testProject, Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalProjectSpec{
			ConnectionRef:  infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			ProjectName:    testProject,
			Slug:           testProject,
			CreationPolicy: infisicalv1alpha1.CreationPolicyCreateOrAdopt,
		},
	}
	kubeClient := testClient(t, connection, secret, project)
	reconciler := &InfisicalProjectReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(project)}); err != nil {
		t.Fatalf("reconcile adopted project: %v", err)
	}
	var observed infisicalv1alpha1.InfisicalProject
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(project), &observed); err != nil {
		t.Fatalf("get adopted project: %v", err)
	}
	if observed.Status.ProjectID != testProjectID || conditionStatus(observed.Status.Conditions, readyCondition) != metav1.ConditionTrue {
		t.Fatalf("unexpected adopted project status: %#v", observed.Status)
	}
	if createRequests != 0 {
		t.Fatalf("adoption issued %d create requests", createRequests)
	}
}

func TestProjectReconcilerRecreatesRemoteProjectAfterDeletion(t *testing.T) {
	created := false
	createRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodGet && request.URL.Path == testProjectByIDPath:
			if created {
				http.Error(writer, "unexpected stale project lookup", http.StatusNotFound)
				return
			}
			http.Error(writer, "project deleted", http.StatusNotFound)
		case request.Method == http.MethodPost && request.URL.Path == testProjectsPath:
			createRequests++
			created = true
			_, _ = writer.Write([]byte(`{"project":{"id":"project-2","name":"demo","slug":"demo","environments":[]}}`))
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/projects/project-2":
			_, _ = writer.Write([]byte(`{"project":{"id":"project-2","name":"demo","slug":"demo","environments":[]}}`))
		default:
			http.Error(writer, fmt.Sprintf("unexpected %s %s", request.Method, request.URL.Path), http.StatusNotFound)
		}
	}))
	defer server.Close()

	connection, secret := connectionAndSecret(server.URL)
	project := &infisicalv1alpha1.InfisicalProject{
		ObjectMeta: metav1.ObjectMeta{Name: testProject, Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalProjectSpec{
			ConnectionRef:  infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			ProjectName:    testProject,
			CreationPolicy: infisicalv1alpha1.CreationPolicyCreate,
		},
		Status: infisicalv1alpha1.InfisicalProjectStatus{ProjectID: testProjectID},
	}
	kubeClient := testClient(t, connection, secret, project)
	reconciler := &InfisicalProjectReconciler{Client: kubeClient}
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(project)}

	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile deleted project: %v", err)
	}
	var observed infisicalv1alpha1.InfisicalProject
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(project), &observed); err != nil {
		t.Fatalf("get missing project status: %v", err)
	}
	if observed.Status.ProjectID != "" || conditionReason(observed.Status.Conditions, readyCondition) != "RemoteProjectMissing" {
		t.Fatalf("expected missing-project recovery state, got %#v", observed.Status)
	}
	assertKstatusStates(t, observed.Status.Conditions, metav1.ConditionFalse, metav1.ConditionTrue, metav1.ConditionFalse)

	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile recreated project: %v", err)
	}
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(project), &observed); err != nil {
		t.Fatalf("get recreated project: %v", err)
	}
	if observed.Status.ProjectID != "project-2" || conditionStatus(observed.Status.Conditions, readyCondition) != metav1.ConditionTrue {
		t.Fatalf("unexpected recreated project status: %#v", observed.Status)
	}
	if createRequests != 1 {
		t.Fatalf("expected one recreation request, got %d", createRequests)
	}
}

func TestProjectReconcilerDeletesRemoteProjectBeforeRemovingFinalizer(t *testing.T) {
	deleteRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodDelete || request.URL.Path != testProjectByIDPath {
			http.Error(writer, fmt.Sprintf("unexpected %s %s", request.Method, request.URL.Path), http.StatusNotFound)
			return
		}
		deleteRequests++
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	connection, secret := connectionAndSecret(server.URL)
	project := &infisicalv1alpha1.InfisicalProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:              testProject,
			Namespace:         testNamespace,
			Finalizers:        []string{finalizerName},
			DeletionTimestamp: &metav1.Time{Time: metav1.Now().Time},
		},
		Spec: infisicalv1alpha1.InfisicalProjectSpec{
			ConnectionRef:  infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			DeletionPolicy: infisicalv1alpha1.DeletionPolicyDelete,
		},
		Status: infisicalv1alpha1.InfisicalProjectStatus{ProjectID: testProjectID},
	}
	kubeClient := testClient(t, connection, secret, project)
	reconciler := &InfisicalProjectReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(project)}); err != nil {
		t.Fatalf("reconcile project deletion: %v", err)
	}
	if deleteRequests != 1 {
		t.Fatalf("expected one remote deletion request, got %d", deleteRequests)
	}

	var observed infisicalv1alpha1.InfisicalProject
	err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(project), &observed)
	if err == nil {
		if len(observed.Finalizers) != 0 {
			t.Fatalf("expected finalizer removal, got %#v", observed.Finalizers)
		}
	} else if !apierrors.IsNotFound(err) {
		t.Fatalf("get deleted project: %v", err)
	}
}

func TestEnvironmentReconcilerCorrectsMutableDrift(t *testing.T) {
	patchRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodGet && request.URL.Path == testEnvironmentByIDPath:
			_, _ = writer.Write([]byte(`{"environment":{"id":"environment-1","name":"Old","slug":"qa","position":1,"projectId":"project-1"}}`))
		case request.Method == http.MethodPatch && request.URL.Path == testEnvironmentByIDPath:
			patchRequests++
			_, _ = writer.Write([]byte(`{"environment":{"id":"environment-1","name":"QA","slug":"qa","position":4,"projectId":"project-1"}}`))
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
	position := int32(4)
	environment := &infisicalv1alpha1.InfisicalEnvironment{
		ObjectMeta: metav1.ObjectMeta{Name: "qa", Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalEnvironmentSpec{
			ConnectionRef:   infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			ProjectRef:      infisicalv1alpha1.LocalObjectReference{Name: project.Name},
			EnvironmentName: "QA",
			Slug:            "qa",
			Position:        &position,
		},
		Status: infisicalv1alpha1.InfisicalEnvironmentStatus{
			EnvironmentID: testEnvironmentID,
			ProjectID:     testProjectID,
		},
	}
	kubeClient := testClient(t, connection, secret, project, environment)
	reconciler := &InfisicalEnvironmentReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(environment)}); err != nil {
		t.Fatalf("reconcile environment drift: %v", err)
	}
	if patchRequests != 1 {
		t.Fatalf("expected one environment drift patch, got %d", patchRequests)
	}

	var observed infisicalv1alpha1.InfisicalEnvironment
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(environment), &observed); err != nil {
		t.Fatalf("get reconciled environment: %v", err)
	}
	if observed.Status.Name != "QA" || observed.Status.Position != 4 || conditionStatus(observed.Status.Conditions, readyCondition) != metav1.ConditionTrue {
		t.Fatalf("unexpected reconciled environment status: %#v", observed.Status)
	}
}
