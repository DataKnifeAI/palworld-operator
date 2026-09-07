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
	"strings"
	"sync"
	"testing"

	palworldv1alpha1 "github.com/DataKnifeAI/palworld-operator/api/v1alpha1"
	"k8s.io/utils/ptr"
)

const testNewerTag = "v1.0.2.101000"

type fakeCR struct {
	mu     sync.Mutex
	server *palworldv1alpha1.PalworldServer
}

func (f *fakeCR) Get(context.Context) (*palworldv1alpha1.PalworldServer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.server.DeepCopy(), nil
}

func (f *fakeCR) Update(_ context.Context, server *palworldv1alpha1.PalworldServer) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.server = server.DeepCopy()
	return nil
}

type fakeTags struct {
	tags []string
}

func (f fakeTags) ListTags(context.Context, string) ([]string, error) {
	return append([]string(nil), f.tags...), nil
}

func testUpdateServer(t *testing.T, cr ServerCR, tags []string, restURL string) *Server {
	t.Helper()
	s, err := New(Config{
		Password: testPassword,
		RESTBase: restURL,
		CR:       cr,
		Tags:     fakeTags{tags: tags},
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestValidateAndBuildUpdate(t *testing.T) {
	_, fields := validateAndBuildUpdate(updateSettings{CheckInterval: "banana"})
	if fields["checkInterval"] == "" {
		t.Fatal("expected checkInterval error")
	}
	_, fields = validateAndBuildUpdate(updateSettings{CheckInterval: "0s"})
	if fields["checkInterval"] == "" {
		t.Fatal("expected non-positive interval error")
	}
	_, fields = validateAndBuildUpdate(updateSettings{CheckSchedule: "0 4 * *"})
	if fields["checkSchedule"] == "" {
		t.Fatal("expected cron error")
	}
	cfg, fields := validateAndBuildUpdate(updateSettings{
		CheckInterval:   "6h",
		CheckSchedule:   "0 */6 * * *",
		ApplySchedule:   "@hourly",
		TimeZone:        "UTC",
		ImageRepository: "ghcr.io/pocketpairjp/palserver",
		NotifySchedule:  "60m, 10s",
		OnlyWhenEmpty:   ptr.To(false),
		AutoUpdateImage: true,
	})
	if len(fields) > 0 {
		t.Fatalf("unexpected fields: %v", fields)
	}
	if !cfg.AutoUpdateImage || cfg.CheckInterval != "6h" || cfg.OnlyWhenEmpty == nil || *cfg.OnlyWhenEmpty {
		t.Fatalf("cfg = %+v", cfg)
	}
	if len(cfg.NotifySchedule) != 2 {
		t.Fatalf("notify = %v", cfg.NotifySchedule)
	}
	_, fields = validateAndBuildUpdate(updateSettings{NotifySchedule: "60m, no"})
	if fields["notifySchedule"] == "" {
		t.Fatal("expected notifySchedule error")
	}
	_, fields = validateAndBuildUpdate(updateSettings{TimeZone: "notazone"})
	if fields["timeZone"] == "" {
		t.Fatal("expected timeZone error")
	}
	_, fields = validateAndBuildUpdate(updateSettings{ImageRepository: "nopath"})
	if fields["imageRepository"] == "" {
		t.Fatal("expected imageRepository error")
	}
	empty, fields := validateAndBuildUpdate(updateSettings{OnlyWhenEmpty: ptr.To(true)})
	if len(fields) > 0 || empty.OnlyWhenEmpty != nil || empty.CheckInterval != "" {
		t.Fatalf("empty inherit = %+v fields=%v", empty, fields)
	}
}

func TestUpdatesGetAndSave(t *testing.T) {
	cr := &fakeCR{server: &palworldv1alpha1.PalworldServer{}}
	cr.server.Name = testCRName
	cr.server.Spec.ServerImage = "ghcr.io/pocketpairjp/palserver:v1.0.1.100619"
	s := testUpdateServer(t, cr, []string{"v1.0.1.100619", testNewerTag, "latest"}, "")

	rec := doAuth(t, s, http.MethodGet, "/api/updates", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got updatesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Pinned == "" || got.Latest != testNewerTag || !got.UpdateAvailable {
		t.Fatalf("get = %+v", got)
	}

	body := []byte(`{"autoUpdateImage":true,"checkInterval":"1h","onlyWhenEmpty":true}`)
	rec = doAuth(t, s, http.MethodPut, "/api/updates", bytes.NewReader(body), "application/json")
	if rec.Code != http.StatusOK {
		t.Fatalf("save status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !cr.server.Spec.Update.AutoUpdateImage || cr.server.Spec.Update.CheckInterval != "1h" {
		t.Fatalf("saved spec = %+v", cr.server.Spec.Update)
	}
	if cr.server.Spec.Update.OnlyWhenEmpty != nil {
		t.Fatal("onlyWhenEmpty true should stay unset to inherit default")
	}

	rec = doAuth(t, s, http.MethodPut, "/api/updates", bytes.NewReader([]byte(`{"checkInterval":"nope"}`)), "application/json")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid save status=%d", rec.Code)
	}

	rec = doAuth(t, s, http.MethodPost, "/api/updates/reset", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("reset status=%d body=%s", rec.Code, rec.Body.String())
	}
	if cr.server.Spec.Update.AutoUpdateImage || cr.server.Spec.Update.CheckInterval != "" {
		t.Fatalf("reset spec = %+v", cr.server.Spec.Update)
	}
}

func TestUpdatesForce(t *testing.T) {
	prev := forceCountdown
	forceCountdown = 0
	t.Cleanup(func() { forceCountdown = prev })

	var announced []string
	var saved int
	rest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "save") {
			saved++
		}
		if strings.Contains(r.URL.Path, "announce") {
			var req struct {
				Message string `json:"message"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			announced = append(announced, req.Message)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(rest.Close)

	cr := &fakeCR{server: &palworldv1alpha1.PalworldServer{}}
	cr.server.Spec.ServerImage = "ghcr.io/pocketpairjp/palserver:v1.0.1.100619"
	s := testUpdateServer(t, cr, []string{"v1.0.1.100619", testNewerTag}, rest.URL)

	rec := doAuth(t, s, http.MethodPost, "/api/updates/force", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("force status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.HasSuffix(cr.server.Spec.ServerImage, testNewerTag) {
		t.Fatalf("pinned = %s", cr.server.Spec.ServerImage)
	}
	if saved != 1 {
		t.Fatalf("saves = %d", saved)
	}
	if len(announced) < 2 {
		t.Fatalf("announces = %v", announced)
	}
}

func TestUpdatesDisabled(t *testing.T) {
	s, _ := testServer(t, nil)
	rec := doAuth(t, s, http.MethodGet, "/api/updates", nil, "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestNotifyMessageHint(t *testing.T) {
	if notifyMessageHint("") != "" || notifyMessageHint("hi {version}") != "" {
		t.Fatal("known tokens should pass")
	}
	if notifyMessageHint("{foo}") == "" || notifyMessageHint("hi {") == "" {
		t.Fatal("expected hint")
	}
}
