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
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	banlistFileName = "banlist.txt"
	jsonUserID      = "userid"
)

type playerActionRequest struct {
	UserID  string `json:"userid"`
	Message string `json:"message"`
}

type banEntry struct {
	ID string `json:"id"`
}

type banListResponse struct {
	Bans []banEntry `json:"bans"`
	Path string     `json:"path"`
}

func (s *Server) handleKick(w http.ResponseWriter, r *http.Request) {
	s.proxyPlayerAction(w, r, "/v1/api/kick", true, "Kicked.")
}

func (s *Server) handleBan(w http.ResponseWriter, r *http.Request) {
	s.proxyPlayerAction(w, r, "/v1/api/ban", true, "Banned.")
}

func (s *Server) handleUnban(w http.ResponseWriter, r *http.Request) {
	s.proxyPlayerAction(w, r, "/v1/api/unban", false, "Unbanned.")
}

func (s *Server) proxyPlayerAction(w http.ResponseWriter, r *http.Request, path string, allowMessage bool, okMsg string) {
	if !s.restConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errRESTDisabled})
		return
	}
	var req playerActionRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxRESTBodyBytes)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errInvalidJSON})
		return
	}
	req.UserID = strings.TrimSpace(req.UserID)
	req.Message = strings.TrimSpace(req.Message)
	if req.UserID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "userid is required"})
		return
	}
	body := map[string]string{jsonUserID: req.UserID}
	if allowMessage && req.Message != "" {
		body[restMessageField] = req.Message
	}
	if err := s.restPost(r.Context(), path, body); err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	if path == "/v1/api/unban" {
		_ = s.removeBanlistID(req.UserID)
	}
	writeJSON(w, http.StatusOK, actionResponse{Status: "ok", Message: okMsg})
}

func (s *Server) handleBans(w http.ResponseWriter, _ *http.Request) {
	saveGames, _, err := s.resolveSaveRoots()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: err.Error()})
		return
	}
	rel := filepath.ToSlash(filepath.Join(relToSaves(s.savesRoot, saveGames), banlistFileName))
	ids, readErr := readBanlist(filepath.Join(saveGames, banlistFileName))
	if readErr != nil && !os.IsNotExist(readErr) {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "read banlist failed"})
		return
	}
	bans := make([]banEntry, 0, len(ids))
	for _, id := range ids {
		bans = append(bans, banEntry{ID: id})
	}
	writeJSON(w, http.StatusOK, banListResponse{Bans: bans, Path: rel})
}

func readBanlist(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var out []string
	seen := map[string]struct{}{}
	sc := bufio.NewScanner(io.LimitReader(f, maxRESTBodyBytes))
	for sc.Scan() {
		id := strings.TrimSpace(sc.Text())
		if id == "" || strings.HasPrefix(id, "#") {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, sc.Err()
}

func (s *Server) removeBanlistID(id string) error {
	saveGames, _, err := s.resolveSaveRoots()
	if err != nil {
		return err
	}
	path := filepath.Join(saveGames, banlistFileName)
	ids, err := readBanlist(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	kept := ids[:0]
	for _, existing := range ids {
		if existing != id {
			kept = append(kept, existing)
		}
	}
	if len(kept) == len(ids) {
		return nil
	}
	return os.WriteFile(path, []byte(strings.Join(kept, "\n")+"\n"), 0o644)
}
