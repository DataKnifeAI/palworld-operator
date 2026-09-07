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

package modmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	palworldv1alpha1 "github.com/DataKnifeAI/palworld-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const testExpRate20 = "2.0"

type fakeSecrets struct {
	mu   sync.Mutex
	data map[string]*corev1.Secret
}

func (f *fakeSecrets) GetSecret(_ context.Context, name string) (*corev1.Secret, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.data[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return s.DeepCopy(), nil
}

func (f *fakeSecrets) UpdateSecret(_ context.Context, secret *corev1.Secret) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.data == nil {
		f.data = map[string]*corev1.Secret{}
	}
	f.data[secret.Name] = secret.DeepCopy()
	return nil
}

func testSettingsServer(t *testing.T, cr *fakeCR, secrets *fakeSecrets, restURL string, restarter Restarter) *Server {
	t.Helper()
	s, err := New(Config{
		Password:  testPassword,
		RESTBase:  restURL,
		CR:        cr,
		Secrets:   secrets,
		Restarter: restarter,
		SavesRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSettingsGetAndApply(t *testing.T) {
	cr := &fakeCR{server: &palworldv1alpha1.PalworldServer{}}
	cr.server.Name = testCRName
	cr.server.Spec.ServerName = "Island Keep"
	cr.server.Spec.MaxPlayers = 32
	cr.server.Spec.CrossplayPlatforms = "(Steam,Xbox,PS5,Mac)"
	cr.server.Spec.OptionSettings = map[string]string{"ExpRate": testExpRate20, "DeathPenalty": optNone}
	cr.server.Spec.GenerateSecrets = true
	secrets := &fakeSecrets{data: map[string]*corev1.Secret{
		testSecretName: {
			ObjectMeta: metav1.ObjectMeta{Name: testSecretName},
			Data: map[string][]byte{
				secretKeyJoin:  []byte("join-present"),
				secretKeyAdmin: []byte("admin-present"),
			},
		},
	}}
	rest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/api/settings" {
			_, _ = w.Write([]byte(`{"ExpRate":2,"DeathPenalty":"None","ServerPassword":"secret-join"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(rest.Close)
	r := &countingRestarter{}
	s := testSettingsServer(t, cr, secrets, rest.URL, r)

	rec := doAuth(t, s, http.MethodGet, "/api/settings", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got settingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Profile.Name != "Island Keep" || got.Profile.MaxPlayers != 32 || !got.Profile.Steam {
		t.Fatalf("profile = %+v", got.Profile)
	}
	if got.Options["ExpRate"] != testExpRate20 {
		t.Fatalf("options = %+v", got.Options)
	}
	if got.Live["ExpRate"] != "2" || got.Live["ServerPassword"] != liveRedacted {
		t.Fatalf("live = %+v", got.Live)
	}
	if !got.Credentials.Join.Set || !got.Credentials.Admin.Set {
		t.Fatalf("creds = %+v", got.Credentials)
	}

	body := []byte(`{"profile":{"name":"Keep Two","description":"x","maxPlayers":16,"steam":true,"xbox":false,"ps5":true,"mac":true,"community":true},"options":{"ExpRate":"3.0","bIsPvP":"False"}}`)
	rec = doAuth(t, s, http.MethodPut, "/api/settings", bytes.NewReader(body), "application/json")
	if rec.Code != http.StatusOK {
		t.Fatalf("apply status=%d body=%s", rec.Code, rec.Body.String())
	}
	if r.n.Load() != 1 {
		t.Fatalf("recreate = %d", r.n.Load())
	}
	if cr.server.Spec.ServerName != "Keep Two" || cr.server.Spec.MaxPlayers != 16 {
		t.Fatalf("spec profile = %+v", cr.server.Spec)
	}
	if cr.server.Spec.CrossplayPlatforms != "(Steam,PS5,Mac)" {
		t.Fatalf("crossplay = %q", cr.server.Spec.CrossplayPlatforms)
	}
	if cr.server.Spec.Community.Enabled == nil || !*cr.server.Spec.Community.Enabled {
		t.Fatal("community should be enabled")
	}
	if cr.server.Spec.OptionSettings["ExpRate"] != "3.0" || cr.server.Spec.OptionSettings["DeathPenalty"] != "" {
		t.Fatalf("options replaced = %+v", cr.server.Spec.OptionSettings)
	}

	rec = doAuth(t, s, http.MethodPut, "/api/settings", bytes.NewReader([]byte(`{"profile":{"maxPlayers":99}}`)), "application/json")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid apply status=%d", rec.Code)
	}
}

func TestRotateOneKeyLeavesOther(t *testing.T) {
	cr := &fakeCR{server: &palworldv1alpha1.PalworldServer{}}
	cr.server.Name = testCRName
	cr.server.Spec.GenerateSecrets = true
	joinWas := []byte("keep-join")
	adminWas := []byte("old-admin")
	secrets := &fakeSecrets{data: map[string]*corev1.Secret{
		testSecretName: {
			ObjectMeta: metav1.ObjectMeta{Name: testSecretName},
			Data: map[string][]byte{
				secretKeyJoin:  append([]byte(nil), joinWas...),
				secretKeyAdmin: append([]byte(nil), adminWas...),
			},
		},
	}}
	r := &countingRestarter{}
	s := testSettingsServer(t, cr, secrets, "", r)

	rec := doAuth(t, s, http.MethodPost, "/api/credentials/rotate", strings.NewReader(`{"key":"admin"}`), "application/json")
	if rec.Code != http.StatusOK {
		t.Fatalf("rotate status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out rotateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Password == "" || out.Key != credKindAdmin || strings.Contains(out.Password, string(adminWas)) {
		t.Fatalf("rotate = %+v", out)
	}
	sec, err := secrets.GetSecret(context.Background(), testSecretName)
	if err != nil {
		t.Fatal(err)
	}
	if string(sec.Data[secretKeyJoin]) != string(joinWas) {
		t.Fatalf("join clobbered: %q", sec.Data[secretKeyJoin])
	}
	if string(sec.Data[secretKeyAdmin]) == string(adminWas) || string(sec.Data[secretKeyAdmin]) != out.Password {
		t.Fatalf("admin not rotated")
	}
	if r.n.Load() != 1 {
		t.Fatalf("recreate = %d", r.n.Load())
	}
}

func TestParseOptionSettingsTuple(t *testing.T) {
	got := parseOptionSettingsTuple(`OptionSettings=(ExpRate=2.0,DeathPenalty=None,CrossplayPlatforms="(Steam,Xbox)",ServerPassword="secret")`)
	if got["ExpRate"] != testExpRate20 || got["DeathPenalty"] != optNone || got["CrossplayPlatforms"] != "(Steam,Xbox)" {
		t.Fatalf("parsed = %+v", got)
	}
	if got["ServerPassword"] != "secret" {
		t.Fatalf("quoted = %+v", got)
	}
}

func TestLiveSettingsFromINI(t *testing.T) {
	s, err := New(Config{Password: testPassword, SavesRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(s.savesRoot, "Config", "LinuxServer")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(s.savesRoot, "SaveGames"), 0o755); err != nil {
		t.Fatal(err)
	}
	ini := "[/Script/Pal.PalGameWorldSettings]\nOptionSettings=(ExpRate=1.5,AdminPassword=\"pw\")\n"
	if err := os.WriteFile(filepath.Join(cfg, settingsININame), []byte(ini), 0o644); err != nil {
		t.Fatal(err)
	}
	live := s.liveSettingsFromINI()
	if live["ExpRate"] != "1.5" {
		t.Fatalf("ini live = %+v", live)
	}
	redacted := redactLive(live)
	if redacted["AdminPassword"] != liveRedacted {
		t.Fatalf("redact = %+v", redacted)
	}
}

func TestCommunityPtr(t *testing.T) {
	spec := palworldv1alpha1.PalworldServerSpec{Community: palworldv1alpha1.CommunityConfig{Enabled: ptr.To(true)}}
	p := profileFromSpec(spec)
	if !p.Community {
		t.Fatal("expected community")
	}
}
