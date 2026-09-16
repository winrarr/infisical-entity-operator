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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
	testNamespace                    = "default"
	testProject                      = "demo"
	testProjectID                    = "project-1"
	testConnection                   = "infisical"
	testTokenSecret                  = "infisical-token"
	testTokenKey                     = "token"
	testWorkload                     = "workload"
	testIdentityID                   = "identity-1"
	testOrganizationID               = "org-1"
	testSecondProjectID              = "project-2"
	testTenantSlug                   = "tenant"
	testUniversalAuthIdentityPath    = "/api/v1/auth/universal-auth/identities/identity-1"
	testUniversalAuthSecret          = "universal-auth"
	testUniversalAuthOutputSecret    = "tenant-credentials"
	testUniversalAuthClientSecretID  = "secret-1"
	testUniversalAuthClientSecret    = "one-time-secret"
	testClusterAuthName              = "cluster-auth"
	testKubernetesHost               = "https://kubernetes.default.svc"
	testKubernetesCAKey              = "ca.crt"
	testKubernetesCASecret           = "kubernetes-ca"
	testKubernetesReviewerSecret     = "kubernetes-reviewer"
	testIdentityPath                 = "/api/v1/projects/project-1/identities"
	testIdentityByIDPath             = "/api/v1/projects/project-1/identities/identity-1"
	testMembershipPath               = "/api/v1/projects/project-1/memberships/identities/identity-1"
	testOrganizationIdentityPath     = "/api/v1/identities"
	testOrganizationIdentityByIDPath = "/api/v1/identities/identity-1"
	testSecondMembershipPath         = "/api/v1/projects/project-2/memberships/identities/identity-1"
	testMemberRole                   = "member"
	testProjectsPath                 = "/api/v1/projects"
	testProjectByIDPath              = "/api/v1/projects/project-1"
	testEnvironmentID                = "environment-1"
	testEnvironmentByIDPath          = "/api/v1/projects/project-1/environments/environment-1"
	testTenantOrganizationName       = "tenant-org"
	testConfigurationInvalidReason   = "ConfigurationInvalid"
)

type infisicalMembershipRequest struct {
	Roles []struct {
		Role        string `json:"role"`
		IsTemporary bool   `json:"isTemporary"`
	} `json:"roles"`
}

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
		WithStatusSubresource(
			&infisicalv1alpha1.InfisicalConnection{},
			&infisicalv1alpha1.InfisicalOrganization{},
			&infisicalv1alpha1.InfisicalProject{},
			&infisicalv1alpha1.InfisicalIdentity{},
			&infisicalv1alpha1.InfisicalEnvironment{},
			&infisicalv1alpha1.InfisicalKubernetesAuth{},
			&infisicalv1alpha1.InfisicalUniversalAuth{},
		).
		Build()
}

func conditionStatus(conditions []metav1.Condition, conditionType string) metav1.ConditionStatus {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status
		}
	}
	return metav1.ConditionUnknown
}

func conditionReason(conditions []metav1.Condition, conditionType string) string {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Reason
		}
	}
	return ""
}

func conditionMessage(conditions []metav1.Condition, conditionType string) string {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Message
		}
	}
	return ""
}

func assertKstatusStates(t *testing.T, conditions []metav1.Condition, ready, reconciling, stalled metav1.ConditionStatus) {
	t.Helper()
	expected := map[string]metav1.ConditionStatus{
		readyCondition:       ready,
		reconcilingCondition: reconciling,
		stalledCondition:     stalled,
	}
	for conditionType, expectedStatus := range expected {
		if actual := conditionStatus(conditions, conditionType); actual != expectedStatus {
			t.Fatalf("expected %s=%s, got %#v", conditionType, expectedStatus, conditions)
		}
	}
}

func TestSetConditionReportsKstatusLifecycle(t *testing.T) {
	var conditions []metav1.Condition

	setCondition(&conditions, 3, metav1.ConditionFalse, "ProjectNotReady", "project is not ready")
	if len(conditions) != 3 {
		t.Fatalf("expected Ready, Reconciling, and Stalled conditions, got %#v", conditions)
	}
	assertKstatusStates(t, conditions, metav1.ConditionFalse, metav1.ConditionTrue, metav1.ConditionFalse)
	if conditionReason(conditions, reconcilingCondition) != "Progressing" {
		t.Fatalf("expected progressing reason, got %#v", conditions)
	}

	setCondition(&conditions, 3, metav1.ConditionTrue, "Ready", "reconciliation completed")
	assertKstatusStates(t, conditions, metav1.ConditionTrue, metav1.ConditionFalse, metav1.ConditionFalse)
	if conditionReason(conditions, stalledCondition) != "NotStalled" {
		t.Fatalf("expected not-stalled reason, got %#v", conditions)
	}

	setCondition(&conditions, 4, metav1.ConditionFalse, testConfigurationInvalidReason, "configuration is invalid")
	assertKstatusStates(t, conditions, metav1.ConditionFalse, metav1.ConditionFalse, metav1.ConditionTrue)
	if conditionReason(conditions, stalledCondition) != testConfigurationInvalidReason {
		t.Fatalf("expected stalled reason to explain the invalid configuration, got %#v", conditions)
	}

	for _, condition := range conditions {
		if condition.ObservedGeneration != 4 {
			t.Fatalf("expected all conditions to observe generation 4, got %#v", conditions)
		}
	}
}

func identityProjectRef(name string) *infisicalv1alpha1.LocalObjectReference {
	return &infisicalv1alpha1.LocalObjectReference{Name: name}
}

func connectionAndSecret(serverURL string) (*infisicalv1alpha1.InfisicalConnection, *corev1.Secret) {
	return &infisicalv1alpha1.InfisicalConnection{
		ObjectMeta: metav1.ObjectMeta{Name: testConnection, Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalConnectionSpec{
			HostAPI: serverURL + "/api",
			AuthSecretRef: &infisicalv1alpha1.SecretKeyReference{
				Name: testTokenSecret,
				Key:  testTokenKey,
			},
		},
	}, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testTokenSecret, Namespace: testNamespace},
		Data:       map[string][]byte{testTokenKey: []byte("test-token")},
	}
}

func TestConnectionReconcilerReportsReachability(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != testProjectsPath || request.Header.Get("Authorization") != "Bearer test-token" {
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
	if conditionStatus(observed.Status.Conditions, readyCondition) != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True, got %#v", observed.Status.Conditions)
	}
	assertKstatusStates(t, observed.Status.Conditions, metav1.ConditionTrue, metav1.ConditionFalse, metav1.ConditionFalse)
}

func TestConnectionReconcilerRejectsInvalidSpecBeforeExternalCalls(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*infisicalv1alpha1.InfisicalConnectionSpec)
		message string
	}{
		{
			name:    "missing authentication",
			mutate:  func(spec *infisicalv1alpha1.InfisicalConnectionSpec) { spec.AuthSecretRef = nil },
			message: "exactly one of authSecretRef or universalAuth must be configured",
		},
		{
			name: "multiple authentication methods",
			mutate: func(spec *infisicalv1alpha1.InfisicalConnectionSpec) {
				spec.UniversalAuth = &infisicalv1alpha1.UniversalAuthConnectionSpec{
					SecretRef: infisicalv1alpha1.UniversalAuthSecretReference{Name: testUniversalAuthSecret},
				}
			},
			message: "exactly one of authSecretRef or universalAuth must be configured",
		},
		{
			name:    "invalid URL scheme",
			mutate:  func(spec *infisicalv1alpha1.InfisicalConnectionSpec) { spec.HostAPI = "ftp://infisical.example/api" },
			message: "infisical API URL must use http or https",
		},
		{
			name: "URL query",
			mutate: func(spec *infisicalv1alpha1.InfisicalConnectionSpec) {
				spec.HostAPI = "https://infisical.example/api?tenant=one"
			},
			message: "infisical API URL must not contain a query or fragment",
		},
		{
			name: "non-positive timeout",
			mutate: func(spec *infisicalv1alpha1.InfisicalConnectionSpec) {
				spec.RequestTimeout = &metav1.Duration{}
			},
			message: "requestTimeout must be greater than zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := infisicalv1alpha1.InfisicalConnectionSpec{
				HostAPI: "https://infisical.example/api",
				AuthSecretRef: &infisicalv1alpha1.SecretKeyReference{
					Name: testTokenSecret,
				},
			}
			tt.mutate(&spec)
			connection := &infisicalv1alpha1.InfisicalConnection{
				ObjectMeta: metav1.ObjectMeta{Name: tt.name, Namespace: testNamespace},
				Spec:       spec,
			}
			kubeClient := testClient(t, connection)
			reconciler := &InfisicalConnectionReconciler{Client: kubeClient}
			if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(connection)}); err != nil {
				t.Fatalf("reconcile invalid connection: %v", err)
			}

			var observed infisicalv1alpha1.InfisicalConnection
			if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(connection), &observed); err != nil {
				t.Fatalf("get connection: %v", err)
			}
			if conditionReason(observed.Status.Conditions, readyCondition) != "ConfigurationInvalid" || !strings.Contains(conditionMessage(observed.Status.Conditions, readyCondition), tt.message) {
				t.Fatalf("expected configuration error %q, got %#v", tt.message, observed.Status.Conditions)
			}
			assertKstatusStates(t, observed.Status.Conditions, metav1.ConditionFalse, metav1.ConditionFalse, metav1.ConditionTrue)
		})
	}
}

func TestConnectionReconcilerUsesUniversalAuthSecret(t *testing.T) {
	claims := base64.RawURLEncoding.EncodeToString([]byte(`{"identityId":"identity-1"}`))
	accessToken := "header." + claims + ".signature"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/api/v1/auth/universal-auth/login":
			_, _ = writer.Write([]byte(`{"accessToken":"` + accessToken + `","expiresIn":3600,"accessTokenMaxTTL":3600,"tokenType":"Bearer"}`))
		case request.Method == http.MethodGet && request.URL.Path == testProjectsPath:
			if request.Header.Get("Authorization") != "Bearer "+accessToken {
				t.Errorf("unexpected Universal Auth bearer token: %s", request.Header.Get("Authorization"))
			}
			_, _ = writer.Write([]byte(`{"projects":[]}`))
		default:
			http.Error(writer, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	connection := &infisicalv1alpha1.InfisicalConnection{
		ObjectMeta: metav1.ObjectMeta{Name: testConnection, Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalConnectionSpec{
			HostAPI: server.URL + "/api",
			UniversalAuth: &infisicalv1alpha1.UniversalAuthConnectionSpec{
				SecretRef:        infisicalv1alpha1.UniversalAuthSecretReference{Name: testUniversalAuthSecret},
				OrganizationSlug: testTenantSlug,
			},
		},
	}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: testUniversalAuthSecret, Namespace: testNamespace}, Data: map[string][]byte{"clientId": []byte("client-1"), "clientSecret": []byte("client-secret-1")}}
	kubeClient := testClient(t, connection, secret)
	reconciler := &InfisicalConnectionReconciler{Client: kubeClient}
	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(connection)}); err != nil {
		t.Fatalf("reconcile Universal Auth connection: %v", err)
	}
	var observed infisicalv1alpha1.InfisicalConnection
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(connection), &observed); err != nil {
		t.Fatalf("get Universal Auth connection: %v", err)
	}
	if !conditionReady(observed.Status.Conditions) {
		t.Fatalf("expected Universal Auth connection to be ready: %#v", observed.Status)
	}
	assertKstatusStates(t, observed.Status.Conditions, metav1.ConditionTrue, metav1.ConditionFalse, metav1.ConditionFalse)
}

func TestUniversalAuthReconcilerPublishesClientSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == testUniversalAuthIdentityPath:
			_, _ = writer.Write([]byte(`{"identityUniversalAuth":{"id":"ua-1","clientId":"client-1","identityId":"identity-1","accessTokenTTL":7200,"accessTokenMaxTTL":7200,"accessTokenNumUsesLimit":0,"accessTokenPeriod":0,"lockoutEnabled":true,"lockoutThreshold":3,"lockoutDurationSeconds":300,"lockoutCounterResetSeconds":30}}`))
		case request.Method == http.MethodGet && request.URL.Path == testUniversalAuthIdentityPath:
			_, _ = writer.Write([]byte(`{"identityUniversalAuth":{"id":"ua-1","clientId":"client-1","identityId":"identity-1","accessTokenTTL":7200,"accessTokenMaxTTL":7200,"accessTokenNumUsesLimit":0,"accessTokenPeriod":0,"lockoutEnabled":true,"lockoutThreshold":3,"lockoutDurationSeconds":300,"lockoutCounterResetSeconds":30}}`))
		case request.Method == http.MethodPost && request.URL.Path == testUniversalAuthIdentityPath+"/client-secrets":
			_, _ = writer.Write([]byte(`{"clientSecret":"one-time-secret","clientSecretData":{"id":"secret-1","description":"operator","clientSecretPrefix":"uats_","clientSecretNumUsesLimit":0,"clientSecretTTL":0,"identityUAId":"ua-1","isClientSecretRevoked":false}}`))
		case request.Method == http.MethodGet && request.URL.Path == testUniversalAuthIdentityPath+"/client-secrets/"+testUniversalAuthClientSecretID:
			_, _ = writer.Write([]byte(`{"clientSecretData":{"id":"secret-1","description":"operator","clientSecretPrefix":"uats_","clientSecretNumUsesLimit":0,"clientSecretTTL":0,"identityUAId":"ua-1","isClientSecretRevoked":false}}`))
		default:
			http.Error(writer, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	connection, secret := connectionAndSecret(server.URL)
	identity := &infisicalv1alpha1.InfisicalIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant-identity", Namespace: testNamespace},
		Status: infisicalv1alpha1.InfisicalIdentityStatus{
			IdentityID: "identity-1",
			Conditions: []metav1.Condition{{Type: readyCondition, Status: metav1.ConditionTrue}},
		},
	}
	auth := &infisicalv1alpha1.InfisicalUniversalAuth{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant-universal-auth", Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalUniversalAuthSpec{
			ConnectionRef: infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			IdentityRef:   infisicalv1alpha1.LocalObjectReference{Name: identity.Name},
			ClientSecret: infisicalv1alpha1.UniversalAuthClientSecretSpec{
				SecretRef:   infisicalv1alpha1.UniversalAuthSecretReference{Name: testUniversalAuthOutputSecret},
				Description: "operator",
			},
		},
	}
	kubeClient := testClient(t, connection, secret, identity, auth)
	reconciler := &InfisicalUniversalAuthReconciler{Client: kubeClient}
	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(auth)}); err != nil {
		t.Fatalf("reconcile Universal Auth: %v", err)
	}

	var observed infisicalv1alpha1.InfisicalUniversalAuth
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(auth), &observed); err != nil {
		t.Fatalf("get Universal Auth: %v", err)
	}
	if observed.Status.AuthID != "ua-1" || observed.Status.ClientID != "client-1" || observed.Status.ClientSecret.ClientSecretID != testUniversalAuthClientSecretID || !conditionReady(observed.Status.Conditions) {
		t.Fatalf("unexpected Universal Auth status: %#v", observed.Status)
	}
	assertKstatusStates(t, observed.Status.Conditions, metav1.ConditionTrue, metav1.ConditionFalse, metav1.ConditionFalse)
	var published corev1.Secret
	if err := kubeClient.Get(context.Background(), client.ObjectKey{Namespace: testNamespace, Name: testUniversalAuthOutputSecret}, &published); err != nil {
		t.Fatalf("get published client Secret: %v", err)
	}
	if string(published.Data["clientId"]) != "client-1" || string(published.Data["clientSecret"]) != "one-time-secret" {
		t.Fatalf("unexpected published client Secret data")
	}
	if observed.Status.ClientSecret.ClientSecretID == "one-time-secret" {
		t.Fatal("client secret value was exposed in status")
	}

	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(auth)}); err != nil {
		t.Fatalf("reconcile stable Universal Auth: %v", err)
	}
}

func TestUniversalAuthReconcilerRotatesClientSecretAfterNonceChange(t *testing.T) {
	currentSecretID := testUniversalAuthClientSecretID
	secretNumber := 0
	revoked := ""
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == testUniversalAuthIdentityPath:
			_, _ = writer.Write([]byte(`{"identityUniversalAuth":{"id":"ua-1","clientId":"client-1","identityId":"identity-1"}}`))
		case request.Method == http.MethodGet && request.URL.Path == testUniversalAuthIdentityPath:
			_, _ = writer.Write([]byte(`{"identityUniversalAuth":{"id":"ua-1","clientId":"client-1","identityId":"identity-1"}}`))
		case request.Method == http.MethodPost && request.URL.Path == testUniversalAuthIdentityPath+"/client-secrets":
			secretNumber++
			currentSecretID = fmt.Sprintf("secret-%d", secretNumber)
			_, _ = fmt.Fprintf(writer, `{"clientSecret":"value-%d","clientSecretData":{"id":"%s","identityUAId":"ua-1","clientSecretPrefix":"uats_","isClientSecretRevoked":false}}`, secretNumber, currentSecretID)
		case request.Method == http.MethodGet && request.URL.Path == testUniversalAuthIdentityPath+"/client-secrets/"+currentSecretID:
			_, _ = fmt.Fprintf(writer, `{"clientSecretData":{"id":"%s","identityUAId":"ua-1","clientSecretPrefix":"uats_","isClientSecretRevoked":false}}`, currentSecretID)
		case request.Method == http.MethodPost && request.URL.Path == testUniversalAuthIdentityPath+"/client-secrets/"+testUniversalAuthClientSecretID+"/revoke":
			revoked = testUniversalAuthClientSecretID
		default:
			http.Error(writer, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	connection, secret := connectionAndSecret(server.URL)
	identity := &infisicalv1alpha1.InfisicalIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant-identity", Namespace: testNamespace},
		Status:     infisicalv1alpha1.InfisicalIdentityStatus{IdentityID: testIdentityID, Conditions: []metav1.Condition{{Type: readyCondition, Status: metav1.ConditionTrue}}},
	}
	auth := &infisicalv1alpha1.InfisicalUniversalAuth{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant-universal-auth", Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalUniversalAuthSpec{
			ConnectionRef: infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			IdentityRef:   infisicalv1alpha1.LocalObjectReference{Name: identity.Name},
			ClientSecret: infisicalv1alpha1.UniversalAuthClientSecretSpec{
				SecretRef:     infisicalv1alpha1.UniversalAuthSecretReference{Name: testUniversalAuthOutputSecret},
				RotationNonce: "one",
			},
		},
	}
	kubeClient := testClient(t, connection, secret, identity, auth)
	reconciler := &InfisicalUniversalAuthReconciler{Client: kubeClient}
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(auth)}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("initial reconcile Universal Auth: %v", err)
	}
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(auth), auth); err != nil {
		t.Fatalf("get initial Universal Auth: %v", err)
	}
	auth.Spec.ClientSecret.RotationNonce = "two"
	if err := kubeClient.Update(context.Background(), auth); err != nil {
		t.Fatalf("update rotation nonce: %v", err)
	}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("rotating reconcile Universal Auth: %v", err)
	}
	var observed infisicalv1alpha1.InfisicalUniversalAuth
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(auth), &observed); err != nil {
		t.Fatalf("get rotated Universal Auth: %v", err)
	}
	if observed.Status.ClientSecret.ClientSecretID != "secret-2" || observed.Status.ClientSecret.LastRotationNonce != "two" || revoked != testUniversalAuthClientSecretID {
		t.Fatalf("rotation did not replace and revoke the expected credentials: status=%#v revoked=%q", observed.Status, revoked)
	}
	var published corev1.Secret
	if err := kubeClient.Get(context.Background(), client.ObjectKey{Namespace: testNamespace, Name: testUniversalAuthOutputSecret}, &published); err != nil {
		t.Fatalf("get rotated Secret: %v", err)
	}
	if string(published.Data["clientSecret"]) != "value-2" {
		t.Fatalf("published Secret was not replaced")
	}
}

func TestOrganizationReconcilerCreatesOrganization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/api/v2/organizations":
			_, _ = writer.Write([]byte(`{"organization":{"id":"org-1","name":"tenant-org","slug":"tenant-org"}}`))
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/organization/org-1":
			_, _ = writer.Write([]byte(`{"organization":{"id":"org-1","name":"tenant-org","slug":"tenant-org"}}`))
		default:
			http.Error(writer, fmt.Sprintf("unexpected %s %s", request.Method, request.URL.Path), http.StatusNotFound)
		}
	}))
	defer server.Close()

	connection, secret := connectionAndSecret(server.URL)
	organization := &infisicalv1alpha1.InfisicalOrganization{
		ObjectMeta: metav1.ObjectMeta{Name: testTenantOrganizationName, Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalOrganizationSpec{
			ConnectionRef:    infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			OrganizationName: testTenantOrganizationName,
		},
	}
	kubeClient := testClient(t, connection, secret, organization)
	reconciler := &InfisicalOrganizationReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(organization)}); err != nil {
		t.Fatalf("reconcile organization: %v", err)
	}

	var observed infisicalv1alpha1.InfisicalOrganization
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(organization), &observed); err != nil {
		t.Fatalf("get organization: %v", err)
	}
	if observed.Status.OrganizationID != testOrganizationID || observed.Status.OrganizationName != testTenantOrganizationName || observed.Status.Slug != testTenantOrganizationName {
		t.Fatalf("unexpected organization status: %#v", observed.Status)
	}
	if conditionStatus(observed.Status.Conditions, readyCondition) != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True, got %#v", observed.Status.Conditions)
	}
	assertKstatusStates(t, observed.Status.Conditions, metav1.ConditionTrue, metav1.ConditionFalse, metav1.ConditionFalse)
}

func TestOrganizationReconcilerAdoptsWithMachineIdentityProjectVisibility(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/organization/org-1":
			http.Error(writer, "user JWT required", http.StatusForbidden)
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/projects":
			_, _ = writer.Write([]byte(`{"projects":[{"id":"project-1","name":"anchor","orgId":"org-1"}]}`))
		default:
			http.Error(writer, fmt.Sprintf("unexpected %s %s", request.Method, request.URL.Path), http.StatusNotFound)
		}
	}))
	defer server.Close()

	connection, secret := connectionAndSecret(server.URL)
	organization := &infisicalv1alpha1.InfisicalOrganization{
		ObjectMeta: metav1.ObjectMeta{Name: testTenantOrganizationName, Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalOrganizationSpec{
			ConnectionRef:  infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			OrganizationID: testOrganizationID,
			CreationPolicy: infisicalv1alpha1.CreationPolicyAdopt,
		},
	}
	kubeClient := testClient(t, connection, secret, organization)
	reconciler := &InfisicalOrganizationReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(organization)}); err != nil {
		t.Fatalf("reconcile adopted organization: %v", err)
	}

	var observed infisicalv1alpha1.InfisicalOrganization
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(organization), &observed); err != nil {
		t.Fatalf("get organization: %v", err)
	}
	if observed.Status.OrganizationID != testOrganizationID || conditionStatus(observed.Status.Conditions, readyCondition) != metav1.ConditionTrue {
		t.Fatalf("unexpected adopted organization status: %#v", observed.Status)
	}
	assertKstatusStates(t, observed.Status.Conditions, metav1.ConditionTrue, metav1.ConditionFalse, metav1.ConditionFalse)
}

func TestProjectReconcilerCreatesAndUpdatesProject(t *testing.T) {
	patchRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == testProjectsPath:
			_, _ = writer.Write([]byte(`{"project":{"id":"project-1","name":"demo","slug":"demo-project","orgId":"org-1","environments":[]}}`))
		case request.Method == http.MethodPatch && request.URL.Path == testProjectByIDPath:
			patchRequests++
			_, _ = writer.Write([]byte(`{"project":{"id":"project-1","name":"demo","slug":"demo-project","orgId":"org-1","description":"updated","environments":[{"id":"env-1","name":"Production","slug":"prod"}]}}`))
		case request.Method == http.MethodGet && request.URL.Path == testProjectByIDPath:
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
	if conditionStatus(observed.Status.Conditions, readyCondition) != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True, got %#v", observed.Status.Conditions)
	}
	assertKstatusStates(t, observed.Status.Conditions, metav1.ConditionTrue, metav1.ConditionFalse, metav1.ConditionFalse)
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
	if patchRequests != 1 {
		t.Fatalf("expected one project drift patch, got %d", patchRequests)
	}
}

func TestProjectReconcilerAddsDeleteFinalizer(t *testing.T) {
	connection := &infisicalv1alpha1.InfisicalConnection{
		ObjectMeta: metav1.ObjectMeta{Name: testConnection, Namespace: testNamespace},
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

func TestProjectReconcilerGrantsCreatorAccessForOrganizationProject(t *testing.T) {
	claims := base64.RawURLEncoding.EncodeToString([]byte(`{"identityId":"identity-1"}`))
	token := "header." + claims + ".signature"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == testProjectsPath:
			_, _ = writer.Write([]byte(`{"project":{"id":"project-1","name":"demo","slug":"demo","orgId":"org-1"}}`))
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/projects/project-1/memberships/identities/identity-1":
			http.Error(writer, "membership not found", http.StatusNotFound)
		case request.Method == http.MethodPost && request.URL.Path == "/api/v1/projects/project-1/memberships/identities/identity-1":
			var body infisicalMembershipRequest
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode membership request: %v", err)
			}
			if len(body.Roles) != 1 || body.Roles[0].Role != "admin" || body.Roles[0].IsTemporary {
				t.Errorf("unexpected membership request: %#v", body)
			}
			_, _ = writer.Write([]byte(`{"identityMembership":{"id":"membership-1","projectId":"project-1","identityId":"identity-1"}}`))
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/projects/project-1":
			_, _ = writer.Write([]byte(`{"project":{"id":"project-1","name":"demo","slug":"demo","orgId":"org-1","environments":[]}}`))
		default:
			http.Error(writer, fmt.Sprintf("unexpected %s %s", request.Method, request.URL.Path), http.StatusNotFound)
		}
	}))
	defer server.Close()

	connection, secret := connectionAndSecret(server.URL)
	secret.Data[testTokenKey] = []byte(token)
	organization := &infisicalv1alpha1.InfisicalOrganization{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant-boundary", Namespace: testNamespace},
		Status:     infisicalv1alpha1.InfisicalOrganizationStatus{OrganizationID: testOrganizationID},
	}
	project := &infisicalv1alpha1.InfisicalProject{
		ObjectMeta: metav1.ObjectMeta{Name: testProject, Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalProjectSpec{
			ConnectionRef:   infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			OrganizationRef: &infisicalv1alpha1.LocalObjectReference{Name: organization.Name},
			ProjectName:     testProject,
		},
	}
	kubeClient := testClient(t, connection, secret, organization, project)
	reconciler := &InfisicalProjectReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(project)}); err != nil {
		t.Fatalf("reconcile organization project: %v", err)
	}

	var observed infisicalv1alpha1.InfisicalProject
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(project), &observed); err != nil {
		t.Fatalf("get project: %v", err)
	}
	if observed.Status.ProjectID != testProjectID || observed.Status.OrganizationID != testOrganizationID {
		t.Fatalf("unexpected organization project status: %#v", observed.Status)
	}
	if conditionStatus(observed.Status.Conditions, readyCondition) != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True, got %#v", observed.Status.Conditions)
	}
	assertKstatusStates(t, observed.Status.Conditions, metav1.ConditionTrue, metav1.ConditionFalse, metav1.ConditionFalse)
}

func TestIdentityReconcilerWaitsForProjectThenCreatesIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == testIdentityPath:
			_, _ = writer.Write([]byte(`{"identity":{"id":"identity-1","name":"workload","projectId":"project-1"}}`))
		case request.Method == http.MethodGet && request.URL.Path == testIdentityByIDPath:
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
		ObjectMeta: metav1.ObjectMeta{Name: testWorkload, Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalIdentitySpec{
			ConnectionRef: infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			ProjectRef:    identityProjectRef(project.Name),
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
	if observed.Status.IdentityID != testIdentityID || observed.Status.ProjectID != testProjectID {
		t.Fatalf("unexpected identity status: %#v", observed.Status)
	}
	if conditionStatus(observed.Status.Conditions, readyCondition) != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True, got %#v", observed.Status.Conditions)
	}
	assertKstatusStates(t, observed.Status.Conditions, metav1.ConditionTrue, metav1.ConditionFalse, metav1.ConditionFalse)
	if len(observed.Finalizers) != 0 {
		t.Fatalf("expected no finalizer for default orphan policy, got %#v", observed.Finalizers)
	}
}

func TestIdentityReconcilerCreatesOrganizationIdentityAndProjectMembership(t *testing.T) {
	membershipReads := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == testOrganizationIdentityPath:
			var body struct {
				Name           string `json:"name"`
				OrganizationID string `json:"organizationId"`
				Role           string `json:"role"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode organization identity request: %v", err)
			}
			if body.Name != testWorkload || body.OrganizationID != testOrganizationID || body.Role != "no-access" {
				t.Errorf("unexpected organization identity request: %#v", body)
			}
			_, _ = writer.Write([]byte(`{"identity":{"id":"identity-1","name":"workload","orgId":"org-1","role":"no-access"}}`))
		case request.Method == http.MethodGet && request.URL.Path == testOrganizationIdentityByIDPath:
			_, _ = writer.Write([]byte(`{"identity":{"id":"membership-1","identityId":"identity-1","orgId":"org-1","role":"no-access","identity":{"id":"identity-1","name":"workload","orgId":"org-1","hasDeleteProtection":false}}}`))
		case request.Method == http.MethodGet && request.URL.Path == testSecondMembershipPath:
			membershipReads++
			if membershipReads == 1 {
				http.Error(writer, "membership not found", http.StatusNotFound)
				return
			}
			_, _ = writer.Write([]byte(`{"identityMembership":{"id":"membership-2","projectId":"project-2","identityId":"identity-1","roles":[{"id":"assignment-1","role":"member","isTemporary":false}]}}`))
		case request.Method == http.MethodPost && request.URL.Path == testSecondMembershipPath:
			var body infisicalMembershipRequest
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode organization membership request: %v", err)
			}
			if len(body.Roles) != 1 || body.Roles[0].Role != testMemberRole || body.Roles[0].IsTemporary {
				t.Errorf("unexpected organization membership request: %#v", body)
			}
			_, _ = writer.Write([]byte(`{"identityMembership":{"id":"membership-2","projectId":"project-2","identityId":"identity-1"}}`))
		default:
			http.Error(writer, fmt.Sprintf("unexpected %s %s", request.Method, request.URL.Path), http.StatusNotFound)
		}
	}))
	defer server.Close()

	connection, secret := connectionAndSecret(server.URL)
	organization := &infisicalv1alpha1.InfisicalOrganization{
		ObjectMeta: metav1.ObjectMeta{Name: testTenantOrganizationName, Namespace: testNamespace},
		Status:     infisicalv1alpha1.InfisicalOrganizationStatus{OrganizationID: testOrganizationID},
	}
	boundProject := &infisicalv1alpha1.InfisicalProject{
		ObjectMeta: metav1.ObjectMeta{Name: "secondary", Namespace: testNamespace},
		Status:     infisicalv1alpha1.InfisicalProjectStatus{ProjectID: testSecondProjectID, OrganizationID: testOrganizationID},
	}
	identity := &infisicalv1alpha1.InfisicalIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: testWorkload, Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalIdentitySpec{
			ConnectionRef: infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			Scope:         infisicalv1alpha1.IdentityScopeOrganization,
			OrganizationRef: &infisicalv1alpha1.LocalObjectReference{
				Name: organization.Name,
			},
			ProjectRoleBindings: []infisicalv1alpha1.IdentityProjectRoleBinding{{
				ProjectRef: infisicalv1alpha1.LocalObjectReference{Name: boundProject.Name},
				RoleSlugs:  []string{testMemberRole},
			}},
		},
	}
	kubeClient := testClient(t, connection, secret, organization, boundProject, identity)
	reconciler := &InfisicalIdentityReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(identity)}); err != nil {
		t.Fatalf("reconcile organization identity: %v", err)
	}

	var observed infisicalv1alpha1.InfisicalIdentity
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(identity), &observed); err != nil {
		t.Fatalf("get organization identity: %v", err)
	}
	if observed.Status.IdentityID != testIdentityID || observed.Status.OrganizationID != testOrganizationID || observed.Status.OrganizationRole != "no-access" {
		t.Fatalf("unexpected organization identity status: %#v", observed.Status)
	}
	if len(observed.Status.ProjectMemberships) != 1 || observed.Status.ProjectMemberships[0].ProjectID != testSecondProjectID || observed.Status.ProjectMemberships[0].MembershipID != "membership-2" {
		t.Fatalf("unexpected organization project membership status: %#v", observed.Status.ProjectMemberships)
	}
	if len(observed.Status.ProjectMemberships[0].Roles) != 1 || observed.Status.ProjectMemberships[0].Roles[0].Slug != testMemberRole {
		t.Fatalf("unexpected organization role status: %#v", observed.Status.ProjectMemberships[0].Roles)
	}
	if conditionStatus(observed.Status.Conditions, readyCondition) != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True, got %#v", observed.Status.Conditions)
	}
	assertKstatusStates(t, observed.Status.Conditions, metav1.ConditionTrue, metav1.ConditionFalse, metav1.ConditionFalse)
}

func TestIdentityReconcilerAdoptsIdentityAndRoleMembership(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodGet && request.URL.Path == testIdentityPath:
			_, _ = writer.Write([]byte(`{"identities":[{"id":"identity-1","name":"workload","projectId":"project-1"}]}`))
		case request.Method == http.MethodGet && request.URL.Path == testIdentityByIDPath:
			_, _ = writer.Write([]byte(`{"identity":{"id":"identity-1","name":"workload","projectId":"project-1"}}`))
		case request.Method == http.MethodGet && request.URL.Path == testMembershipPath:
			_, _ = writer.Write([]byte(`{"identityMembership":{"id":"membership-1","projectId":"project-1","identityId":"identity-1","roles":[{"id":"assignment-1","role":"member","isTemporary":false}]}}`))
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
		ObjectMeta: metav1.ObjectMeta{Name: testWorkload, Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalIdentitySpec{
			ConnectionRef:  infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			ProjectRef:     identityProjectRef(project.Name),
			CreationPolicy: infisicalv1alpha1.CreationPolicyCreateOrAdopt,
			RoleSlugs:      []string{testMemberRole},
		},
	}
	kubeClient := testClient(t, connection, secret, project, identity)
	reconciler := &InfisicalIdentityReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(identity)}); err != nil {
		t.Fatalf("reconcile adopted identity: %v", err)
	}

	var observed infisicalv1alpha1.InfisicalIdentity
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(identity), &observed); err != nil {
		t.Fatalf("get identity: %v", err)
	}
	if observed.Status.IdentityID != testIdentityID || observed.Status.MembershipID != "membership-1" || len(observed.Status.Roles) != 1 || observed.Status.Roles[0].Slug != testMemberRole {
		t.Fatalf("unexpected adopted identity status: %#v", observed.Status)
	}
}

func TestIdentityReconcilerWaitsForProjectStatus(t *testing.T) {
	connection, secret := connectionAndSecret("http://127.0.0.1")
	project := &infisicalv1alpha1.InfisicalProject{
		ObjectMeta: metav1.ObjectMeta{Name: testProject, Namespace: testNamespace},
	}
	identity := &infisicalv1alpha1.InfisicalIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: testWorkload, Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalIdentitySpec{
			ConnectionRef: infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			ProjectRef:    identityProjectRef(project.Name),
			RoleSlugs:     []string{testMemberRole},
		},
	}
	kubeClient := testClient(t, connection, secret, project, identity)
	reconciler := &InfisicalIdentityReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(identity)}); err != nil {
		t.Fatalf("reconcile identity dependency: %v", err)
	}

	var observed infisicalv1alpha1.InfisicalIdentity
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(identity), &observed); err != nil {
		t.Fatalf("get identity: %v", err)
	}
	if conditionReason(observed.Status.Conditions, readyCondition) != "ProjectNotReady" || conditionStatus(observed.Status.Conditions, readyCondition) != metav1.ConditionFalse {
		t.Fatalf("expected project dependency condition, got %#v", observed.Status.Conditions)
	}
	assertKstatusStates(t, observed.Status.Conditions, metav1.ConditionFalse, metav1.ConditionTrue, metav1.ConditionFalse)
}

func TestIdentityReconcilerRejectsPaidRoleSlugs(t *testing.T) {
	tests := []struct {
		name   string
		spec   infisicalv1alpha1.InfisicalIdentitySpec
		reason string
	}{
		{
			name: "project role",
			spec: infisicalv1alpha1.InfisicalIdentitySpec{
				ProjectRef: identityProjectRef(testProject),
				RoleSlugs:  []string{"custom-reader"},
			},
			reason: testConfigurationInvalidReason,
		},
		{
			name: "organization role",
			spec: infisicalv1alpha1.InfisicalIdentitySpec{
				Scope:            infisicalv1alpha1.IdentityScopeOrganization,
				OrganizationRef:  &infisicalv1alpha1.LocalObjectReference{Name: testTenantOrganizationName},
				OrganizationRole: "custom-admin",
			},
			reason: testConfigurationInvalidReason,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity := &infisicalv1alpha1.InfisicalIdentity{
				ObjectMeta: metav1.ObjectMeta{Name: tt.name, Namespace: testNamespace},
				Spec:       tt.spec,
			}
			kubeClient := testClient(t, identity)
			reconciler := &InfisicalIdentityReconciler{Client: kubeClient}
			if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(identity)}); err != nil {
				t.Fatalf("reconcile invalid identity: %v", err)
			}

			var observed infisicalv1alpha1.InfisicalIdentity
			if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(identity), &observed); err != nil {
				t.Fatalf("get identity: %v", err)
			}
			if conditionReason(observed.Status.Conditions, readyCondition) != tt.reason || conditionStatus(observed.Status.Conditions, readyCondition) != metav1.ConditionFalse {
				t.Fatalf("expected configuration failure, got %#v", observed.Status.Conditions)
			}
			assertKstatusStates(t, observed.Status.Conditions, metav1.ConditionFalse, metav1.ConditionFalse, metav1.ConditionTrue)
		})
	}
}

func TestIdentityReconcilerAddsDeleteFinalizer(t *testing.T) {
	connection := &infisicalv1alpha1.InfisicalConnection{
		ObjectMeta: metav1.ObjectMeta{Name: testConnection, Namespace: testNamespace},
	}
	identity := &infisicalv1alpha1.InfisicalIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: testWorkload, Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalIdentitySpec{
			ConnectionRef:  infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			DeletionPolicy: infisicalv1alpha1.DeletionPolicyDelete,
		},
	}
	kubeClient := testClient(t, connection, identity)
	reconciler := &InfisicalIdentityReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(identity)}); err != nil {
		t.Fatalf("reconcile identity finalizer: %v", err)
	}

	var observed infisicalv1alpha1.InfisicalIdentity
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(identity), &observed); err != nil {
		t.Fatalf("get identity: %v", err)
	}
	if len(observed.Finalizers) != 1 {
		t.Fatalf("expected delete finalizer, got %#v", observed.Finalizers)
	}
}

func TestIdentityReconcilerReportsRoleMembershipError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case testIdentityByIDPath:
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"identity":{"id":"identity-1","name":"workload","projectId":"project-1"}}`))
		case testMembershipPath:
			http.Error(writer, "role membership unavailable", http.StatusInternalServerError)
		default:
			http.Error(writer, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	connection, secret := connectionAndSecret(server.URL)
	project := &infisicalv1alpha1.InfisicalProject{
		ObjectMeta: metav1.ObjectMeta{Name: testProject, Namespace: testNamespace},
		Status:     infisicalv1alpha1.InfisicalProjectStatus{ProjectID: testProjectID},
	}
	identity := &infisicalv1alpha1.InfisicalIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: testWorkload, Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalIdentitySpec{
			ConnectionRef: infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			ProjectRef:    identityProjectRef(project.Name),
			RoleSlugs:     []string{testMemberRole},
		},
		Status: infisicalv1alpha1.InfisicalIdentityStatus{IdentityID: testIdentityID, ProjectID: testProjectID},
	}
	kubeClient := testClient(t, connection, secret, project, identity)
	reconciler := &InfisicalIdentityReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(identity)}); err != nil {
		t.Fatalf("reconcile role membership error: %v", err)
	}

	var observed infisicalv1alpha1.InfisicalIdentity
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(identity), &observed); err != nil {
		t.Fatalf("get identity: %v", err)
	}
	if conditionReason(observed.Status.Conditions, readyCondition) != "RoleMembershipReconcileFailed" || conditionStatus(observed.Status.Conditions, readyCondition) != metav1.ConditionFalse {
		t.Fatalf("expected role membership failure condition, got %#v", observed.Status.Conditions)
	}
	assertKstatusStates(t, observed.Status.Conditions, metav1.ConditionFalse, metav1.ConditionTrue, metav1.ConditionFalse)
}

func TestEnvironmentReconcilerWaitsForProjectThenCreatesEnvironment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/api/v1/projects/project-1/environments":
			_, _ = writer.Write([]byte(`{"environment":{"id":"environment-1","name":"QA","slug":"qa","position":4,"projectId":"project-1"}}`))
		case request.Method == http.MethodGet && request.URL.Path == testEnvironmentByIDPath:
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
	}
	kubeClient := testClient(t, connection, secret, project, environment)
	reconciler := &InfisicalEnvironmentReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(environment)}); err != nil {
		t.Fatalf("reconcile environment: %v", err)
	}

	var observed infisicalv1alpha1.InfisicalEnvironment
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(environment), &observed); err != nil {
		t.Fatalf("get environment: %v", err)
	}
	if observed.Status.EnvironmentID != testEnvironmentID || observed.Status.ProjectID != testProjectID || observed.Status.Slug != "qa" {
		t.Fatalf("unexpected environment status: %#v", observed.Status)
	}
	if conditionStatus(observed.Status.Conditions, readyCondition) != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True, got %#v", observed.Status.Conditions)
	}
	assertKstatusStates(t, observed.Status.Conditions, metav1.ConditionTrue, metav1.ConditionFalse, metav1.ConditionFalse)
}

func TestEnvironmentReconcilerRestoresSoftDeletedEnvironment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/projects/project-1/environments/slug/qa":
			_, _ = writer.Write([]byte(`{"environment":{"id":"environment-1","name":"QA","slug":"qa","position":4,"projectId":"project-1","softDeletedAt":"2026-09-09T12:00:00Z"}}`))
		case request.Method == http.MethodPost && request.URL.Path == "/api/v1/projects/project-1/environments/environment-1/restore":
			_, _ = writer.Write([]byte(`{"environment":{"id":"environment-1","name":"QA","slug":"qa","position":4,"projectId":"project-1"}}`))
		case request.Method == http.MethodGet && request.URL.Path == testEnvironmentByIDPath:
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
	environment := &infisicalv1alpha1.InfisicalEnvironment{
		ObjectMeta: metav1.ObjectMeta{Name: "qa", Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalEnvironmentSpec{
			ConnectionRef:   infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			ProjectRef:      infisicalv1alpha1.LocalObjectReference{Name: project.Name},
			EnvironmentName: "QA",
			Slug:            "qa",
			CreationPolicy:  infisicalv1alpha1.CreationPolicyCreateOrAdopt,
		},
	}
	kubeClient := testClient(t, connection, secret, project, environment)
	reconciler := &InfisicalEnvironmentReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(environment)}); err != nil {
		t.Fatalf("reconcile soft-deleted environment: %v", err)
	}

	var observed infisicalv1alpha1.InfisicalEnvironment
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(environment), &observed); err != nil {
		t.Fatalf("get environment: %v", err)
	}
	if observed.Status.EnvironmentID != testEnvironmentID || conditionStatus(observed.Status.Conditions, readyCondition) != metav1.ConditionTrue {
		t.Fatalf("expected restored environment to be Ready, got %#v", observed.Status)
	}
	assertKstatusStates(t, observed.Status.Conditions, metav1.ConditionTrue, metav1.ConditionFalse, metav1.ConditionFalse)
}

func validKubernetesAuthForValidation() *infisicalv1alpha1.InfisicalKubernetesAuth {
	return &infisicalv1alpha1.InfisicalKubernetesAuth{
		ObjectMeta: metav1.ObjectMeta{Name: "workload-auth", Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalKubernetesAuthSpec{
			ConnectionRef:        infisicalv1alpha1.InfisicalConnectionReference{Name: testConnection},
			IdentityRef:          infisicalv1alpha1.LocalObjectReference{Name: testWorkload},
			AllowedNamespaces:    []string{testNamespace},
			AllowedNames:         []string{testWorkload},
			KubernetesHost:       testKubernetesHost,
			CACertSecretRef:      &infisicalv1alpha1.SecretKeyReference{Name: testKubernetesCASecret, Key: testKubernetesCAKey},
			VerifyTLSCertificate: boolPtr(true),
			TokenReviewerJWTSecretRef: &infisicalv1alpha1.SecretKeyReference{
				Name: testKubernetesReviewerSecret,
				Key:  testTokenKey,
			},
			TokenReviewMode:         infisicalv1alpha1.KubernetesTokenReviewModeAPI,
			AccessTokenTrustedIPs:   []infisicalv1alpha1.KubernetesTrustedIP{{IPAddress: "10.0.0.0/8"}},
			AccessTokenTTL:          int64Ptr(3600),
			AccessTokenMaxTTL:       int64Ptr(7200),
			AccessTokenNumUsesLimit: int64Ptr(2),
		},
	}
}

func TestValidateKubernetesAuthSpec(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*infisicalv1alpha1.InfisicalKubernetesAuth)
		message string
	}{
		{
			name: "TLS verification requires CA",
			mutate: func(auth *infisicalv1alpha1.InfisicalKubernetesAuth) {
				auth.Spec.CACertSecretRef = nil
			},
			message: "caCertSecretRef is required when verifyTLSCertificate is true",
		},
		{
			name: "disabled TLS verification rejects CA",
			mutate: func(auth *infisicalv1alpha1.InfisicalKubernetesAuth) {
				auth.Spec.VerifyTLSCertificate = boolPtr(false)
			},
			message: "caCertSecretRef cannot be set when verifyTLSCertificate is false",
		},
		{
			name: "unsupported token review mode",
			mutate: func(auth *infisicalv1alpha1.InfisicalKubernetesAuth) {
				auth.Spec.TokenReviewMode = infisicalv1alpha1.KubernetesTokenReviewMode("gateway")
			},
			message: "only api is available in the free-tier API",
		},
		{
			name: "empty namespace allowlist entry",
			mutate: func(auth *infisicalv1alpha1.InfisicalKubernetesAuth) {
				auth.Spec.AllowedNamespaces = []string{""}
			},
			message: "allowedNamespaces[0] must not be empty",
		},
		{
			name: "invalid trusted IP",
			mutate: func(auth *infisicalv1alpha1.InfisicalKubernetesAuth) {
				auth.Spec.AccessTokenTrustedIPs = []infisicalv1alpha1.KubernetesTrustedIP{{IPAddress: "not-an-ip"}}
			},
			message: "must be an IP address or CIDR range",
		},
		{
			name: "access token lifetime out of range",
			mutate: func(auth *infisicalv1alpha1.InfisicalKubernetesAuth) {
				auth.Spec.AccessTokenTTL = int64Ptr(315360001)
			},
			message: "accessTokenTTL must be between 0 and 315360000 seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := validKubernetesAuthForValidation()
			tt.mutate(auth)
			if err := validateKubernetesAuthSpec(auth); err == nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("expected validation error containing %q, got %v", tt.message, err)
			}
		})
	}
}

func TestKubernetesAuthReconcilerReportsInvalidSpecBeforeDependencies(t *testing.T) {
	auth := validKubernetesAuthForValidation()
	auth.Spec.CACertSecretRef = nil
	kubeClient := testClient(t, auth)
	reconciler := &InfisicalKubernetesAuthReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(auth)}); err != nil {
		t.Fatalf("reconcile invalid Kubernetes Auth: %v", err)
	}

	var observed infisicalv1alpha1.InfisicalKubernetesAuth
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(auth), &observed); err != nil {
		t.Fatalf("get Kubernetes Auth: %v", err)
	}
	if conditionReason(observed.Status.Conditions, readyCondition) != "InvalidSpec" {
		t.Fatalf("expected InvalidSpec, got %#v", observed.Status.Conditions)
	}
	assertKstatusStates(t, observed.Status.Conditions, metav1.ConditionFalse, metav1.ConditionFalse, metav1.ConditionTrue)
}

func TestKubernetesAuthReconcilerAttachesAuthWithSecretBackedCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/api/v1/auth/kubernetes-auth/identities/identity-1":
			_, _ = writer.Write([]byte(`{"identityKubernetesAuth":{"id":"auth-1","identityId":"identity-1","kubernetesHost":"https://kubernetes.default.svc","allowedNamespaces":"default","allowedNames":"workload","allowedAudience":"infisical","tokenReviewMode":"api","verifyTlsCertificate":true,"caCert":"ca-data","tokenReviewerJwt":"reviewer-token","accessTokenTrustedIps":[{"ipAddress":"10.0.0.0/8"}],"accessTokenTTL":3600,"accessTokenMaxTTL":7200,"accessTokenNumUsesLimit":2}}`))
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/auth/kubernetes-auth/identities/identity-1":
			_, _ = writer.Write([]byte(`{"identityKubernetesAuth":{"id":"auth-1","identityId":"identity-1","kubernetesHost":"https://kubernetes.default.svc","allowedNamespaces":"default","allowedNames":"workload","allowedAudience":"infisical","tokenReviewMode":"api","verifyTlsCertificate":true,"caCert":"ca-data","tokenReviewerJwt":"reviewer-token","accessTokenTrustedIps":[{"ipAddress":"10.0.0.0/8"}],"accessTokenTTL":3600,"accessTokenMaxTTL":7200,"accessTokenNumUsesLimit":2}}`))
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
		ObjectMeta: metav1.ObjectMeta{Name: testWorkload, Namespace: testNamespace},
		Status:     infisicalv1alpha1.InfisicalIdentityStatus{IdentityID: testIdentityID, ProjectID: testProjectID},
	}
	auth := &infisicalv1alpha1.InfisicalKubernetesAuth{
		ObjectMeta: metav1.ObjectMeta{Name: "workload-auth", Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalKubernetesAuthSpec{
			ConnectionRef:     infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			IdentityRef:       infisicalv1alpha1.LocalObjectReference{Name: identity.Name},
			KubernetesHost:    "https://kubernetes.default.svc",
			AllowedNamespaces: []string{"default"},
			AllowedNames:      []string{"workload"},
			AllowedAudience:   "infisical",
			CACertSecretRef:   &infisicalv1alpha1.SecretKeyReference{Name: "kubernetes-ca", Key: "ca.crt"},
			TokenReviewerJWTSecretRef: &infisicalv1alpha1.SecretKeyReference{
				Name: "kubernetes-reviewer", Key: testTokenKey,
			},
			VerifyTLSCertificate:    boolPtr(true),
			TokenReviewMode:         infisicalv1alpha1.KubernetesTokenReviewModeAPI,
			AccessTokenTrustedIPs:   []infisicalv1alpha1.KubernetesTrustedIP{{IPAddress: "10.0.0.0/8"}},
			AccessTokenTTL:          int64Ptr(3600),
			AccessTokenMaxTTL:       int64Ptr(7200),
			AccessTokenNumUsesLimit: int64Ptr(2),
		},
	}
	caSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "kubernetes-ca", Namespace: testNamespace}, Data: map[string][]byte{"ca.crt": []byte("ca-data")}}
	reviewerSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "kubernetes-reviewer", Namespace: testNamespace}, Data: map[string][]byte{"token": []byte("reviewer-token")}}
	kubeClient := testClient(t, connection, secret, project, identity, auth, caSecret, reviewerSecret)
	reconciler := &InfisicalKubernetesAuthReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(auth)}); err != nil {
		t.Fatalf("reconcile Kubernetes Auth: %v", err)
	}

	var observed infisicalv1alpha1.InfisicalKubernetesAuth
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(auth), &observed); err != nil {
		t.Fatalf("get Kubernetes Auth: %v", err)
	}
	if observed.Status.AuthID != "auth-1" || !observed.Status.HasCACertificate || !observed.Status.HasTokenReviewerJWT {
		t.Fatalf("unexpected Kubernetes Auth status: %#v", observed.Status)
	}
	if len(observed.Status.AllowedNamespaces) != 1 || observed.Status.AllowedNamespaces[0] != "default" || observed.Status.AccessTokenTTL != 3600 {
		t.Fatalf("unexpected Kubernetes Auth configuration: %#v", observed.Status)
	}
	if conditionStatus(observed.Status.Conditions, readyCondition) != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True, got %#v", observed.Status.Conditions)
	}
	assertKstatusStates(t, observed.Status.Conditions, metav1.ConditionTrue, metav1.ConditionFalse, metav1.ConditionFalse)
}

func TestPersistStatusPreservesConcurrentSpecUpdate(t *testing.T) {
	ctx := context.Background()
	connection := &infisicalv1alpha1.InfisicalConnection{
		ObjectMeta: metav1.ObjectMeta{Name: testConnection, Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalConnectionSpec{
			HostAPI: "https://initial.example/api",
			AuthSecretRef: &infisicalv1alpha1.SecretKeyReference{
				Name: testTokenSecret,
			},
		},
	}
	kubeClient := testClient(t, connection)

	var working infisicalv1alpha1.InfisicalConnection
	if err := kubeClient.Get(ctx, client.ObjectKeyFromObject(connection), &working); err != nil {
		t.Fatalf("get working connection: %v", err)
	}
	before := working.DeepCopy()
	working.Status.ObservedGeneration = working.Generation
	setCondition(&working.Status.Conditions, working.Generation, metav1.ConditionTrue, "Ready", "connection is ready")

	var latest infisicalv1alpha1.InfisicalConnection
	if err := kubeClient.Get(ctx, client.ObjectKeyFromObject(connection), &latest); err != nil {
		t.Fatalf("get latest connection: %v", err)
	}
	latest.Spec.HostAPI = "https://changed.example/api"
	if err := kubeClient.Update(ctx, &latest); err != nil {
		t.Fatalf("update connection spec concurrently: %v", err)
	}

	if err := persistStatus(ctx, kubeClient, &working, before); err != nil {
		t.Fatalf("persist status after concurrent spec update: %v", err)
	}

	var observed infisicalv1alpha1.InfisicalConnection
	if err := kubeClient.Get(ctx, client.ObjectKeyFromObject(connection), &observed); err != nil {
		t.Fatalf("get persisted connection: %v", err)
	}
	if observed.Spec.HostAPI != "https://changed.example/api" {
		t.Fatalf("concurrent spec update was lost: %q", observed.Spec.HostAPI)
	}
	if observed.Status.ObservedGeneration != working.Generation || conditionStatus(observed.Status.Conditions, readyCondition) != metav1.ConditionTrue {
		t.Fatalf("status was not persisted: %#v", observed.Status)
	}
	assertKstatusStates(t, observed.Status.Conditions, metav1.ConditionTrue, metav1.ConditionFalse, metav1.ConditionFalse)
}

func boolPtr(value bool) *bool { return &value }

func int64Ptr(value int64) *int64 { return &value }
