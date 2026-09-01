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
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxUploadBytes  = 2 << 30
	maxMultipartMem = 32 << 20
	healthzPath     = "/healthz"
	basicAuthRealm  = `Basic realm="Palworld Mod Manager"`
	headerWWWAuth   = "WWW-Authenticate"
	// DefaultUser is the basic-auth username (same as Palworld REST admin).
	DefaultUser = "admin"
)

// Config is the HTTP file-manager configuration. Password must be non-empty.
type Config struct {
	Root      string
	User      string
	Password  string
	Restarter Restarter
}

// Server is an authenticated HTTP UI/API over a mods PVC root.
type Server struct {
	root      string
	user      string
	password  string
	restarter Restarter
	mux       *http.ServeMux
}

type fileEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Dir  bool   `json:"dir"`
	Size int64  `json:"size"`
}

type listResponse struct {
	Path    string      `json:"path"`
	Entries []fileEntry `json:"entries"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type restartResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// New returns a handler. Password must be set — unauthenticated mode is rejected.
func New(cfg Config) (*Server, error) {
	if strings.TrimSpace(cfg.Password) == "" {
		return nil, errors.New("mod manager password is required")
	}
	if cfg.User == "" {
		cfg.User = DefaultUser
	}
	root, err := filepath.Abs(cfg.Root)
	if err != nil {
		return nil, err
	}
	s := &Server{
		root:      root,
		user:      cfg.User,
		password:  cfg.Password,
		restarter: cfg.Restarter,
		mux:       http.NewServeMux(),
	}
	s.mux.HandleFunc("GET "+healthzPath, s.handleHealthz)
	s.mux.HandleFunc("GET /{$}", s.handleUI)
	s.mux.HandleFunc("GET /api/files", s.handleList)
	s.mux.HandleFunc("GET /api/download", s.handleDownload)
	s.mux.HandleFunc("POST /api/upload", s.handleUpload)
	s.mux.HandleFunc("DELETE /api/files", s.handleDelete)
	s.mux.HandleFunc("POST /api/restart", s.handleRestart)
	return s, nil
}

// ServeHTTP applies basic auth to every path except /healthz.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != healthzPath {
		if !s.authorized(r) {
			w.Header().Set(headerWWWAuth, basicAuthRealm)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) authorized(r *http.Request) bool {
	user, pass, ok := r.BasicAuth()
	if !ok {
		return false
	}
	userOK := subtle.ConstantTimeCompare([]byte(user), []byte(s.user)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(pass), []byte(s.password)) == 1
	return userOK && passOK
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(uiHTML))
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	abs, err := SafeJoin(s.root, rel)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list failed"})
		return
	}
	out := make([]fileEntry, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if name == "." || name == ".." {
			continue
		}
		info, infoErr := e.Info()
		size := int64(0)
		dir := e.IsDir()
		if infoErr == nil {
			size = info.Size()
			dir = info.IsDir()
		}
		child := name
		if rel != "" && rel != "." {
			child = strings.TrimSuffix(rel, "/") + "/" + name
		}
		out = append(out, fileEntry{Name: name, Path: child, Dir: dir, Size: size})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Dir != out[j].Dir {
			return out[i].Dir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	writeJSON(w, http.StatusOK, listResponse{Path: rel, Entries: out})
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	abs, err := SafeJoin(s.root, rel)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	info, err := os.Stat(abs)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
		return
	}
	if info.IsDir() {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "cannot download a directory"})
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+strings.ReplaceAll(filepath.Base(abs), `"`, "")+`"`)
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, abs)
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxMultipartMem); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid upload"})
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "file is required"})
		return
	}
	defer func() { _ = file.Close() }()
	base, err := safeBaseName(hdr.Filename)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	dirRel := r.FormValue("path")
	destRel := base
	if dirRel != "" && dirRel != "." {
		destRel = strings.TrimSuffix(dirRel, "/") + "/" + base
	}
	abs, err := SafeJoin(s.root, destRel)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "create directory failed"})
		return
	}
	dst, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "write failed"})
		return
	}
	defer func() { _ = dst.Close() }()
	if _, err := io.Copy(dst, file); err != nil {
		_ = os.Remove(abs)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "write failed"})
		return
	}
	writeJSON(w, http.StatusCreated, fileEntry{Name: base, Path: destRel, Size: hdr.Size})
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	if rel == "" || rel == "." {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "cannot delete mods root"})
		return
	}
	abs, err := SafeJoin(s.root, rel)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	rootAbs, err := filepath.EvalSymlinks(s.root)
	if err == nil && abs == rootAbs {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "cannot delete mods root"})
		return
	}
	if err := os.RemoveAll(abs); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "delete failed"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRestart(w http.ResponseWriter, r *http.Request) {
	if s.restarter == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "restart is not configured"})
		return
	}
	if err := s.restarter.Restart(r.Context()); err != nil {
		log.Printf("mod manager restart failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "restart failed"})
		return
	}
	writeJSON(w, http.StatusOK, restartResponse{
		Status:  "restarting",
		Message: "Palworld Deployment Recreate requested. Players will disconnect until Ready.",
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("mod manager write json: %v", err)
	}
}
