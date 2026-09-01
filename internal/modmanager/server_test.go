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
	s, err := New(Config{Root: root, User: DefaultUser, Password: testPassword, Restarter: restarter})
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
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Palworld Mod Manager") {
		t.Fatalf("ui status=%d", rec.Code)
	}
}
