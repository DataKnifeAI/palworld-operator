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
	"time"

	"github.com/DataKnifeAI/palworld-operator/internal/controller"
)

const (
	maxUploadBytes          = 2 << 30
	maxMultipartMem         = 32 << 20
	healthzPath             = "/healthz"
	logoutPath              = "/logout"
	basicAuthRealm          = `Basic realm="Palworld Server Manager"`
	errModsDisabled         = "mods PVC is not mounted; enable spec.mods"
	errRESTDisabled         = "Palworld REST is not configured on this sidecar"
	errInvalidJSON          = "invalid JSON"
	errUploadWrite          = "write failed"
	errUploadMkdir          = "create directory failed"
	errSpaceCheck           = "space check failed"
	errRestartNotConfigured = "restart is not configured"
	errRestartFailed        = "restart failed"
	errReplaceDir           = "cannot replace a directory"
	headerWWWAuth           = "WWW-Authenticate"
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
	CR        ServerCR
	Secrets   SecretStore
	Tags      controller.TagLister
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
	cr         ServerCR
	secrets    SecretStore
	tags       controller.TagLister
	httpClient *http.Client
	mux        *http.ServeMux
	usage      func(root string) (diskUsage, error)
}

type fileEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Dir   bool   `json:"dir"`
	Size  int64  `json:"size"`
	MTime string `json:"mtime,omitempty"`
}

var errFileExists = errors.New("file already exists; send replace=1 to overwrite")

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
		cr:         cfg.CR,
		secrets:    cfg.Secrets,
		tags:       cfg.Tags,
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
	s.mux.HandleFunc("POST /api/kick", s.handleKick)
	s.mux.HandleFunc("POST /api/ban", s.handleBan)
	s.mux.HandleFunc("POST /api/unban", s.handleUnban)
	s.mux.HandleFunc("GET /api/bans", s.handleBans)
	s.mux.HandleFunc("POST /api/shutdown", s.handleShutdown)
	s.mux.HandleFunc("GET /api/settings", s.handleSettingsGet)
	s.mux.HandleFunc("PUT /api/settings", s.handleSettingsApply)
	s.mux.HandleFunc("PATCH /api/settings", s.handleSettingsApply)
	s.mux.HandleFunc("POST /api/credentials/rotate", s.handleCredentialsRotate)
	s.mux.HandleFunc("GET /api/saves", s.handleSavesList)
	s.mux.HandleFunc("GET /api/saves/download", s.handleSavesDownload)
	s.mux.HandleFunc("POST /api/saves/upload", s.handleSavesUpload)
	s.mux.HandleFunc("GET /api/updates", s.handleUpdatesGet)
	s.mux.HandleFunc("POST /api/updates/check", s.handleUpdatesCheck)
	s.mux.HandleFunc("PUT /api/updates", s.handleUpdatesSave)
	s.mux.HandleFunc("POST /api/updates/reset", s.handleUpdatesReset)
	s.mux.HandleFunc("POST /api/updates/force", s.handleUpdatesForce)
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
		mtime := ""
		if infoErr == nil {
			size = info.Size()
			dir = info.IsDir()
			mtime = fileModTime(info)
		}
		child := name
		if rel != "" && rel != "." {
			child = strings.TrimSuffix(rel, "/") + "/" + name
		}
		out = append(out, fileEntry{Name: name, Path: child, Dir: dir, Size: size, MTime: mtime})
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
	if _, usageErr := s.modsUsage(); usageErr != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: errSpaceCheck})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	reader, err := r.MultipartReader()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: uploadErrorMessage(err)})
		return
	}

	replace := isTruthy(r.URL.Query().Get("replace"))
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
				moved, moveErr := s.relocateUpload(*written, stagedAbs, dirRel, replace)
				if moveErr != nil {
					writeJSON(w, uploadStatus(moveErr), errorResponse{Error: moveErr.Error()})
					return
				}
				written = &moved.entry
				stagedAbs = moved.abs
			}
		case "replace":
			b, readErr := io.ReadAll(io.LimitReader(part, 64))
			_ = part.Close()
			if readErr != nil {
				writeJSON(w, uploadStatus(readErr), errorResponse{Error: uploadErrorMessage(readErr)})
				return
			}
			replace = isTruthy(string(b))
		case "file":
			entry, abs, writeErr := s.streamUploadPart(dirRel, part, replace)
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
	if errors.Is(err, errFileExists) {
		return http.StatusConflict
	}
	if err != nil && (err.Error() == errUploadWrite || err.Error() == errUploadMkdir || err.Error() == errSpaceCheck) {
		return http.StatusInternalServerError
	}
	return http.StatusBadRequest
}

func fileModTime(info os.FileInfo) string {
	if info == nil {
		return ""
	}
	return info.ModTime().UTC().Format(time.RFC3339)
}

func withFileMeta(entry fileEntry, abs string) fileEntry {
	info, err := os.Stat(abs)
	if err != nil {
		return entry
	}
	entry.Size = info.Size()
	entry.MTime = fileModTime(info)
	return entry
}

func joinUploadRel(dirRel, base string) string {
	if dirRel == "" || dirRel == "." {
		return base
	}
	return strings.TrimSuffix(dirRel, "/") + "/" + base
}

func (s *Server) streamUploadPart(dirRel string, part *multipart.Part, replace bool) (fileEntry, string, error) {
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
	destRel := joinUploadRel(dirRel, base)
	abs, err := SafeJoin(s.root, destRel)
	if err != nil {
		return fileEntry{}, "", err
	}
	var destSize int64
	if info, statErr := os.Stat(abs); statErr == nil {
		if info.IsDir() {
			return fileEntry{}, "", errors.New(errReplaceDir)
		}
		if !replace {
			return fileEntry{}, "", errFileExists
		}
		destSize = info.Size()
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		return fileEntry{}, "", errors.New(errUploadWrite)
	}
	available := usage.Free + destSize
	if available <= 0 {
		return fileEntry{}, "", errors.New(spaceError(0))
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return fileEntry{}, "", errors.New(errUploadMkdir)
	}
	tmp := abs + ".partial"
	dst, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fileEntry{}, "", errors.New(errUploadWrite)
	}
	n, copyErr := copyReplaceBudget(dst, part, usage.Free, destSize, func() error {
		if destSize <= 0 {
			return nil
		}
		if remErr := os.Remove(abs); remErr != nil && !errors.Is(remErr, fs.ErrNotExist) {
			return remErr
		}
		return nil
	})
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
	if n > available {
		_ = os.Remove(tmp)
		return fileEntry{}, "", errors.New(spaceError(available))
	}
	if err := os.Rename(tmp, abs); err != nil {
		_ = os.Remove(tmp)
		return fileEntry{}, "", errors.New(errUploadWrite)
	}
	return withFileMeta(fileEntry{Name: base, Path: destRel, Size: n}, abs), abs, nil
}

type relocatedUpload struct {
	entry fileEntry
	abs   string
}

func (s *Server) relocateUpload(current fileEntry, stagedAbs, dirRel string, replace bool) (relocatedUpload, error) {
	destRel := joinUploadRel(dirRel, current.Name)
	abs, err := SafeJoin(s.root, destRel)
	if err != nil {
		return relocatedUpload{}, err
	}
	if abs == stagedAbs {
		current.Path = destRel
		return relocatedUpload{entry: withFileMeta(current, abs), abs: abs}, nil
	}
	if info, statErr := os.Stat(abs); statErr == nil {
		if info.IsDir() {
			_ = os.Remove(stagedAbs)
			return relocatedUpload{}, errors.New(errReplaceDir)
		}
		if !replace {
			_ = os.Remove(stagedAbs)
			return relocatedUpload{}, errFileExists
		}
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		_ = os.Remove(stagedAbs)
		return relocatedUpload{}, errors.New(errUploadWrite)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return relocatedUpload{}, errors.New(errUploadMkdir)
	}
	if err := os.Rename(stagedAbs, abs); err != nil {
		_ = os.Remove(stagedAbs)
		return relocatedUpload{}, errors.New(errUploadWrite)
	}
	current.Path = destRel
	return relocatedUpload{entry: withFileMeta(current, abs), abs: abs}, nil
}

func copyReplaceBudget(dst io.Writer, src io.Reader, free, reclaimable int64, reclaim func() error) (int64, error) {
	available := free + reclaimable
	if available < 0 {
		available = 0
	}
	src = io.LimitReader(src, available+1)
	var n int64
	if free > 0 {
		copied, err := io.CopyN(dst, src, free)
		n += copied
		if err == io.EOF {
			return n, nil
		}
		if err != nil {
			return n, err
		}
	}
	peek := make([]byte, 1)
	k, peekErr := src.Read(peek)
	if k == 0 {
		if peekErr != nil && peekErr != io.EOF {
			return n, peekErr
		}
		return n, nil
	}
	if reclaimable > 0 && reclaim != nil {
		if err := reclaim(); err != nil {
			return n, err
		}
	}
	w, werr := dst.Write(peek[:k])
	n += int64(w)
	if werr != nil {
		return n, werr
	}
	extra, err := io.Copy(dst, src)
	n += extra
	if err != nil {
		return n, err
	}
	return n, nil
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

type restartRequest struct {
	SaveFirst bool `json:"saveFirst"`
}

func (s *Server) handleRestart(w http.ResponseWriter, r *http.Request) {
	if s.restarter == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errRestartNotConfigured})
		return
	}
	var req restartRequest
	if r.Body != nil {
		if err := json.NewDecoder(io.LimitReader(r.Body, maxRESTBodyBytes)).Decode(&req); err != nil && err != io.EOF {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: errInvalidJSON})
			return
		}
	}
	ctx := r.Context()
	if req.SaveFirst {
		if !s.restConfigured() {
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errRESTDisabled})
			return
		}
		if err := s.restPost(ctx, "/v1/api/save", map[string]string{}); err != nil {
			writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
			return
		}
	}
	if !s.announceRebootCountdown(w, ctx) {
		return
	}
	if err := s.restarter.Restart(ctx); err != nil {
		log.Printf("server manager restart failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: errRestartFailed})
		return
	}
	msg := "Palworld Deployment Recreate requested. Players will disconnect until Ready."
	if req.SaveFirst {
		msg = "Saved, then Recreate."
	}
	writeJSON(w, http.StatusOK, restartResponse{
		Status:  "restarting",
		Message: msg,
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("server manager write json: %v", err)
	}
}
