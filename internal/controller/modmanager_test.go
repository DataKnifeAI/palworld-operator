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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

func testServerForModManager(mods, manager bool) *palworldv1alpha1.PalworldServer {
	server := testServerForMods(mods)
	server.Spec.ModManager.Enabled = manager
	server.Spec.AdminPasswordSecretRef = &corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{Name: "palworld-test-secrets"},
		Key:                  secretKeyAdminPassword,
	}
	return server
}

func TestValidateModManagerRequiresMods(t *testing.T) {
	server := testServerForModManager(false, true)
	err := validateModManager(server)
	if err == nil {
		t.Fatal("expected error when modManager is enabled without mods")
	}

	server.Spec.Mods.Enabled = true
	if err := validateModManager(server); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	disabled := testServerForModManager(false, false)
	if err := validateModManager(disabled); err != nil {
		t.Fatalf("disabled should be valid: %v", err)
	}
}

func TestGameServicePortsModManager(t *testing.T) {
	off := palworldv1alpha1.PalworldServerSpec{
		Gateway: palworldv1alpha1.GatewayConfig{Address: testGatewayAddress},
	}
	for _, p := range gameServicePorts(off) {
		if p.Name == portNameModManager {
			t.Fatal("disabled must not expose mod-manager port")
		}
	}

	on := palworldv1alpha1.PalworldServerSpec{
		Gateway:    palworldv1alpha1.GatewayConfig{Address: testGatewayAddress},
		Mods:       palworldv1alpha1.ModsConfig{Enabled: true},
		ModManager: palworldv1alpha1.ModManagerConfig{Enabled: true},
	}
	found := false
	for _, p := range gameServicePorts(on) {
		if p.Name == portNameModManager && p.Port == defaultModManagerPort && p.Protocol == corev1.ProtocolTCP {
			found = true
		}
		if p.Name == "game-udp" && p.Protocol != corev1.ProtocolUDP {
			t.Fatal("game-udp must stay UDP")
		}
	}
	if !found {
		t.Fatal("enabled must expose TCP 8088")
	}
}

func TestReconcileDeploymentModManagerSidecar(t *testing.T) {
	scheme := secretsTestScheme(t)
	ctx := context.Background()

	disabled := testServerForModManager(true, false)
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

	enabled := testServerForModManager(true, true)
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
	if side.Name != containerModManager || side.Command[0] != modManagerBinary {
		t.Fatalf("sidecar = %+v", side)
	}
	if mountByName(side.VolumeMounts, volumeMods) == nil {
		t.Fatal("sidecar must mount mods PVC")
	}
	if dep.Spec.Template.Spec.ServiceAccountName != "palworld-test-mod-manager" {
		t.Fatalf("SA = %q", dep.Spec.Template.Spec.ServiceAccountName)
	}
	if dep.Spec.Strategy.Type != appsv1.RecreateDeploymentStrategyType {
		t.Fatal("must keep Recreate")
	}
}

func TestReconcileModManagerRBACAndHTTPRoute(t *testing.T) {
	scheme := secretsTestScheme(t)
	ctx := context.Background()

	disabled := testServerForModManager(true, false)
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
	err := rDisabled.Get(ctx, types.NamespacedName{Name: names.modManagerSA, Namespace: disabled.Namespace}, sa)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("SA should be absent when disabled, got %v", err)
	}
	httpRoute := &gatewayv1.HTTPRoute{}
	err = rDisabled.Get(ctx, types.NamespacedName{Name: names.modManagerHTTPRoute, Namespace: disabled.Namespace}, httpRoute)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("HTTPRoute should be absent when disabled, got %v", err)
	}
	udp := &gatewayv1alpha2.UDPRoute{}
	if err := rDisabled.Get(ctx, types.NamespacedName{Name: names.gameUDPRoute, Namespace: disabled.Namespace}, udp); err != nil {
		t.Fatalf("game UDPRoute must still exist: %v", err)
	}

	enabled := testServerForModManager(true, true)
	rEnabled := &PalworldServerReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(enabled).Build(),
		Scheme: scheme,
	}
	enNames := deriveNames(enabled)
	if err := rEnabled.reconcileModManagerRBAC(ctx, enabled, enNames); err != nil {
		t.Fatal(err)
	}
	if err := rEnabled.reconcileEnvoyGateway(ctx, enabled, enNames); err != nil {
		t.Fatal(err)
	}
	if err := rEnabled.Get(ctx, types.NamespacedName{Name: enNames.modManagerSA, Namespace: enabled.Namespace}, sa); err != nil {
		t.Fatalf("SA missing: %v", err)
	}
	role := &rbacv1.Role{}
	if err := rEnabled.Get(ctx, types.NamespacedName{Name: enNames.modManagerRole, Namespace: enabled.Namespace}, role); err != nil {
		t.Fatalf("Role missing: %v", err)
	}
	if len(role.Rules) != 1 || len(role.Rules[0].ResourceNames) != 1 || role.Rules[0].ResourceNames[0] != enabled.Name {
		t.Fatalf("Role must be least-privilege for %s, got %+v", enabled.Name, role.Rules)
	}
	if err := rEnabled.Get(ctx, types.NamespacedName{Name: enNames.modManagerHTTPRoute, Namespace: enabled.Namespace}, httpRoute); err != nil {
		t.Fatalf("HTTPRoute missing: %v", err)
	}
	gw := &gatewayv1.Gateway{}
	if err := rEnabled.Get(ctx, types.NamespacedName{Name: enNames.gatewayName, Namespace: enabled.Namespace}, gw); err != nil {
		t.Fatal(err)
	}
	var hasHTTP, hasUDP bool
	for _, l := range gw.Spec.Listeners {
		if l.Name == gatewayListenerModManagerHTTP && l.Protocol == gatewayv1.HTTPProtocolType && l.Port == defaultModManagerPort {
			hasHTTP = true
		}
		if l.Name == gatewayListenerGameUDP && l.Protocol == gatewayv1.UDPProtocolType {
			hasUDP = true
		}
	}
	if !hasHTTP {
		t.Fatal("gateway missing HTTP listener for mod manager")
	}
	if !hasUDP {
		t.Fatal("gateway must keep game UDP listener")
	}
	udpEnabled := &gatewayv1alpha2.UDPRoute{}
	if err := rEnabled.Get(ctx, types.NamespacedName{Name: enNames.gameUDPRoute, Namespace: enabled.Namespace}, udpEnabled); err != nil {
		t.Fatalf("game UDPRoute must remain when mod manager is on: %v", err)
	}
}

func TestModManagerDefaults(t *testing.T) {
	spec := palworldv1alpha1.PalworldServerSpec{}
	if modManagerEnabled(spec) {
		t.Fatal("must default disabled")
	}
	if got := modManagerPort(spec); got != defaultModManagerPort {
		t.Fatalf("port = %d", got)
	}
	if got := modManagerImage(spec); got != defaultModManagerImage {
		t.Fatalf("image = %q", got)
	}
}
