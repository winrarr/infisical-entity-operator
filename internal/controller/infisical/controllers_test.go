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
	testProjectTemplateName          = "platform-defaults"
	testProjectTemplateID            = "template-1"
	testTenantSlug                   = "tenant"
	testUniversalAuthIdentityPath    = "/api/v1/auth/universal-auth/identities/identity-1"
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
	testCustomReaderRole             = "custom-reader"
	testMemberRole                   = "member"
	testProjectsPath                 = "/api/v1/projects"
	testProjectByIDPath              = "/api/v1/projects/project-1"
	testTenantOrganizationName       = "tenant-org"
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
			&infisicalv1alpha1.InfisicalProjectTemplate{},
			&infisicalv1alpha1.InfisicalIdentity{},
			&infisicalv1alpha1.InfisicalEnvironment{},
			&infisicalv1alpha1.InfisicalKubernetesAuth{},
			&infisicalv1alpha1.InfisicalProjectRole{},
			&infisicalv1alpha1.InfisicalIdentityTemplate{},
			&infisicalv1alpha1.InfisicalUniversalAuth{},
		).
		Build()
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
	if len(observed.Status.Conditions) != 1 || observed.Status.Conditions[0].Status != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True, got %#v", observed.Status.Conditions)
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
				SecretRef:        infisicalv1alpha1.UniversalAuthSecretReference{Name: "universal-auth"},
				OrganizationSlug: testTenantSlug,
			},
		},
	}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "universal-auth", Namespace: testNamespace}, Data: map[string][]byte{"clientId": []byte("client-1"), "clientSecret": []byte("client-secret-1")}}
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

func TestIdentityTemplateReconcilerUsesOrganizationAndSecretBackedFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		template := `{"id":"template-1","name":"cluster-auth","orgId":"org-1","authMethod":"kubernetes","templateFields":{"tokenReviewMode":"api","kubernetesHost":"https://kubernetes.default.svc","caCert":"ca-data","verifyTlsCertificate":true,"hasTokenReviewerJwt":true,"allowedAudience":"infisical"}}`
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/api/v1/identity-templates":
			_, _ = writer.Write([]byte(template))
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/identity-templates/"+testProjectTemplateID:
			_, _ = writer.Write([]byte(template))
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/identity-templates/search":
			_, _ = writer.Write([]byte(`{"templates":[],"totalCount":0}`))
		case request.Method == http.MethodPatch && request.URL.Path == "/api/v1/identity-templates/"+testProjectTemplateID:
			_, _ = writer.Write([]byte(template))
		default:
			http.Error(writer, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	connection, secret := connectionAndSecret(server.URL)
	organization := &infisicalv1alpha1.InfisicalOrganization{
		ObjectMeta: metav1.ObjectMeta{Name: testTenantSlug, Namespace: testNamespace},
		Status:     infisicalv1alpha1.InfisicalOrganizationStatus{OrganizationID: testOrganizationID, Conditions: []metav1.Condition{{Type: readyCondition, Status: metav1.ConditionTrue}}},
	}
	template := &infisicalv1alpha1.InfisicalIdentityTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: testClusterAuthName, Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalIdentityTemplateSpec{
			ConnectionRef:   infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			OrganizationRef: infisicalv1alpha1.LocalObjectReference{Name: organization.Name},
			AuthMethod:      infisicalv1alpha1.IdentityTemplateAuthMethodKubernetes,
			Kubernetes: &infisicalv1alpha1.IdentityTemplateKubernetesSpec{
				KubernetesHost:            testKubernetesHost,
				CACertSecretRef:           &infisicalv1alpha1.SecretKeyReference{Name: testKubernetesCASecret, Key: testKubernetesCAKey},
				TokenReviewerJWTSecretRef: &infisicalv1alpha1.SecretKeyReference{Name: testKubernetesReviewerSecret, Key: testTokenKey},
				VerifyTLSCertificate:      boolPtr(true),
				TokenReviewMode:           infisicalv1alpha1.KubernetesTokenReviewModeAPI,
				AllowedAudience:           "infisical",
			},
		},
	}
	caSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: testKubernetesCASecret, Namespace: testNamespace}, Data: map[string][]byte{testKubernetesCAKey: []byte("ca-data")}}
	reviewerSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: testKubernetesReviewerSecret, Namespace: testNamespace}, Data: map[string][]byte{testTokenKey: []byte("reviewer-token")}}
	kubeClient := testClient(t, connection, secret, organization, template, caSecret, reviewerSecret)
	reconciler := &InfisicalIdentityTemplateReconciler{Client: kubeClient}
	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(template)}); err != nil {
		t.Fatalf("reconcile identity template: %v", err)
	}

	var observed infisicalv1alpha1.InfisicalIdentityTemplate
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(template), &observed); err != nil {
		t.Fatalf("get identity template: %v", err)
	}
	if observed.Status.TemplateID != testProjectTemplateID || observed.Status.OrganizationID != testOrganizationID || observed.Status.Kubernetes == nil || !observed.Status.Kubernetes.HasCACertificate || !observed.Status.Kubernetes.HasTokenReviewerJWT || !conditionReady(observed.Status.Conditions) {
		t.Fatalf("unexpected identity template status: %#v", observed.Status)
	}
	encoded, err := json.Marshal(observed.Status)
	if err != nil {
		t.Fatalf("marshal identity template status: %v", err)
	}
	if strings.Contains(string(encoded), "reviewer-token") || strings.Contains(string(encoded), "ca-data") {
		t.Fatal("identity template status exposed a Secret value")
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
	if len(observed.Status.Conditions) != 1 || observed.Status.Conditions[0].Status != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True, got %#v", observed.Status.Conditions)
	}
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
	if observed.Status.OrganizationID != testOrganizationID || len(observed.Status.Conditions) != 1 || observed.Status.Conditions[0].Status != metav1.ConditionTrue {
		t.Fatalf("unexpected adopted organization status: %#v", observed.Status)
	}
}

func TestProjectReconcilerCreatesAndUpdatesProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == testProjectsPath:
			_, _ = writer.Write([]byte(`{"project":{"id":"project-1","name":"demo","slug":"demo-project","orgId":"org-1","environments":[]}}`))
		case request.Method == http.MethodPatch && request.URL.Path == testProjectByIDPath:
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

func TestProjectTemplateAndProjectReconciliersUseTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/api/v1/project-templates":
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode project template request: %v", err)
			}
			if body["name"] != testProjectTemplateName || body["type"] != "secret-manager" {
				t.Errorf("unexpected project template request: %#v", body)
			}
			_, _ = writer.Write([]byte(`{"projectTemplate":{"id":"template-1","name":"platform-defaults","type":"secret-manager","orgId":"org-1","roles":[],"environments":[{"name":"Production","slug":"prod","position":1}],"users":[],"groups":[],"identities":[],"projectManagedIdentities":[]}}`))
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/project-templates/template-1":
			_, _ = writer.Write([]byte(`{"projectTemplate":{"id":"template-1","name":"platform-defaults","type":"secret-manager","orgId":"org-1","roles":[],"environments":[{"name":"Production","slug":"prod","position":1}],"users":[],"groups":[],"identities":[],"projectManagedIdentities":[]}}`))
		case request.Method == http.MethodPost && request.URL.Path == testProjectsPath:
			var body struct {
				Template string `json:"template"`
				KMSKeyID string `json:"kmsKeyId"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode project request: %v", err)
			}
			if body.Template != testProjectTemplateName || body.KMSKeyID != "key-1" {
				t.Errorf("unexpected project request: %#v", body)
			}
			_, _ = writer.Write([]byte(`{"project":{"id":"project-1","name":"demo","slug":"demo","orgId":"org-1","environments":[]}}`))
		case request.Method == http.MethodGet && request.URL.Path == testProjectByIDPath:
			_, _ = writer.Write([]byte(`{"project":{"id":"project-1","name":"demo","slug":"demo","orgId":"org-1","environments":[]}}`))
		default:
			http.Error(writer, fmt.Sprintf("unexpected %s %s", request.Method, request.URL.Path), http.StatusNotFound)
		}
	}))
	defer server.Close()

	connection, secret := connectionAndSecret(server.URL)
	template := &infisicalv1alpha1.InfisicalProjectTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: testProjectTemplateName, Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalProjectTemplateSpec{
			ConnectionRef: infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			TemplateName:  testProjectTemplateName,
			Environments:  []infisicalv1alpha1.ProjectTemplateEnvironment{{Name: "Production", Slug: "prod", Position: 1}},
		},
	}
	kubeClient := testClient(t, connection, secret, template)
	templateReconciler := &InfisicalProjectTemplateReconciler{Client: kubeClient}
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(template)}
	if _, err := templateReconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile project template: %v", err)
	}

	var observedTemplate infisicalv1alpha1.InfisicalProjectTemplate
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &observedTemplate); err != nil {
		t.Fatalf("get project template: %v", err)
	}
	if observedTemplate.Status.TemplateID != testProjectTemplateID || observedTemplate.Status.Name != testProjectTemplateName {
		t.Fatalf("unexpected project template status: %#v", observedTemplate.Status)
	}

	project := &infisicalv1alpha1.InfisicalProject{
		ObjectMeta: metav1.ObjectMeta{Name: testProject, Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalProjectSpec{
			ConnectionRef: infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			TemplateRef:   &infisicalv1alpha1.LocalObjectReference{Name: template.Name},
			KMSKeyID:      "key-1",
		},
	}
	if err := kubeClient.Create(context.Background(), project); err != nil {
		t.Fatalf("create project: %v", err)
	}
	projectReconciler := &InfisicalProjectReconciler{Client: kubeClient}
	if _, err := projectReconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(project)}); err != nil {
		t.Fatalf("reconcile project using template: %v", err)
	}
	var observedProject infisicalv1alpha1.InfisicalProject
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(project), &observedProject); err != nil {
		t.Fatalf("get project: %v", err)
	}
	if observedProject.Status.ProjectID != testProjectID {
		t.Fatalf("unexpected project status: %#v", observedProject.Status)
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
	if len(observed.Status.Conditions) != 1 || observed.Status.Conditions[0].Status != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True, got %#v", observed.Status.Conditions)
	}
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
	if len(observed.Status.Conditions) != 1 || observed.Status.Conditions[0].Status != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True, got %#v", observed.Status.Conditions)
	}
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
	if len(observed.Status.Conditions) != 1 || observed.Status.Conditions[0].Status != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True, got %#v", observed.Status.Conditions)
	}
}

func TestIdentityReconcilerCreatesPermanentRoleMembership(t *testing.T) {
	membershipReads := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == testIdentityPath:
			_, _ = writer.Write([]byte(`{"identity":{"id":"identity-1","name":"workload","projectId":"project-1"}}`))
		case request.Method == http.MethodGet && request.URL.Path == testIdentityByIDPath:
			_, _ = writer.Write([]byte(`{"identity":{"id":"identity-1","name":"workload","projectId":"project-1"}}`))
		case request.Method == http.MethodGet && request.URL.Path == testMembershipPath:
			membershipReads++
			if membershipReads == 1 {
				http.Error(writer, "membership not found", http.StatusNotFound)
				return
			}
			_, _ = writer.Write([]byte(`{"identityMembership":{"id":"membership-1","projectId":"project-1","identityId":"identity-1","roles":[{"id":"assignment-1","role":"custom-reader","customRoleId":"role-1","customRoleName":"Custom Reader","customRoleSlug":"custom-reader","isTemporary":false},{"id":"assignment-2","role":"member","isTemporary":false}]}}`))
		case request.Method == http.MethodPost && request.URL.Path == testMembershipPath:
			var body infisicalMembershipRequest
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode membership request: %v", err)
			}
			if len(body.Roles) != 2 || body.Roles[0].Role != testCustomReaderRole || body.Roles[1].Role != testMemberRole || body.Roles[0].IsTemporary || body.Roles[1].IsTemporary {
				t.Errorf("unexpected membership request: %#v", body)
			}
			_, _ = writer.Write([]byte(`{"identityMembership":{"id":"membership-1","projectId":"project-1","identityId":"identity-1"}}`))
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
			RoleSlugs:     []string{testMemberRole, testCustomReaderRole},
		},
	}
	kubeClient := testClient(t, connection, secret, project, identity)
	reconciler := &InfisicalIdentityReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(identity)}); err != nil {
		t.Fatalf("reconcile identity with roles: %v", err)
	}

	var observed infisicalv1alpha1.InfisicalIdentity
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(identity), &observed); err != nil {
		t.Fatalf("get identity: %v", err)
	}
	if observed.Status.MembershipID != "membership-1" || len(observed.Status.Roles) != 2 {
		t.Fatalf("unexpected identity membership status: %#v", observed.Status)
	}
	if observed.Status.Roles[0].Slug != testCustomReaderRole || observed.Status.Roles[1].Slug != testMemberRole {
		t.Fatalf("unexpected observed role order: %#v", observed.Status.Roles)
	}
	if len(observed.Status.Conditions) != 1 || observed.Status.Conditions[0].Status != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True, got %#v", observed.Status.Conditions)
	}
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
	if len(observed.Status.Conditions) != 1 || observed.Status.Conditions[0].Reason != "ProjectNotReady" || observed.Status.Conditions[0].Status != metav1.ConditionFalse {
		t.Fatalf("expected project dependency condition, got %#v", observed.Status.Conditions)
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

func TestIdentityReconcilerCorrectsRoleMembershipDrift(t *testing.T) {
	membershipReads := 0
	roleUpdates := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodGet && request.URL.Path == testIdentityByIDPath:
			_, _ = writer.Write([]byte(`{"identity":{"id":"identity-1","name":"workload","projectId":"project-1"}}`))
		case request.Method == http.MethodGet && request.URL.Path == testMembershipPath:
			membershipReads++
			if membershipReads == 1 {
				_, _ = writer.Write([]byte(`{"identityMembership":{"id":"membership-1","projectId":"project-1","identityId":"identity-1","roles":[{"id":"assignment-1","role":"custom-reader","customRoleId":"role-1","customRoleName":"Custom Reader","customRoleSlug":"custom-reader","isTemporary":true}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"identityMembership":{"id":"membership-1","projectId":"project-1","identityId":"identity-1","roles":[{"id":"assignment-2","role":"custom-reader","customRoleId":"role-1","customRoleName":"Custom Reader","customRoleSlug":"custom-reader","isTemporary":false}]}}`))
		case request.Method == http.MethodPatch && request.URL.Path == testMembershipPath:
			roleUpdates++
			var body infisicalMembershipRequest
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode membership update: %v", err)
			}
			if len(body.Roles) != 1 || body.Roles[0].Role != testCustomReaderRole || body.Roles[0].IsTemporary {
				t.Errorf("unexpected membership update: %#v", body)
			}
			_, _ = writer.Write([]byte(`{"identityMembership":{"id":"membership-1","projectId":"project-1","identityId":"identity-1"}}`))
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
			RoleSlugs:     []string{testCustomReaderRole},
		},
		Status: infisicalv1alpha1.InfisicalIdentityStatus{IdentityID: testIdentityID, ProjectID: testProjectID},
	}
	kubeClient := testClient(t, connection, secret, project, identity)
	reconciler := &InfisicalIdentityReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(identity)}); err != nil {
		t.Fatalf("reconcile identity role drift: %v", err)
	}
	if roleUpdates != 1 {
		t.Fatalf("expected one role update, got %d", roleUpdates)
	}

	var observed infisicalv1alpha1.InfisicalIdentity
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(identity), &observed); err != nil {
		t.Fatalf("get identity: %v", err)
	}
	if len(observed.Status.Roles) != 1 || observed.Status.Roles[0].Slug != testCustomReaderRole {
		t.Fatalf("unexpected corrected role status: %#v", observed.Status.Roles)
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
			RoleSlugs:     []string{testCustomReaderRole},
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
	if len(observed.Status.Conditions) != 1 || observed.Status.Conditions[0].Reason != "RoleMembershipReconcileFailed" || observed.Status.Conditions[0].Status != metav1.ConditionFalse {
		t.Fatalf("expected role membership failure condition, got %#v", observed.Status.Conditions)
	}
}

func TestEnvironmentReconcilerWaitsForProjectThenCreatesEnvironment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/api/v1/projects/project-1/environments":
			_, _ = writer.Write([]byte(`{"environment":{"id":"environment-1","name":"QA","slug":"qa","position":4,"projectId":"project-1"}}`))
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/projects/project-1/environments/environment-1":
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
	if observed.Status.EnvironmentID != "environment-1" || observed.Status.ProjectID != testProjectID || observed.Status.Slug != "qa" {
		t.Fatalf("unexpected environment status: %#v", observed.Status)
	}
	if len(observed.Status.Conditions) != 1 || observed.Status.Conditions[0].Status != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True, got %#v", observed.Status.Conditions)
	}
}

func TestEnvironmentReconcilerRestoresSoftDeletedEnvironment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/projects/project-1/environments/slug/qa":
			_, _ = writer.Write([]byte(`{"environment":{"id":"environment-1","name":"QA","slug":"qa","position":4,"projectId":"project-1","softDeletedAt":"2026-09-09T12:00:00Z"}}`))
		case request.Method == http.MethodPost && request.URL.Path == "/api/v1/projects/project-1/environments/environment-1/restore":
			_, _ = writer.Write([]byte(`{"environment":{"id":"environment-1","name":"QA","slug":"qa","position":4,"projectId":"project-1"}}`))
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/projects/project-1/environments/environment-1":
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
	if observed.Status.EnvironmentID != "environment-1" || len(observed.Status.Conditions) != 1 || observed.Status.Conditions[0].Status != metav1.ConditionTrue {
		t.Fatalf("expected restored environment to be Ready, got %#v", observed.Status)
	}
}

func TestProjectRoleReconcilerCreatesRoleWithConditions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/api/v1/projects/project-1/roles":
			_, _ = writer.Write([]byte(`{"role":{"id":"role-1","name":"Read Production","slug":"read-production","projectId":"project-1","permissions":[{"subject":"secrets","action":"readValue","conditions":{"environment":{"$eq":"production"}}}]}}`))
		case request.Method == http.MethodGet && request.URL.Path == "/api/v1/projects/roles/role-1":
			_, _ = writer.Write([]byte(`{"role":{"id":"role-1","name":"Read Production","slug":"read-production","projectId":"project-1","permissions":[{"subject":"secrets","action":["readValue"],"conditions":{"environment":{"$eq":"production"}}}]}}`))
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
	role := &infisicalv1alpha1.InfisicalProjectRole{
		ObjectMeta: metav1.ObjectMeta{Name: "read-production", Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalProjectRoleSpec{
			ConnectionRef: infisicalv1alpha1.InfisicalConnectionReference{Name: connection.Name},
			ProjectRef:    infisicalv1alpha1.LocalObjectReference{Name: project.Name},
			RoleName:      "Read Production",
			Slug:          "read-production",
			Permissions: []infisicalv1alpha1.ProjectRolePermission{{
				Subject: "secrets",
				Action:  []infisicalv1alpha1.ProjectRoleAction{"readValue"},
				Conditions: &infisicalv1alpha1.ProjectRoleConditions{
					Environment: &infisicalv1alpha1.ProjectRoleStringCondition{Eq: "production"},
				},
			}},
		},
	}
	kubeClient := testClient(t, connection, secret, project, role)
	reconciler := &InfisicalProjectRoleReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(role)}); err != nil {
		t.Fatalf("reconcile project role: %v", err)
	}

	var observed infisicalv1alpha1.InfisicalProjectRole
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(role), &observed); err != nil {
		t.Fatalf("get project role: %v", err)
	}
	if observed.Status.RoleID != "role-1" || observed.Status.ProjectID != testProjectID || len(observed.Status.Permissions) != 1 {
		t.Fatalf("unexpected project role status: %#v", observed.Status)
	}
	if observed.Status.Permissions[0].Conditions == nil || observed.Status.Permissions[0].Conditions.Environment == nil || observed.Status.Permissions[0].Conditions.Environment.Eq != "production" {
		t.Fatalf("expected environment condition in status: %#v", observed.Status.Permissions)
	}
	if len(observed.Status.Conditions) != 1 || observed.Status.Conditions[0].Status != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True, got %#v", observed.Status.Conditions)
	}
}

func TestValidateProjectRoleSpecRejectsUnsupportedActionCombinations(t *testing.T) {
	valid := &infisicalv1alpha1.InfisicalProjectRole{
		Spec: infisicalv1alpha1.InfisicalProjectRoleSpec{
			Permissions: []infisicalv1alpha1.ProjectRolePermission{{
				Subject: "secrets",
				Action:  []infisicalv1alpha1.ProjectRoleAction{"readValue"},
			}},
		},
	}
	if err := validateProjectRoleSpec(valid); err != nil {
		t.Fatalf("expected valid project role permission: %v", err)
	}

	invalidAction := valid.DeepCopy()
	invalidAction.Spec.Permissions[0].Action = []infisicalv1alpha1.ProjectRoleAction{"assign-role"}
	if err := validateProjectRoleSpec(invalidAction); err == nil {
		t.Fatal("expected action/subject validation error")
	}

	invalidCondition := valid.DeepCopy()
	invalidCondition.Spec.Permissions[0].Subject = "audit-logs"
	invalidCondition.Spec.Permissions[0].Action = []infisicalv1alpha1.ProjectRoleAction{"read"}
	invalidCondition.Spec.Permissions[0].Conditions = &infisicalv1alpha1.ProjectRoleConditions{
		SecretName: &infisicalv1alpha1.ProjectRoleStringCondition{Eq: "database-password"},
	}
	if err := validateProjectRoleSpec(invalidCondition); err == nil {
		t.Fatal("expected condition/subject validation error")
	}
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
				Name: "kubernetes-reviewer", Key: "token",
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
	if len(observed.Status.Conditions) != 1 || observed.Status.Conditions[0].Status != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True, got %#v", observed.Status.Conditions)
	}
}

func TestKubernetesAuthTemplateOmitsTemplateManagedFields(t *testing.T) {
	auth := &infisicalv1alpha1.InfisicalKubernetesAuth{
		Spec: infisicalv1alpha1.InfisicalKubernetesAuthSpec{
			TemplateRef:       &infisicalv1alpha1.LocalObjectReference{Name: testProjectTemplateID},
			AllowedNamespaces: []string{"tenant"},
			AllowedNames:      []string{"workload"},
		},
	}
	request := kubernetesAuthRequestFrom(auth, testProjectTemplateID, "should-not-be-sent", "should-not-be-sent")
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal Kubernetes Auth request: %v", err)
	}
	if string(encoded) != `{"templateId":"template-1","allowedNamespaces":"tenant","allowedNames":"workload"}` {
		t.Fatalf("unexpected template-backed Kubernetes Auth request: %s", encoded)
	}
	auth.Spec.KubernetesHost = testKubernetesHost
	if err := validateKubernetesAuthSpec(auth); err == nil {
		t.Fatal("expected template-managed and per-resource Kubernetes settings to conflict")
	}
}

func TestKubernetesAuthTemplateReferenceRequiresReadyKubernetesTemplate(t *testing.T) {
	auth := &infisicalv1alpha1.InfisicalKubernetesAuth{
		ObjectMeta: metav1.ObjectMeta{Name: "workload-auth", Namespace: testNamespace},
		Spec: infisicalv1alpha1.InfisicalKubernetesAuthSpec{
			TemplateRef: &infisicalv1alpha1.LocalObjectReference{Name: testClusterAuthName},
		},
	}
	wrongMethod := &infisicalv1alpha1.InfisicalIdentityTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-auth", Namespace: testNamespace},
		Spec:       infisicalv1alpha1.InfisicalIdentityTemplateSpec{AuthMethod: infisicalv1alpha1.IdentityTemplateAuthMethodOIDC},
	}
	kubeClient := testClient(t, auth, wrongMethod)
	reconciler := &InfisicalKubernetesAuthReconciler{Client: kubeClient}
	if _, err := reconciler.kubernetesAuthTemplateID(context.Background(), auth); err == nil {
		t.Fatal("expected non-Kubernetes template reference to fail")
	}
	if err := kubeClient.Delete(context.Background(), wrongMethod); err != nil {
		t.Fatalf("delete wrong template: %v", err)
	}
	readyTemplate := &infisicalv1alpha1.InfisicalIdentityTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-auth", Namespace: testNamespace},
		Spec:       infisicalv1alpha1.InfisicalIdentityTemplateSpec{AuthMethod: infisicalv1alpha1.IdentityTemplateAuthMethodKubernetes},
		Status: infisicalv1alpha1.InfisicalIdentityTemplateStatus{
			TemplateID: testProjectTemplateID,
			Conditions: []metav1.Condition{{Type: readyCondition, Status: metav1.ConditionTrue}},
		},
	}
	if err := kubeClient.Create(context.Background(), readyTemplate); err != nil {
		t.Fatalf("create ready template: %v", err)
	}
	readyTemplate.Status.TemplateID = testProjectTemplateID
	readyTemplate.Status.Conditions = []metav1.Condition{{Type: readyCondition, Status: metav1.ConditionTrue}}
	if err := kubeClient.Status().Update(context.Background(), readyTemplate); err != nil {
		t.Fatalf("update ready template status: %v", err)
	}
	if got, err := reconciler.kubernetesAuthTemplateID(context.Background(), auth); err != nil || got != testProjectTemplateID {
		t.Fatalf("expected ready Kubernetes template ID, got %q, %v", got, err)
	}
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
	if observed.Status.ObservedGeneration != working.Generation || len(observed.Status.Conditions) != 1 || observed.Status.Conditions[0].Status != metav1.ConditionTrue {
		t.Fatalf("status was not persisted: %#v", observed.Status)
	}
}

func boolPtr(value bool) *bool { return &value }

func int64Ptr(value int64) *int64 { return &value }
