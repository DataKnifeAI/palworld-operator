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
	testRate15         = "1.5"
	testRate20         = "2.0"
	testCRName         = "palworld-server"
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
		"RCONEnabled=" + boolStrTrueINI,
		"RESTAPIEnabled=" + boolStrTrueINI,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in %s", want, body)
		}
	}
}

func TestBuildPalWorldSettingsINIOptionSettingsMerge(t *testing.T) {
	spec := palworldv1alpha1.PalworldServerSpec{
		Gateway:    palworldv1alpha1.GatewayConfig{Address: testGatewayAddress},
		ServerName: "FromCR",
		MaxPlayers: 8,
		OptionSettings: map[string]string{
			"ServerName":                   "FromMap", // CR wins
			"ServerPlayerMaxNum":           "32",      // CR wins
			optionKeyExpRate:               testRate20,
			optionKeyWorkSpeedRate:         testRate15,
			optionKeyEnableNonLoginPenalty: boolStrFalseINI,
			"DeathPenalty":                 iniBareNone,
			"CustomNote":                   `hello "world"`,
		},
	}

	body := buildPalWorldSettingsINI(spec, "admin", "join")
	for _, want := range []string{
		`ServerName="FromCR"`,
		`ServerPlayerMaxNum=8`,
		optionKeyExpRate + "=" + testRate20,
		optionKeyWorkSpeedRate + "=" + testRate15,
		optionKeyEnableNonLoginPenalty + "=" + boolStrFalseINI,
		"DeathPenalty=" + iniBareNone,
		`CustomNote="hello \"world\""`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in %s", want, body)
		}
	}
	if strings.Contains(body, `ServerName="FromMap"`) {
		t.Fatal("optionSettings ServerName must not override CR serverName")
	}
	if strings.Contains(body, "ServerPlayerMaxNum=32") {
		t.Fatal("optionSettings ServerPlayerMaxNum must not override CR maxPlayers")
	}
}

func TestFormatOptionSettingValue(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{boolStrTrueLower, boolStrTrueINI},
		{boolStrFalseINI, boolStrFalseINI},
		{testRate15, testRate15},
		{iniBareNone, iniBareNone},
		{`(Steam,Xbox)`, `(Steam,Xbox)`},
		{`"already"`, `"already"`},
		{`say "hi"`, `"say \"hi\""`},
		{"", `""`},
	}
	for _, tt := range tests {
		if got := formatOptionSettingValue(tt.in); got != tt.want {
			t.Fatalf("formatOptionSettingValue(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestCommunityOptionSettingsEnv(t *testing.T) {
	spec := palworldv1alpha1.PalworldServerSpec{
		OptionSettings: map[string]string{
			optionKeyExpRate:               testRate20,
			optionKeyWorkSpeedRate:         testRate15,
			optionKeyEnableNonLoginPenalty: boolStrFalseINI,
			"UnknownFutureKey":             "1",
		},
	}
	env := communityOptionSettingsEnv(spec)
	got := map[string]string{}
	for _, e := range env {
		got[e.Name] = e.Value
	}
	want := map[string]string{
		"EXP_RATE":                 testRate20,
		"WORK_SPEED_RATE":          testRate15,
		"ENABLE_NON_LOGIN_PENALTY": boolStrFalseLower,
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("env %s = %q, want %q (all=%v)", k, got[k], v, got)
		}
	}
	if _, ok := got["UnknownFutureKey"]; ok {
		t.Fatal("unmapped optionSettings keys must not become env vars")
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
		ObjectMeta: metav1.ObjectMeta{Name: testCRName},
		Spec: palworldv1alpha1.PalworldServerSpec{
			Gateway: palworldv1alpha1.GatewayConfig{Address: testGatewayAddress},
		},
	}

	names := deriveNames(server)
	checks := map[string]string{
		names.pvcName:                "palworld-server-files",
		names.modsPVCName:            "palworld-server-mods",
		names.envoyService:           "palworld-server-envoy",
		names.gatewayName:            "palworld-gateway",
		names.envoyProxyName:         "game-palworld-kubevip",
		names.gameUDPRoute:           "palworld-game-udp",
		names.queryUDPRoute:          "palworld-query-udp",
		names.serverManagerHTTPRoute: "palworld-server-manager",
		names.serverManagerSA:        "palworld-server-manager",
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
	if strings.Contains(joined, workshopDirArgPrefix) {
		t.Fatalf("workshopdir must stay off by default: %v", args)
	}
}
