package controller

import (
	"context"
	"testing"

	palworldv1alpha1 "github.com/DataKnifeAI/palworld-operator/api/v1alpha1"
	egv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

func secretsTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(palworldv1alpha1.AddToScheme(scheme))
	utilruntime.Must(gatewayv1.Install(scheme))
	utilruntime.Must(gatewayv1alpha2.Install(scheme))
	utilruntime.Must(egv1a1.AddToScheme(scheme))
	return scheme
}

const (
	testKeepAdmin = "keep-admin"
	testKeepJoin  = "keep-join"
)

func testServerForSecrets(generate bool) *palworldv1alpha1.PalworldServer {
	return &palworldv1alpha1.PalworldServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "palworld-test",
			Namespace: "game-servers",
			UID:       "test-uid",
		},
		Spec: palworldv1alpha1.PalworldServerSpec{
			Gateway:         palworldv1alpha1.GatewayConfig{Address: testGatewayAddress},
			GenerateSecrets: generate,
		},
	}
}

func TestCredentialsSecretName(t *testing.T) {
	server := testServerForSecrets(true)
	if got := credentialsSecretName(server); got != "palworld-test-secrets" {
		t.Fatalf("credentialsSecretName() = %q, want palworld-test-secrets", got)
	}
	server.Spec.CredentialsSecretName = "custom-creds"
	if got := credentialsSecretName(server); got != "custom-creds" {
		t.Fatalf("credentialsSecretName() = %q, want custom-creds", got)
	}
}

func TestGeneratePassword(t *testing.T) {
	a, err := generatePassword()
	if err != nil {
		t.Fatalf("generatePassword() error = %v", err)
	}
	b, err := generatePassword()
	if err != nil {
		t.Fatalf("generatePassword() error = %v", err)
	}
	if a == "" || b == "" {
		t.Fatal("expected non-empty passwords")
	}
	if a == b {
		t.Fatal("expected distinct passwords")
	}
	if len(a) < 16 {
		t.Fatalf("password too short: %q", a)
	}
}

func TestReconcileCredentialsSecretCreatesMissing(t *testing.T) {
	scheme := secretsTestScheme(t)
	server := testServerForSecrets(true)
	secretName := credentialsSecretName(server)

	reconciler := &PalworldServerReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(server).Build(),
		Scheme: scheme,
	}

	admin, join, creds, generated, err := reconciler.resolvePasswords(context.Background(), server)
	if err != nil {
		t.Fatalf("resolvePasswords() error = %v", err)
	}
	if !generated {
		t.Fatal("expected credentialsGenerated=true")
	}
	if creds != secretName {
		t.Fatalf("credentialsSecret = %q, want %q", creds, secretName)
	}
	if admin == "" || join == "" {
		t.Fatalf("expected generated passwords, got admin=%q join=%q", admin, join)
	}

	secret := &corev1.Secret{}
	if err := reconciler.Get(context.Background(), types.NamespacedName{
		Name: secretName, Namespace: server.Namespace,
	}, secret); err != nil {
		t.Fatalf("Get secret: %v", err)
	}
	if secret.Type != corev1.SecretTypeOpaque {
		t.Fatalf("secret type = %q, want Opaque", secret.Type)
	}
	if len(secret.OwnerReferences) != 1 || secret.OwnerReferences[0].UID != server.UID {
		t.Fatalf("owner references = %+v", secret.OwnerReferences)
	}
	if string(secret.Data[secretKeyAdminPassword]) != admin {
		t.Fatalf("admin password mismatch")
	}
	if string(secret.Data[secretKeyServerPassword]) != join {
		t.Fatalf("server password mismatch")
	}
}

func TestReconcileCredentialsSecretDoesNotClobber(t *testing.T) {
	scheme := secretsTestScheme(t)
	server := testServerForSecrets(true)
	secretName := credentialsSecretName(server)

	existing := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: server.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			secretKeyAdminPassword:  []byte(testKeepAdmin),
			secretKeyServerPassword: []byte(testKeepJoin),
		},
	}

	reconciler := &PalworldServerReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(server, existing).Build(),
		Scheme: scheme,
	}

	admin, join, _, _, err := reconciler.resolvePasswords(context.Background(), server)
	if err != nil {
		t.Fatalf("resolvePasswords() error = %v", err)
	}
	if admin != testKeepAdmin || join != testKeepJoin {
		t.Fatalf("clobbered passwords: admin=%q join=%q", admin, join)
	}

	// Second reconcile must still preserve values.
	admin2, join2, _, _, err := reconciler.resolvePasswords(context.Background(), server)
	if err != nil {
		t.Fatalf("second resolvePasswords() error = %v", err)
	}
	if admin2 != testKeepAdmin || join2 != testKeepJoin {
		t.Fatalf("second reconcile clobbered: admin=%q join=%q", admin2, join2)
	}
}

func TestReconcileCredentialsSecretFillsEmptyKeysOnly(t *testing.T) {
	scheme := secretsTestScheme(t)
	server := testServerForSecrets(true)
	secretName := credentialsSecretName(server)

	existing := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: server.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			secretKeyAdminPassword:  []byte(testKeepAdmin),
			secretKeyServerPassword: make([]byte, 0),
		},
	}

	reconciler := &PalworldServerReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(server, existing).Build(),
		Scheme: scheme,
	}

	admin, join, _, _, err := reconciler.resolvePasswords(context.Background(), server)
	if err != nil {
		t.Fatalf("resolvePasswords() error = %v", err)
	}
	if admin != testKeepAdmin {
		t.Fatalf("admin clobbered: %q", admin)
	}
	if join == "" || join == testKeepJoin {
		t.Fatalf("expected newly generated join password, got %q", join)
	}
}

func TestResolvePasswordsBringYourOwn(t *testing.T) {
	scheme := secretsTestScheme(t)
	server := testServerForSecrets(false)
	server.Spec.AdminPasswordSecretRef = defaultSecretKeySelector("palworld-server-secrets", secretKeyAdminPassword)
	server.Spec.ServerPasswordSecretRef = defaultSecretKeySelector("palworld-server-secrets", secretKeyServerPassword)

	existing := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "palworld-server-secrets",
			Namespace: server.Namespace,
		},
		Data: map[string][]byte{
			secretKeyAdminPassword:  []byte("byo-admin"),
			secretKeyServerPassword: []byte("byo-join"),
		},
	}

	reconciler := &PalworldServerReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(server, existing).Build(),
		Scheme: scheme,
	}

	admin, join, creds, generated, err := reconciler.resolvePasswords(context.Background(), server)
	if err != nil {
		t.Fatalf("resolvePasswords() error = %v", err)
	}
	if generated {
		t.Fatal("expected credentialsGenerated=false for BYO")
	}
	if creds != "palworld-server-secrets" {
		t.Fatalf("credentialsSecret = %q", creds)
	}
	if admin != "byo-admin" || join != "byo-join" {
		t.Fatalf("unexpected passwords admin=%q join=%q", admin, join)
	}
}
