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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKickBanUnbanAndList(t *testing.T) {
	var kicks, bans, unbans int
	var lastBody map[string]string
	game := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&lastBody)
		switch r.URL.Path {
		case "/v1/api/kick":
			kicks++
		case "/v1/api/ban":
			bans++
		case "/v1/api/unban":
			unbans++
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(game.Close)

	saves := t.TempDir()
	if err := os.MkdirAll(filepath.Join(saves, saveGamesDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(saves, saveGamesDir, banlistFileName), []byte("steam_1\nsteam_2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := New(Config{Password: testPassword, RESTBase: game.URL, Client: game.Client(), SavesRoot: saves})
	if err != nil {
		t.Fatal(err)
	}

	rec := doAuth(t, s, http.MethodPost, "/api/kick", strings.NewReader(`{"userid":"steam_9","message":"bye"}`), "application/json")
	if rec.Code != http.StatusOK || kicks != 1 || lastBody[jsonUserID] != "steam_9" || lastBody[restMessageField] != "bye" {
		t.Fatalf("kick status=%d kicks=%d body=%s last=%v", rec.Code, kicks, rec.Body.String(), lastBody)
	}

	rec = doAuth(t, s, http.MethodPost, "/api/ban", strings.NewReader(`{"userid":"steam_8"}`), "application/json")
	if rec.Code != http.StatusOK || bans != 1 {
		t.Fatalf("ban status=%d bans=%d body=%s", rec.Code, bans, rec.Body.String())
	}

	rec = doAuth(t, s, http.MethodGet, "/api/bans", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("bans status=%d body=%s", rec.Code, rec.Body.String())
	}
	var listed banListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Bans) != 2 || listed.Bans[0].ID != "steam_1" {
		t.Fatalf("listed = %+v", listed)
	}

	rec = doAuth(t, s, http.MethodPost, "/api/unban", strings.NewReader(`{"userid":"steam_1"}`), "application/json")
	if rec.Code != http.StatusOK || unbans != 1 {
		t.Fatalf("unban status=%d unbans=%d body=%s", rec.Code, unbans, rec.Body.String())
	}
	ids, err := readBanlist(filepath.Join(saves, saveGamesDir, banlistFileName))
	if err != nil || len(ids) != 1 || ids[0] != "steam_2" {
		t.Fatalf("banlist after unban = %v err=%v", ids, err)
	}

	rec = doAuth(t, s, http.MethodPost, "/api/kick", strings.NewReader(`{"userid":""}`), "application/json")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty kick status=%d", rec.Code)
	}
}

func TestRestartSaveFirst(t *testing.T) {
	zeroRebootCountdown(t)
	var saved, announced int
	game := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/api/save" {
			saved++
		}
		if r.URL.Path == "/v1/api/announce" {
			announced++
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(game.Close)
	r := &countingRestarter{}
	s, err := New(Config{Password: testPassword, RESTBase: game.URL, Client: game.Client(), Restarter: r})
	if err != nil {
		t.Fatal(err)
	}
	rec := doAuth(t, s, http.MethodPost, "/api/restart", strings.NewReader(`{"saveFirst":true}`), "application/json")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if saved != 1 || announced != 2 || r.n.Load() != 1 {
		t.Fatalf("saved=%d announced=%d restarts=%d", saved, announced, r.n.Load())
	}
	if !strings.Contains(rec.Body.String(), "Saved, then Recreate") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}
