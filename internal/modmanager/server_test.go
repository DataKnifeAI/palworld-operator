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
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

const testPassword = "test-admin-password"

type countingRestarter struct {
	n atomic.Int32
}

func (c *countingRestarter) Restart(context.Context) error {
	c.n.Add(1)
	return nil
}

func testServer(t *testing.T, restarter Restarter) (*Server, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "paks", "~WorkshopMods"), 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := New(Config{Root: root, SavesRoot: t.TempDir(), User: DefaultUser, Password: testPassword, Restarter: restarter})
	if err != nil {
		t.Fatal(err)
	}
	return s, root
}

func doAuth(t *testing.T, h http.Handler, method, path string, body io.Reader, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.SetBasicAuth(DefaultUser, testPassword)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestNewRequiresPassword(t *testing.T) {
	if _, err := New(Config{Root: t.TempDir(), Password: ""}); err == nil {
		t.Fatal("expected error for empty password")
	}
}

func TestHealthzUnauthenticated(t *testing.T) {
	s, _ := testServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz status = %d", rec.Code)
	}
}

func TestAPIRequiresAuth(t *testing.T) {
	s, _ := testServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/files", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestListUploadDownloadDelete(t *testing.T) {
	s, root := testServer(t, nil)

	rec := doAuth(t, s, http.MethodGet, "/api/files?path=", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", rec.Code, rec.Body.String())
	}
	var listed listResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Entries) == 0 {
		t.Fatal("expected paks dir in listing")
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("path", "paks/~WorkshopMods")
	fw, err := mw.CreateFormFile("file", "demo.pak")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte("pak-bytes")); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	rec = doAuth(t, s, http.MethodPost, "/api/upload", &buf, mw.FormDataContentType())
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload status = %d body=%s", rec.Code, rec.Body.String())
	}
	onDisk := filepath.Join(root, "paks", "~WorkshopMods", "demo.pak")
	got, err := os.ReadFile(onDisk)
	if err != nil || string(got) != "pak-bytes" {
		t.Fatalf("on disk = %q err=%v", got, err)
	}

	rec = doAuth(t, s, http.MethodGet, "/api/download?path="+filepath.ToSlash("paks/~WorkshopMods/demo.pak"), nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("download status = %d", rec.Code)
	}
	if rec.Body.String() != "pak-bytes" {
		t.Fatalf("download body = %q", rec.Body.String())
	}

	rec = doAuth(t, s, http.MethodDelete, "/api/files?path="+filepath.ToSlash("paks/~WorkshopMods/demo.pak"), nil, "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d body=%s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(onDisk); !os.IsNotExist(err) {
		t.Fatalf("file still present: %v", err)
	}
}

func TestUploadRelocatesWhenPathFollowsFile(t *testing.T) {
	s, root := testServer(t, nil)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "later.pak")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte("later-bytes")); err != nil {
		t.Fatal(err)
	}
	if err := mw.WriteField("path", "paks/LogicMods"); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	rec := doAuth(t, s, http.MethodPost, "/api/upload", &buf, mw.FormDataContentType())
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload status = %d body=%s", rec.Code, rec.Body.String())
	}
	onDisk := filepath.Join(root, "paks", "LogicMods", "later.pak")
	got, err := os.ReadFile(onDisk)
	if err != nil || string(got) != "later-bytes" {
		t.Fatalf("on disk = %q err=%v", got, err)
	}
}

func TestUploadRejectsNonPak(t *testing.T) {
	s, root := testServer(t, nil)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("path", "paks/~WorkshopMods")
	fw, err := mw.CreateFormFile("file", "notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte("nope")); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	rec := doAuth(t, s, http.MethodPost, "/api/upload", &buf, mw.FormDataContentType())
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), errNotPak) {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "paks", "~WorkshopMods", "notes.txt")); !os.IsNotExist(err) {
		t.Fatal("non-pak must not be written")
	}
}

func TestUploadRejectsWhenNoSpace(t *testing.T) {
	s, _ := testServer(t, nil)
	s.usage = func(string) (diskUsage, error) {
		return diskUsage{Used: 10, Free: 3, Total: 13}, nil
	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("path", "paks/~WorkshopMods")
	fw, err := mw.CreateFormFile("file", "big.pak")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte("12345")); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	rec := doAuth(t, s, http.MethodPost, "/api/upload", &buf, mw.FormDataContentType())
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "free on the mods PVC") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestSpaceEndpoint(t *testing.T) {
	s, _ := testServer(t, nil)
	s.usage = func(string) (diskUsage, error) {
		return diskUsage{Used: 3200, Free: 6800, Total: 10000}, nil
	}
	rec := doAuth(t, s, http.MethodGet, "/api/space", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"free":6800`) || !strings.Contains(rec.Body.String(), `"used":3200`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestRejectTraversalOnAPI(t *testing.T) {
	s, _ := testServer(t, nil)
	for _, path := range []string{"../etc/passwd", "/etc/passwd", "foo/../../../etc/passwd"} {
		rec := doAuth(t, s, http.MethodGet, "/api/files?path="+path, nil, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("path %q status = %d, want 400", path, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "escapes") {
			t.Fatalf("path %q body = %s", path, rec.Body.String())
		}
		rec = doAuth(t, s, http.MethodDelete, "/api/files?path="+path, nil, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("delete %q status = %d", path, rec.Code)
		}
	}
	rec := doAuth(t, s, http.MethodDelete, "/api/files?path=", nil, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("delete root status = %d", rec.Code)
	}
}

func TestRestart(t *testing.T) {
	r := &countingRestarter{}
	s, _ := testServer(t, r)
	rec := doAuth(t, s, http.MethodPost, "/api/restart", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("restart status = %d body=%s", rec.Code, rec.Body.String())
	}
	if r.n.Load() != 1 {
		t.Fatalf("restart calls = %d", r.n.Load())
	}
}

func TestUIRequiresAuth(t *testing.T) {
	s, _ := testServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
	rec = doAuth(t, s, http.MethodGet, "/", nil, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Palworld Server Manager") {
		t.Fatalf("ui status=%d body missing title", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "data-tab=\"saves\"") {
		t.Fatal("ui must include Saves tab")
	}
	if strings.Contains(rec.Body.String(), "World pulse") {
		t.Fatal("header lede under Server Manager must be removed")
	}
	if !strings.Contains(rec.Body.String(), "hero-world-keeper") {
		t.Fatal("ui must include world-keeper hero")
	}
	if !strings.Contains(rec.Body.String(), "Replace world") {
		t.Fatal("ui must isolate replace-world")
	}
	if !strings.Contains(rec.Body.String(), "Log out") || !strings.Contains(rec.Body.String(), "/logout") {
		t.Fatal("ui must include Log out that calls /logout")
	}
	if strings.Count(rec.Body.String(), `class="btn-logout js-logout"`) != 1 {
		t.Fatal("ui must have exactly one logout control")
	}
	if strings.Contains(rec.Body.String(), "trail__logout") {
		t.Fatal("nav logout duplicate must be removed")
	}
	if !strings.Contains(rec.Body.String(), "mod-progress") || !strings.Contains(rec.Body.String(), "uploadWithProgress") {
		t.Fatal("ui must show mods upload progress")
	}
	if !strings.Contains(rec.Body.String(), "Palworld Server does load community pak files") {
		t.Fatal("ui must lead with community pak support")
	}
	if !strings.Contains(rec.Body.String(), `data-path="paks/~WorkshopMods"`) || !strings.Contains(rec.Body.String(), `const defaultModsPath = "paks/~WorkshopMods"`) {
		t.Fatal("ui must default to paks/~WorkshopMods")
	}
	if !strings.Contains(rec.Body.String(), `accept=".pak"`) {
		t.Fatal("ui must accept .pak only")
	}
	if !strings.Contains(rec.Body.String(), "space-meter") || !strings.Contains(rec.Body.String(), "/api/space") {
		t.Fatal("ui must show mods PVC space bar")
	}
	if !strings.Contains(rec.Body.String(), "mod-notes") {
		t.Fatal("ui must highlight mods notes")
	}
	if !strings.Contains(rec.Body.String(), "Mods typically align to PC players") {
		t.Fatal("ui must note PC-aligned mods")
	}
	if !strings.Contains(rec.Body.String(), "console players may fail to connect") {
		t.Fatal("ui must note Crossplay console caveat")
	}
	if strings.Contains(rec.Body.String(), "Yorkhost") || strings.Contains(rec.Body.String(), "yorkhost") {
		t.Fatal("ui must not mention Yorkhost")
	}
	if strings.Contains(rec.Body.String(), "this image") || strings.Contains(rec.Body.String(), "Linux PalServer does") {
		t.Fatal("ui must drop this-image / Linux PalServer does not load copy")
	}
	if strings.Contains(rec.Body.String(), "Pocketpair mods (Windows)") {
		t.Fatal("Pocketpair mods link must not have a Windows suffix")
	}
	if strings.Contains(rec.Body.String(), "T-Box") || strings.Contains(rec.Body.String(), "PalDressing") {
		t.Fatal("ui must not name the live example mod")
	}
}

func TestLogoutReturns401WithWWWAuthenticate(t *testing.T) {
	s, _ := testServer(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauth logout status = %d", rec.Code)
	}
	if rec.Header().Get(headerWWWAuth) != basicAuthRealm {
		t.Fatalf("WWW-Authenticate = %q", rec.Header().Get(headerWWWAuth))
	}

	rec = doAuth(t, s, http.MethodGet, "/logout", nil, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("authed logout status = %d, want 401 so browsers drop cached basic auth", rec.Code)
	}
	if rec.Header().Get(headerWWWAuth) != basicAuthRealm {
		t.Fatalf("authed WWW-Authenticate = %q", rec.Header().Get(headerWWWAuth))
	}
	if !strings.Contains(rec.Body.String(), "logged out") {
		t.Fatalf("logout body = %q", rec.Body.String())
	}
}

func TestStatsProxiesREST(t *testing.T) {
	game := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != DefaultUser || pass != testPassword {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/api/info":
			_, _ = w.Write([]byte(`{"version":"v1.0.0","servername":"Isle","worldguid":"abc"}`))
		case "/v1/api/metrics":
			_, _ = w.Write([]byte(`{"currentplayernum":2,"maxplayernum":8,"serverfps":30,"days":4,"uptime":3661}`))
		case "/v1/api/players":
			_, _ = w.Write([]byte(`{"players":[{"name":"Lee","level":12}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(game.Close)
	s, err := New(Config{Root: t.TempDir(), Password: testPassword, RESTBase: game.URL, Client: game.Client()})
	if err != nil {
		t.Fatal(err)
	}
	rec := doAuth(t, s, http.MethodGet, "/api/stats", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("stats status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "worldguid") || !strings.Contains(rec.Body.String(), "Lee") {
		t.Fatalf("stats body = %s", rec.Body.String())
	}
}

func TestAnnounceAndSave(t *testing.T) {
	var announce, save int
	game := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		switch r.URL.Path {
		case "/v1/api/announce":
			announce++
		case "/v1/api/save":
			save++
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(game.Close)
	s, err := New(Config{Password: testPassword, RESTBase: game.URL, Client: game.Client()})
	if err != nil {
		t.Fatal(err)
	}
	rec := doAuth(t, s, http.MethodPost, "/api/announce", strings.NewReader(`{"message":"hi"}`), "application/json")
	if rec.Code != http.StatusOK {
		t.Fatalf("announce = %d %s", rec.Code, rec.Body.String())
	}
	rec = doAuth(t, s, http.MethodPost, "/api/save", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("save = %d %s", rec.Code, rec.Body.String())
	}
	if announce != 1 || save != 1 {
		t.Fatalf("announce=%d save=%d", announce, save)
	}
}

func TestSavesDownloadUploadAndTraversal(t *testing.T) {
	saves := t.TempDir()
	if err := os.MkdirAll(filepath.Join(saves, "SaveGames", "0", "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(saves, "SaveGames", "0", "world", "WorldOption.sav"), []byte("sav"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(saves, "Config", "LinuxServer"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(saves, "Config", "LinuxServer", "PalWorldSettings.ini"), []byte(`AdminPassword="secret",ServerPassword="join"`), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := New(Config{SavesRoot: saves, Password: testPassword})
	if err != nil {
		t.Fatal(err)
	}

	rec := doAuth(t, s, http.MethodGet, "/api/saves", nil, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "SaveGames") {
		t.Fatalf("list = %d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "secret") || strings.Contains(rec.Body.String(), "join") {
		t.Fatal("listings must not include INI password values")
	}

	rec = doAuth(t, s, http.MethodGet, "/api/saves/download?includeConfig=1", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("download = %d %s", rec.Code, rec.Body.String())
	}
	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var sawSav, sawINI bool
	for _, f := range zr.File {
		if strings.Contains(f.Name, "WorldOption.sav") {
			sawSav = true
		}
		if strings.Contains(f.Name, "PalWorldSettings.ini") {
			sawINI = true
			rc, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			raw, _ := io.ReadAll(rc)
			_ = rc.Close()
			if bytes.Contains(raw, []byte("secret")) {
				t.Fatal("downloaded INI must redact AdminPassword")
			}
			if !bytes.Contains(raw, []byte("REDACTED")) {
				t.Fatalf("expected redaction, got %s", raw)
			}
		}
	}
	if !sawSav || !sawINI {
		t.Fatal("zip missing save or config")
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("SaveGames/0/world/WorldOption.sav")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("restored")); err != nil {
		t.Fatal(err)
	}
	evil, err := zw.Create("../etc/passwd")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = evil.Write([]byte("nope"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	var mp bytes.Buffer
	mw := multipart.NewWriter(&mp)
	fw, err := mw.CreateFormFile("file", "world.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(buf.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	rec = doAuth(t, s, http.MethodPost, "/api/saves/upload", &mp, mw.FormDataContentType())
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("zip slip upload status = %d body=%s", rec.Code, rec.Body.String())
	}

	buf.Reset()
	zw = zip.NewWriter(&buf)
	w, err = zw.Create("SaveGames/0/world/WorldOption.sav")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("restored")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	mp.Reset()
	mw = multipart.NewWriter(&mp)
	fw, err = mw.CreateFormFile("file", "world.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(buf.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	rec = doAuth(t, s, http.MethodPost, "/api/saves/upload", &mp, mw.FormDataContentType())
	if rec.Code != http.StatusOK {
		t.Fatalf("upload = %d %s", rec.Code, rec.Body.String())
	}
	got, err := os.ReadFile(filepath.Join(saves, "SaveGames", "0", "world", "WorldOption.sav"))
	if err != nil || string(got) != "restored" {
		t.Fatalf("restored = %q err=%v", got, err)
	}
}
