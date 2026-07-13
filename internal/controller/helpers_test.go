package controller

import (
	"strings"
	"testing"

	palworldv1alpha1 "github.com/DataKnifeAI/palworld-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testGatewayAddress = "192.168.14.187"
	testMemLimitLarge  = "7Gi"
)

func TestBuildPalWorldSettingsINI(t *testing.T) {
	spec := palworldv1alpha1.PalworldServerSpec{
		Gateway:           palworldv1alpha1.GatewayConfig{Address: testGatewayAddress},
		ServerName:        `DataKnife "Test"`,
		ServerDescription: "ops",
		MaxPlayers:        4,
		RCON:              palworldv1alpha1.RCONConfig{Enabled: boolPtr(true), Port: 25575},
		RESTAPI:           palworldv1alpha1.RESTAPIConfig{Enabled: boolPtr(true), Port: 8212},
	}

	body := buildPalWorldSettingsINI(spec, "admin", "join")
	for _, want := range []string{
		`[/Script/Pal.PalGameWorldSettings]`,
		`ServerName="DataKnife \"Test\""`,
		`ServerPlayerMaxNum=4`,
		`AdminPassword="admin"`,
		`ServerPassword="join"`,
		`RCONEnabled=True`,
		`RESTAPIEnabled=True`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in %s", want, body)
		}
	}
}

func TestResourcesForPlayerCount(t *testing.T) {
	tests := []struct {
		players  int32
		memReq   string
		memLimit string
	}{
		{players: 4, memReq: "3Gi", memLimit: "6Gi"},
		{players: 8, memReq: "4Gi", memLimit: testMemLimitLarge},
		{players: 16, memReq: "5Gi", memLimit: testMemLimitLarge},
		{players: 32, memReq: "6Gi", memLimit: testMemLimitLarge},
	}

	for _, tt := range tests {
		resources := resourcesForPlayerCount(tt.players)
		if got := resources.Requests.Memory().String(); got != tt.memReq {
			t.Fatalf("players=%d memory request = %s, want %s", tt.players, got, tt.memReq)
		}
		if got := resources.Limits.Memory().String(); got != tt.memLimit {
			t.Fatalf("players=%d memory limit = %s, want %s", tt.players, got, tt.memLimit)
		}
	}
}

func TestDefaultResourcesAutoSelectAndOverride(t *testing.T) {
	auto := defaultResources(palworldv1alpha1.PalworldServerSpec{
		Gateway:    palworldv1alpha1.GatewayConfig{Address: testGatewayAddress},
		MaxPlayers: 4,
	})
	if got := auto.Requests.Memory().String(); got != "3Gi" {
		t.Fatalf("auto-selected memory = %s, want 3Gi", got)
	}

	override := defaultResources(palworldv1alpha1.PalworldServerSpec{
		Gateway:    palworldv1alpha1.GatewayConfig{Address: testGatewayAddress},
		MaxPlayers: 4,
		Resources: &corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resourceQuantity("1"),
				corev1.ResourceMemory: resourceQuantity("2Gi"),
			},
		},
	})
	if got := override.Requests.Memory().String(); got != "2Gi" {
		t.Fatalf("override memory = %s, want 2Gi", got)
	}
}

func TestDeriveNamesPalworldServer(t *testing.T) {
	server := &palworldv1alpha1.PalworldServer{
		ObjectMeta: metav1.ObjectMeta{Name: "palworld-server"},
		Spec: palworldv1alpha1.PalworldServerSpec{
			Gateway: palworldv1alpha1.GatewayConfig{Address: testGatewayAddress},
		},
	}

	names := deriveNames(server)
	checks := map[string]string{
		names.pvcName:        "palworld-server-files",
		names.envoyService:   "palworld-server-envoy",
		names.gatewayName:    "palworld-gateway",
		names.envoyProxyName: "game-palworld-kubevip",
		names.gameUDPRoute:   "palworld-game-udp",
		names.queryUDPRoute:  "palworld-query-udp",
	}
	for got, want := range checks {
		if got != want {
			t.Fatalf("deriveNames() = %q, want %q", got, want)
		}
	}
}

func TestIsCommunityImage(t *testing.T) {
	if isCommunityImage(palworldv1alpha1.PalworldServerSpec{}) {
		t.Fatal("default official image should not be community")
	}
	if !isCommunityImage(palworldv1alpha1.PalworldServerSpec{
		ServerImage: "thijsvanloef/palworld-server-docker:latest",
	}) {
		t.Fatal("thijsvanloef image should be community")
	}
}

func TestOfficialCommandArgs(t *testing.T) {
	args := officialCommandArgs(palworldv1alpha1.PalworldServerSpec{
		GamePort:       8211,
		Multithreading: boolPtr(true),
	})
	joined := strings.Join(args, " ")
	for _, want := range []string{"-port=8211", "-UseMultithreadForDS"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected %q in %v", want, args)
		}
	}
}
