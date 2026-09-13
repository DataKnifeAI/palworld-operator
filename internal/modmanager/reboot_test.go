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
	"strings"
	"testing"
	"time"
)

func zeroRebootCountdown(t *testing.T) {
	t.Helper()
	prev := rebootCountdown
	rebootCountdown = 0
	t.Cleanup(func() { rebootCountdown = prev })
}

func testGameREST(t *testing.T, hook func(path, message string)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		msg := ""
		if strings.Contains(r.URL.Path, "announce") {
			var req struct {
				Message string `json:"message"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			msg = req.Message
		}
		if hook != nil {
			hook(r.URL.Path, msg)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRebootAnnounceText(t *testing.T) {
	got := rebootAnnounceText(10 * time.Second)
	if !strings.Contains(got, rebootAnnounceBase) || !strings.Contains(got, "10s") {
		t.Fatalf("start = %q", got)
	}
	if !strings.Contains(rebootAnnounceText(0), "restarting now") {
		t.Fatalf("now = %q", rebootAnnounceText(0))
	}
}

func TestAnnounceRebootCountdown(t *testing.T) {
	zeroRebootCountdown(t)
	var announced []string
	game := testGameREST(t, func(path, msg string) {
		if strings.Contains(path, "announce") {
			announced = append(announced, msg)
		}
	})
	s, err := New(Config{Password: testPassword, RESTBase: game.URL, Client: game.Client()})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	if !s.announceRebootCountdown(rec, t.Context()) {
		t.Fatalf("countdown failed: %s", rec.Body.String())
	}
	if len(announced) != 2 {
		t.Fatalf("announces = %v", announced)
	}
	if !strings.Contains(announced[0], rebootAnnounceBase) || !strings.Contains(announced[1], "restarting now") {
		t.Fatalf("announces = %v", announced)
	}
}
