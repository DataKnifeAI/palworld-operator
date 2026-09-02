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
	"mime/multipart"
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
	logoutPath      = "/logout"
	basicAuthRealm  = `Basic realm="Palworld Server Manager"`
	errModsDisabled = "mods PVC is not mounted; enable spec.mods"
	errRESTDisabled = "Palworld REST is not configured on this sidecar"
	errUploadWrite  = "write failed"
	errUploadMkdir  = "create directory failed"
	errSpaceCheck   = "space check failed"
	headerWWWAuth   = "WWW-Authenticate"
	// DefaultUser is the basic-auth username (same as Palworld REST admin).
	DefaultUser = "admin"
)

// Config is the HTTP admin UI configuration. Password must be non-empty.
type Config struct {
	Root      string
	SavesRoot string
	User      string
	Password  string
	RESTBase  string
	Restarter Restarter
	Client    *http.Client
}

// Server is an authenticated HTTP UI/API (stats, controls, saves, mods).
type Server struct {
	root       string
	savesRoot  string
	user       string
	password   string
	restBase   string
	restarter  Restarter
	httpClient *http.Client
	mux        *http.ServeMux
	usage      func(root string) (diskUsage, error)
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
		return nil, errors.New("server manager password is required")
	}
	if cfg.User == "" {
		cfg.User = DefaultUser
	}
	s := &Server{
		user:       cfg.User,
		password:   cfg.Password,
		restBase:   strings.TrimSpace(cfg.RESTBase),
		restarter:  cfg.Restarter,
		httpClient: cfg.Client,
		mux:        http.NewServeMux(),
	}
	if s.httpClient == nil {
		s.httpClient = &http.Client{Timeout: defaultRESTClient}
	}
	if strings.TrimSpace(cfg.Root) != "" {
		root, err := filepath.Abs(cfg.Root)
		if err != nil {
			return nil, err
		}
		s.root = root
	}
	if strings.TrimSpace(cfg.SavesRoot) != "" {
		saves, err := filepath.Abs(cfg.SavesRoot)
		if err != nil {
			return nil, err
		}
		s.savesRoot = saves
	}
	s.mux.HandleFunc("GET "+healthzPath, s.handleHealthz)
	s.mux.HandleFunc("GET "+logoutPath, s.handleLogout)
	s.mux.HandleFunc("GET /{$}", s.handleUI)
	s.mux.HandleFunc("GET /api/files", s.handleList)
	s.mux.HandleFunc("GET /api/download", s.handleDownload)
	s.mux.HandleFunc("POST /api/upload", s.handleUpload)
	s.mux.HandleFunc("DELETE /api/files", s.handleDelete)
	s.mux.HandleFunc("POST /api/restart", s.handleRestart)
	s.mux.HandleFunc("GET /api/space", s.handleSpace)
	s.mux.HandleFunc("GET /api/stats", s.handleStats)
	s.mux.HandleFunc("POST /api/announce", s.handleAnnounce)
	s.mux.HandleFunc("POST /api/save", s.handleSave)
	s.mux.HandleFunc("POST /api/shutdown", s.handleShutdown)
	s.mux.HandleFunc("GET /api/saves", s.handleSavesList)
	s.mux.HandleFunc("GET /api/saves/download", s.handleSavesDownload)
	s.mux.HandleFunc("POST /api/saves/upload", s.handleSavesUpload)
	return s, nil
}

// ServeHTTP applies basic auth to every path except /healthz and /logout.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != healthzPath && r.URL.Path != logoutPath {
		if !s.authorized(r) {
			if _, _, ok := r.BasicAuth(); !ok {
				log.Printf("unauthorized %s %s (no basic auth)", r.Method, r.URL.Path)
			} else {
				log.Printf("unauthorized %s %s (bad credentials)", r.Method, r.URL.Path)
			}
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

// handleLogout always 401s with WWW-Authenticate so browsers drop cached basic auth.
func (s *Server) handleLogout(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set(headerWWWAuth, basicAuthRealm)
	http.Error(w, "logged out", http.StatusUnauthorized)
}

func (s *Server) handleUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(uiHTML))
}

func (s *Server) modsUsage() (diskUsage, error) {
	if s.usage != nil {
		return s.usage(s.root)
	}
	return diskUsageOf(s.root)
}

func (s *Server) handleSpace(w http.ResponseWriter, _ *http.Request) {
	if s.root == "" {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errModsDisabled})
		return
	}
	usage, err := s.modsUsage()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: errSpaceCheck})
		return
	}
	writeJSON(w, http.StatusOK, usage)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	if s.root == "" {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errModsDisabled})
		return
	}
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
	if s.root == "" {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errModsDisabled})
		return
	}
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
	if s.root == "" {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errModsDisabled})
		return
	}
	usage, usageErr := s.modsUsage()
	if usageErr != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: errSpaceCheck})
		return
	}
	if r.ContentLength > 0 && r.ContentLength > usage.Free {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: spaceError(usage.Free)})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	reader, err := r.MultipartReader()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: uploadErrorMessage(err)})
		return
	}

	var dirRel string
	var written *fileEntry
	var stagedAbs string
	for {
		part, nextErr := reader.NextPart()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			writeJSON(w, uploadStatus(nextErr), errorResponse{Error: uploadErrorMessage(nextErr)})
			return
		}
		switch part.FormName() {
		case "path":
			b, readErr := io.ReadAll(io.LimitReader(part, 4096))
			_ = part.Close()
			if readErr != nil {
				writeJSON(w, uploadStatus(readErr), errorResponse{Error: uploadErrorMessage(readErr)})
				return
			}
			dirRel = strings.TrimSpace(string(b))
			if written != nil && stagedAbs != "" {
				moved, moveErr := s.relocateUpload(*written, stagedAbs, dirRel)
				if moveErr != nil {
					writeJSON(w, uploadStatus(moveErr), errorResponse{Error: moveErr.Error()})
					return
				}
				written = &moved.entry
				stagedAbs = moved.abs
			}
		case "file":
			entry, abs, writeErr := s.streamUploadPart(dirRel, part)
			_ = part.Close()
			if writeErr != nil {
				writeJSON(w, uploadStatus(writeErr), errorResponse{Error: writeErr.Error()})
				return
			}
			written = &entry
			stagedAbs = abs
		default:
			_, _ = io.Copy(io.Discard, io.LimitReader(part, 1<<20))
			_ = part.Close()
		}
	}
	if written == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "file is required"})
		return
	}
	writeJSON(w, http.StatusCreated, *written)
}

func uploadErrorMessage(err error) string {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return "upload too large"
	}
	return "invalid upload"
}

func uploadStatus(err error) int {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return http.StatusRequestEntityTooLarge
	}
	if errors.Is(err, errPathEscape) || errors.Is(err, errEmptyName) {
		return http.StatusBadRequest
	}
	if err != nil && (err.Error() == errUploadWrite || err.Error() == errUploadMkdir || err.Error() == errSpaceCheck) {
		return http.StatusInternalServerError
	}
	return http.StatusBadRequest
}

func joinUploadRel(dirRel, base string) string {
	if dirRel == "" || dirRel == "." {
		return base
	}
	return strings.TrimSuffix(dirRel, "/") + "/" + base
}

func (s *Server) streamUploadPart(dirRel string, part *multipart.Part) (fileEntry, string, error) {
	base, err := safeBaseName(part.FileName())
	if err != nil {
		return fileEntry{}, "", err
	}
	if !isPakName(base) {
		return fileEntry{}, "", errors.New(errNotPak)
	}
	usage, usageErr := s.modsUsage()
	if usageErr != nil {
		return fileEntry{}, "", errors.New(errSpaceCheck)
	}
	if usage.Free <= 0 {
		return fileEntry{}, "", errors.New(spaceError(0))
	}
	destRel := joinUploadRel(dirRel, base)
	abs, err := SafeJoin(s.root, destRel)
	if err != nil {
		return fileEntry{}, "", err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return fileEntry{}, "", errors.New(errUploadMkdir)
	}
	tmp := abs + ".partial"
	dst, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fileEntry{}, "", errors.New(errUploadWrite)
	}
	n, copyErr := io.Copy(dst, io.LimitReader(part, usage.Free+1))
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(tmp)
		if copyErr != nil {
			var maxErr *http.MaxBytesError
			if errors.As(copyErr, &maxErr) {
				return fileEntry{}, "", copyErr
			}
		}
		return fileEntry{}, "", errors.New(errUploadWrite)
	}
	if n > usage.Free {
		_ = os.Remove(tmp)
		return fileEntry{}, "", errors.New(spaceError(usage.Free))
	}
	if err := os.Rename(tmp, abs); err != nil {
		_ = os.Remove(tmp)
		return fileEntry{}, "", errors.New(errUploadWrite)
	}
	return fileEntry{Name: base, Path: destRel, Size: n}, abs, nil
}

type relocatedUpload struct {
	entry fileEntry
	abs   string
}

func (s *Server) relocateUpload(current fileEntry, stagedAbs, dirRel string) (relocatedUpload, error) {
	destRel := joinUploadRel(dirRel, current.Name)
	abs, err := SafeJoin(s.root, destRel)
	if err != nil {
		return relocatedUpload{}, err
	}
	if abs == stagedAbs {
		current.Path = destRel
		return relocatedUpload{entry: current, abs: abs}, nil
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return relocatedUpload{}, errors.New(errUploadMkdir)
	}
	if err := os.Rename(stagedAbs, abs); err != nil {
		return relocatedUpload{}, errors.New(errUploadWrite)
	}
	current.Path = destRel
	return relocatedUpload{entry: current, abs: abs}, nil
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	if s.root == "" {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errModsDisabled})
		return
	}
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
		log.Printf("server manager restart failed: %v", err)
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
		log.Printf("server manager write json: %v", err)
	}
}
