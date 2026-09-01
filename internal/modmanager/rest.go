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
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	restAdminUser     = "admin"
	maxRESTBodyBytes  = 1 << 20
	maxShutdownWait   = 600
	defaultRESTClient = 15 * time.Second
)

type statsResponse struct {
	Info    json.RawMessage `json:"info"`
	Metrics json.RawMessage `json:"metrics"`
	Players json.RawMessage `json:"players"`
	Errors  []string        `json:"errors,omitempty"`
}

type announceRequest struct {
	Message string `json:"message"`
}

type shutdownRequest struct {
	WaitTime int    `json:"waittime"`
	Message  string `json:"message"`
}

type actionResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (s *Server) restConfigured() bool {
	return strings.TrimSpace(s.restBase) != ""
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if !s.restConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errRESTDisabled})
		return
	}
	ctx := r.Context()
	out := statsResponse{
		Info:    json.RawMessage("null"),
		Metrics: json.RawMessage("null"),
		Players: json.RawMessage("null"),
	}
	if raw, err := s.restGet(ctx, "/v1/api/info"); err != nil {
		out.Errors = append(out.Errors, "info: "+err.Error())
	} else {
		out.Info = raw
	}
	if raw, err := s.restGet(ctx, "/v1/api/metrics"); err != nil {
		out.Errors = append(out.Errors, "metrics: "+err.Error())
	} else {
		out.Metrics = raw
	}
	if raw, err := s.restGet(ctx, "/v1/api/players"); err != nil {
		out.Errors = append(out.Errors, "players: "+err.Error())
	} else {
		out.Players = raw
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAnnounce(w http.ResponseWriter, r *http.Request) {
	if !s.restConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errRESTDisabled})
		return
	}
	var req announceRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxRESTBodyBytes)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "message is required"})
		return
	}
	if err := s.restPost(r.Context(), "/v1/api/announce", map[string]string{"message": req.Message}); err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, actionResponse{Status: "ok", Message: "Announcement sent."})
}

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	if !s.restConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errRESTDisabled})
		return
	}
	if err := s.restPost(r.Context(), "/v1/api/save", map[string]string{}); err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, actionResponse{Status: "ok", Message: "World save requested."})
}

func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if !s.restConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errRESTDisabled})
		return
	}
	var req shutdownRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxRESTBodyBytes)).Decode(&req); err != nil && err != io.EOF {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	if req.WaitTime < 0 || req.WaitTime > maxShutdownWait {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "waittime must be 0–600 seconds"})
		return
	}
	body := map[string]any{"waittime": req.WaitTime}
	if strings.TrimSpace(req.Message) != "" {
		body["message"] = req.Message
	}
	if err := s.restPost(r.Context(), "/v1/api/shutdown", body); err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, actionResponse{Status: "ok", Message: "Shutdown requested. The game process will exit; Kubernetes will Recreate the pod."})
}

func (s *Server) restGet(ctx context.Context, path string) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.restURL(path), nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(restAdminUser, s.password)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("REST %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxRESTBodyBytes))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("REST %s: HTTP %d", path, resp.StatusCode)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return json.RawMessage("null"), nil
	}
	if !json.Valid(raw) {
		return nil, fmt.Errorf("REST %s: invalid JSON", path)
	}
	return json.RawMessage(raw), nil
}

func (s *Server) restPost(ctx context.Context, path string, payload any) error {
	var buf bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&buf).Encode(payload); err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.restURL(path), &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(restAdminUser, s.password)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("REST %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxRESTBodyBytes))
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			return fmt.Errorf("REST %s: HTTP %d", path, resp.StatusCode)
		}
		return fmt.Errorf("REST %s: HTTP %d: %s", path, resp.StatusCode, msg)
	}
	return nil
}

func (s *Server) restURL(path string) string {
	return strings.TrimRight(s.restBase, "/") + path
}
