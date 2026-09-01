/*
Copyright 2026 DataKnifeAI.

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

package controller

import (
	"context"
	"testing"

	palworldv1alpha1 "github.com/DataKnifeAI/palworld-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

const testServerManagerHost = "palworld.example.test"
const testServerManagerTLS = "wildcard-test-tls"

func testServerForServerManager(mods, manager bool) *palworldv1alpha1.PalworldServer {
	server := testServerForMods(mods)
	server.Spec.ServerManager.Enabled = manager
	server.Spec.AdminPasswordSecretRef = &corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{Name: "palworld-test-secrets"},
		Key:                  secretKeyAdminPassword,
	}
	if manager {
		server.Spec.ServerManager.Hostname = testServerManagerHost
		server.Spec.ServerManager.TLSSecretRef = &corev1.SecretReference{Name: testServerManagerTLS}
	}
	return server
}

func testTLSSecret(namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testServerManagerTLS, Namespace: namespace},
		Type:       corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("dummy-cert"),
			corev1.TLSPrivateKeyKey: []byte("dummy-key"),
		},
	}
}

func TestValidateServerManagerRequiresPassword(t *testing.T) {
	server := testServerForMods(false)
	server.Spec.ServerManager.Enabled = true
	err := validateServerManager(server)
	if err == nil {
		t.Fatal("expected error when serverManager is enabled without admin credentials")
	}

	server.Spec.GenerateSecrets = true
	if err := validateServerManager(server); err == nil {
		t.Fatal("expected error when serverManager is enabled without hostname/tls")
	}
	server.Spec.ServerManager.Hostname = testServerManagerHost
	server.Spec.ServerManager.TLSSecretRef = &corev1.SecretReference{Name: testServerManagerTLS}
	if err := validateServerManager(server); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	disabled := testServerForServerManager(false, false)
	if err := validateServerManager(disabled); err != nil {
		t.Fatalf("disabled should be valid: %v", err)
	}
}

func TestValidateServerManagerAllowsWithoutMods(t *testing.T) {
	server := testServerForServerManager(false, true)
	if err := validateServerManager(server); err != nil {
		t.Fatalf("serverManager without mods should be valid: %v", err)
	}
}

func TestServerManagerAliasFromModManager(t *testing.T) {
	spec := palworldv1alpha1.PalworldServerSpec{
		ModManager: palworldv1alpha1.ServerManagerConfig{Enabled: true, Port: 9099},
	}
	if !serverManagerEnabled(spec) {
		t.Fatal("modManager alias must enable server manager")
	}
	if got := serverManagerPort(spec); got != 9099 {
		t.Fatalf("alias port = %d", got)
	}
	spec.ServerManager.Port = 8081
	if got := serverManagerPort(spec); got != 8081 {
		t.Fatalf("serverManager port should win, got %d", got)
	}
}

func TestGameServicePortsServerManager(t *testing.T) {
	off := palworldv1alpha1.PalworldServerSpec{
		Gateway: palworldv1alpha1.GatewayConfig{Address: testGatewayAddress},
	}
	for _, p := range gameServicePorts(off) {
		if p.Name == portNameServerManager {
			t.Fatal("disabled must not expose server-manager port")
		}
	}

	on := palworldv1alpha1.PalworldServerSpec{
		Gateway:       palworldv1alpha1.GatewayConfig{Address: testGatewayAddress},
		ServerManager: palworldv1alpha1.ServerManagerConfig{Enabled: true},
	}
	found := false
	for _, p := range gameServicePorts(on) {
		if p.Name == portNameServerManager && p.Port == defaultServerManagerPort && p.Protocol == corev1.ProtocolTCP {
			found = true
		}
		if p.Name == gatewayListenerGameUDP && p.Protocol != corev1.ProtocolUDP {
			t.Fatal("game-udp must stay UDP")
		}
	}
	if !found {
		t.Fatal("enabled must expose TCP 8088")
	}
}

func TestReconcileDeploymentServerManagerSidecar(t *testing.T) {
	scheme := secretsTestScheme(t)
	ctx := context.Background()

	disabled := testServerForServerManager(true, false)
	rDisabled := &PalworldServerReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(disabled).Build(),
		Scheme: scheme,
	}
	if err := rDisabled.reconcileDeployment(ctx, disabled, deriveNames(disabled), "admin", "join"); err != nil {
		t.Fatalf("deploy disabled: %v", err)
	}
	dep := &appsv1.Deployment{}
	if err := rDisabled.Get(ctx, types.NamespacedName{Name: disabled.Name, Namespace: disabled.Namespace}, dep); err != nil {
		t.Fatal(err)
	}
	if len(dep.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("disabled containers = %d, want 1", len(dep.Spec.Template.Spec.Containers))
	}
	if dep.Spec.Template.Spec.ServiceAccountName != "" {
		t.Fatalf("disabled SA = %q", dep.Spec.Template.Spec.ServiceAccountName)
	}

	enabled := testServerForServerManager(true, true)
	rEnabled := &PalworldServerReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(enabled).Build(),
		Scheme: scheme,
	}
	if err := rEnabled.reconcileDeployment(ctx, enabled, deriveNames(enabled), "admin", "join"); err != nil {
		t.Fatalf("deploy enabled: %v", err)
	}
	if err := rEnabled.Get(ctx, types.NamespacedName{Name: enabled.Name, Namespace: enabled.Namespace}, dep); err != nil {
		t.Fatal(err)
	}
	if len(dep.Spec.Template.Spec.Containers) != 2 {
		t.Fatalf("enabled containers = %d, want 2", len(dep.Spec.Template.Spec.Containers))
	}
	side := dep.Spec.Template.Spec.Containers[1]
	if side.Name != containerServerManager || side.Command[0] != serverManagerBinary {
		t.Fatalf("sidecar = %+v", side)
	}
	if mountByName(side.VolumeMounts, volumeSaves) == nil {
		t.Fatal("sidecar must mount saves PVC")
	}
	if mountByName(side.VolumeMounts, volumeMods) == nil {
		t.Fatal("sidecar must mount mods PVC when mods enabled")
	}
	if dep.Spec.Template.Spec.ServiceAccountName != "palworld-test-manager" {
		t.Fatalf("SA = %q", dep.Spec.Template.Spec.ServiceAccountName)
	}
	if dep.Spec.Strategy.Type != appsv1.RecreateDeploymentStrategyType {
		t.Fatal("must keep Recreate")
	}
	if side.Resources.Limits.Memory().String() != "1Gi" {
		t.Fatalf("sidecar memory limit = %s, want 1Gi so large uploads do not OOM", side.Resources.Limits.Memory().String())
	}
}

func TestReconcileDeploymentServerManagerWithoutMods(t *testing.T) {
	scheme := secretsTestScheme(t)
	ctx := context.Background()
	enabled := testServerForServerManager(false, true)
	r := &PalworldServerReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(enabled).Build(),
		Scheme: scheme,
	}
	if err := r.reconcileDeployment(ctx, enabled, deriveNames(enabled), "admin", "join"); err != nil {
		t.Fatal(err)
	}
	dep := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: enabled.Name, Namespace: enabled.Namespace}, dep); err != nil {
		t.Fatal(err)
	}
	side := dep.Spec.Template.Spec.Containers[1]
	if mountByName(side.VolumeMounts, volumeSaves) == nil {
		t.Fatal("saves mount required")
	}
	if mountByName(side.VolumeMounts, volumeMods) != nil {
		t.Fatal("mods must not be mounted when spec.mods is off")
	}
}

func TestReconcileServerManagerRBACAndHTTPRoute(t *testing.T) {
	scheme := secretsTestScheme(t)
	ctx := context.Background()

	disabled := testServerForServerManager(true, false)
	rDisabled := &PalworldServerReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(disabled).Build(),
		Scheme: scheme,
	}
	names := deriveNames(disabled)
	if err := rDisabled.reconcileModManagerRBAC(ctx, disabled, names); err != nil {
		t.Fatal(err)
	}
	if err := rDisabled.reconcileEnvoyGateway(ctx, disabled, names); err != nil {
		t.Fatal(err)
	}
	sa := &corev1.ServiceAccount{}
	err := rDisabled.Get(ctx, types.NamespacedName{Name: names.serverManagerSA, Namespace: disabled.Namespace}, sa)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("SA should be absent when disabled, got %v", err)
	}
	httpRoute := &gatewayv1.HTTPRoute{}
	err = rDisabled.Get(ctx, types.NamespacedName{Name: names.serverManagerHTTPRoute, Namespace: disabled.Namespace}, httpRoute)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("HTTPRoute should be absent when disabled, got %v", err)
	}
	udp := &gatewayv1alpha2.UDPRoute{}
	if err := rDisabled.Get(ctx, types.NamespacedName{Name: names.gameUDPRoute, Namespace: disabled.Namespace}, udp); err != nil {
		t.Fatalf("game UDPRoute must still exist: %v", err)
	}

	enabled := testServerForServerManager(true, true)
	rEnabled := &PalworldServerReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(enabled, testTLSSecret(enabled.Namespace)).Build(),
		Scheme: scheme,
	}
	enNames := deriveNames(enabled)
	if err := rEnabled.reconcileModManagerRBAC(ctx, enabled, enNames); err != nil {
		t.Fatal(err)
	}
	if err := rEnabled.reconcileEnvoyGateway(ctx, enabled, enNames); err != nil {
		t.Fatal(err)
	}
	if err := rEnabled.Get(ctx, types.NamespacedName{Name: enNames.serverManagerSA, Namespace: enabled.Namespace}, sa); err != nil {
		t.Fatalf("SA missing: %v", err)
	}
	role := &rbacv1.Role{}
	if err := rEnabled.Get(ctx, types.NamespacedName{Name: enNames.serverManagerRole, Namespace: enabled.Namespace}, role); err != nil {
		t.Fatalf("Role missing: %v", err)
	}
	if len(role.Rules) != 1 || len(role.Rules[0].ResourceNames) != 1 || role.Rules[0].ResourceNames[0] != enabled.Name {
		t.Fatalf("Role must be least-privilege for %s, got %+v", enabled.Name, role.Rules)
	}
	if err := rEnabled.Get(ctx, types.NamespacedName{Name: enNames.serverManagerHTTPRoute, Namespace: enabled.Namespace}, httpRoute); err != nil {
		t.Fatalf("HTTPRoute missing: %v", err)
	}
	gw := &gatewayv1.Gateway{}
	if err := rEnabled.Get(ctx, types.NamespacedName{Name: enNames.gatewayName, Namespace: enabled.Namespace}, gw); err != nil {
		t.Fatal(err)
	}
	var hasHTTP, hasHTTPS, hasUDP bool
	for _, l := range gw.Spec.Listeners {
		if l.Name == gatewayListenerServerManagerHTTP && l.Protocol == gatewayv1.HTTPProtocolType && l.Port == defaultServerManagerPort {
			hasHTTP = true
		}
		if l.Name == gatewayListenerServerManagerHTTPS && l.Protocol == gatewayv1.HTTPSProtocolType && l.Port == defaultServerManagerHTTPSPort {
			hasHTTPS = true
			if l.TLS == nil || l.TLS.Mode == nil || *l.TLS.Mode != gatewayv1.TLSModeTerminate {
				t.Fatal("HTTPS listener must terminate TLS")
			}
			if len(l.TLS.CertificateRefs) != 1 || string(l.TLS.CertificateRefs[0].Name) != testServerManagerTLS {
				t.Fatalf("HTTPS listener cert ref = %+v", l.TLS.CertificateRefs)
			}
		}
		if l.Name == gatewayListenerGameUDP && l.Protocol == gatewayv1.UDPProtocolType {
			hasUDP = true
		}
	}
	if !hasHTTPS {
		t.Fatal("gateway missing HTTPS listener for server manager")
	}
	if !hasHTTP {
		t.Fatal("gateway missing HTTP redirect listener for server manager")
	}
	if !hasUDP {
		t.Fatal("gateway must keep game UDP listener")
	}
	if httpRoute.Spec.ParentRefs[0].SectionName == nil || string(*httpRoute.Spec.ParentRefs[0].SectionName) != gatewayListenerServerManagerHTTPS {
		t.Fatalf("HTTPRoute parent = %+v", httpRoute.Spec.ParentRefs)
	}
	if len(httpRoute.Spec.Rules) == 0 || httpRoute.Spec.Rules[0].Timeouts == nil {
		t.Fatal("HTTPRoute must set upload-safe timeouts")
	}
	if httpRoute.Spec.Rules[0].Timeouts.Request == nil || *httpRoute.Spec.Rules[0].Timeouts.Request != serverManagerHTTPTimeout {
		t.Fatalf("HTTPRoute request timeout = %v", httpRoute.Spec.Rules[0].Timeouts)
	}
	redirect := &gatewayv1.HTTPRoute{}
	if err := rEnabled.Get(ctx, types.NamespacedName{Name: enNames.serverManagerRedirectHTTPRoute, Namespace: enabled.Namespace}, redirect); err != nil {
		t.Fatalf("redirect HTTPRoute missing: %v", err)
	}
	udpEnabled := &gatewayv1alpha2.UDPRoute{}
	if err := rEnabled.Get(ctx, types.NamespacedName{Name: enNames.gameUDPRoute, Namespace: enabled.Namespace}, udpEnabled); err != nil {
		t.Fatalf("game UDPRoute must remain when server manager is on: %v", err)
	}
}

func TestServerManagerDefaults(t *testing.T) {
	spec := palworldv1alpha1.PalworldServerSpec{}
	if serverManagerEnabled(spec) {
		t.Fatal("must default disabled")
	}
	if got := serverManagerPort(spec); got != defaultServerManagerPort {
		t.Fatalf("port = %d", got)
	}
	if got := serverManagerHTTPSPort(spec); got != defaultServerManagerHTTPSPort {
		t.Fatalf("https port = %d", got)
	}
	if got := serverManagerImage(spec); got != defaultServerManagerImage {
		t.Fatalf("image = %q", got)
	}
}
