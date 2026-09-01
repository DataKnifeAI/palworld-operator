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
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	saveGamesDir       = "SaveGames"
	configLinuxDir     = "Config/LinuxServer"
	incomingDirName    = ".restore-incoming"
	outgoingDirName    = ".restore-outgoing"
	maxSaveUploadBytes = 8 << 30
	saveSizeWarnBytes  = 256 << 20
	saveArchivePrefix  = "palworld-save"
	iniRedactedValue   = `"REDACTED"`
)

var (
	errSavesNotMounted = errors.New("saves volume is not mounted")
	iniSecretAssign    = regexp.MustCompile(`(?i)((?:Admin|Server)Password)\s*=\s*("[^"]*"|[^,"\s\)]+)`)
)

type saveListResponse struct {
	SaveGames    []fileEntry `json:"saveGames"`
	Config       []fileEntry `json:"config"`
	TotalBytes   int64       `json:"totalBytes"`
	Warning      string      `json:"warning,omitempty"`
	SaveGamesRel string      `json:"saveGamesRel"`
	ConfigRel    string      `json:"configRel"`
}

func (s *Server) savesConfigured() bool {
	return strings.TrimSpace(s.savesRoot) != ""
}

func (s *Server) handleSavesList(w http.ResponseWriter, _ *http.Request) {
	saveGames, config, err := s.resolveSaveRoots()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: err.Error()})
		return
	}
	sgRel := relToSaves(s.savesRoot, saveGames)
	cfgRel := relToSaves(s.savesRoot, config)
	sgEntries, sgSize, err := listDirEntries(saveGames, sgRel)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list SaveGames failed"})
		return
	}
	cfgEntries, _, err := listDirEntries(config, cfgRel)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list Config failed"})
		return
	}
	total := sgSize
	warn := ""
	if total >= saveSizeWarnBytes {
		warn = fmt.Sprintf("World save is %s — download may be slow and the zip can be large.", formatBytes(total))
	}
	writeJSON(w, http.StatusOK, saveListResponse{
		SaveGames:    sgEntries,
		Config:       cfgEntries,
		TotalBytes:   total,
		Warning:      warn,
		SaveGamesRel: sgRel,
		ConfigRel:    cfgRel,
	})
}

func (s *Server) handleSavesDownload(w http.ResponseWriter, r *http.Request) {
	saveGames, config, err := s.resolveSaveRoots()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: err.Error()})
		return
	}
	includeConfig := r.URL.Query().Get("includeConfig") == "1" || r.URL.Query().Get("includeConfig") == "true"
	name := fmt.Sprintf("%s-%s.zip", saveArchivePrefix, time.Now().UTC().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	zw := zip.NewWriter(w)
	defer func() { _ = zw.Close() }()
	if err := addDirToZip(zw, saveGames, saveGamesDir); err != nil {
		return
	}
	if includeConfig {
		if err := addConfigDirToZip(zw, config, configLinuxDir); err != nil {
			return
		}
	}
}

func (s *Server) handleSavesUpload(w http.ResponseWriter, r *http.Request) {
	saveGames, config, err := s.resolveSaveRoots()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: err.Error()})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSaveUploadBytes)
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
	includeConfig := r.FormValue("includeConfig") == "1" || r.FormValue("includeConfig") == "true"
	if err := s.restoreSaveArchive(file, hdr.Filename, saveGames, config, includeConfig); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, actionResponse{
		Status:  "ok",
		Message: "World save restored. Restart the server so PalServer reloads the world.",
	})
}

func (s *Server) resolveSaveRoots() (saveGames, config string, err error) {
	if !s.savesConfigured() {
		return "", "", errSavesNotMounted
	}
	root, err := filepath.Abs(s.savesRoot)
	if err != nil {
		return "", "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", "", fmt.Errorf("saves root: %w", err)
	}
	officialSG := filepath.Join(root, saveGamesDir)
	nestedSG := filepath.Join(root, "Pal", "Saved", saveGamesDir)
	switch {
	case dirExists(officialSG):
		saveGames = officialSG
		config = filepath.Join(root, filepath.FromSlash(configLinuxDir))
	case dirExists(nestedSG):
		saveGames = nestedSG
		config = filepath.Join(root, "Pal", "Saved", filepath.FromSlash(configLinuxDir))
	default:
		saveGames = officialSG
		config = filepath.Join(root, filepath.FromSlash(configLinuxDir))
	}
	return saveGames, config, nil
}

func (s *Server) restoreSaveArchive(r io.Reader, filename, saveGames, config string, includeConfig bool) error {
	incomingParent := filepath.Join(filepath.Dir(saveGames), incomingDirName)
	_ = os.RemoveAll(incomingParent)
	if err := os.MkdirAll(incomingParent, 0o755); err != nil {
		return fmt.Errorf("prepare restore: %w", err)
	}
	defer func() { _ = os.RemoveAll(incomingParent) }()

	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		if err := extractZip(r, incomingParent); err != nil {
			return err
		}
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		if err := extractTarGz(r, incomingParent); err != nil {
			return err
		}
	case strings.HasSuffix(lower, ".tar"):
		if err := extractTar(r, incomingParent); err != nil {
			return err
		}
	default:
		return errors.New("unsupported archive (use .zip, .tar, or .tar.gz)")
	}

	srcSave, srcConfig := locateExtractedSaves(incomingParent)
	if srcSave == "" {
		return errors.New("archive has no SaveGames tree (or world folder)")
	}
	if err := replaceDir(saveGames, srcSave); err != nil {
		return err
	}
	if includeConfig && srcConfig != "" {
		if err := replaceDir(config, srcConfig); err != nil {
			return fmt.Errorf("restore Config: %w", err)
		}
	}
	return nil
}

func locateExtractedSaves(root string) (saveGames, config string) {
	candidates := []string{
		filepath.Join(root, saveGamesDir),
		filepath.Join(root, "Pal", "Saved", saveGamesDir),
	}
	for _, c := range candidates {
		if dirExists(c) {
			saveGames = c
			break
		}
	}
	if saveGames == "" {
		if entries, err := os.ReadDir(root); err == nil {
			for _, e := range entries {
				if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
					continue
				}
				child := filepath.Join(root, e.Name())
				if looksLikeWorldFolder(child) {
					synthetic := filepath.Join(root, saveGamesDir, "0")
					if err := os.MkdirAll(filepath.Dir(synthetic), 0o755); err == nil {
						if err := os.Rename(child, synthetic); err == nil {
							saveGames = filepath.Join(root, saveGamesDir)
						}
					}
					break
				}
			}
		}
	}
	cfgCandidates := []string{
		filepath.Join(root, filepath.FromSlash(configLinuxDir)),
		filepath.Join(root, "Pal", "Saved", filepath.FromSlash(configLinuxDir)),
	}
	for _, c := range cfgCandidates {
		if dirExists(c) {
			config = c
			break
		}
	}
	return saveGames, config
}

func looksLikeWorldFolder(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		name := strings.ToLower(e.Name())
		if strings.HasSuffix(name, ".sav") || name == "players" || name == "worldoption.sav" {
			return true
		}
	}
	return false
}

func replaceDir(dest, src string) error {
	if dest == "" || src == "" {
		return errors.New("missing restore path")
	}
	parent := filepath.Dir(dest)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	outgoing := filepath.Join(parent, outgoingDirName)
	_ = os.RemoveAll(outgoing)
	if dirExists(dest) {
		if err := os.Rename(dest, outgoing); err != nil {
			return fmt.Errorf("park current save: %w", err)
		}
	}
	if err := os.Rename(src, dest); err != nil {
		if dirExists(outgoing) {
			_ = os.Rename(outgoing, dest)
		}
		return fmt.Errorf("install restored save: %w", err)
	}
	_ = os.RemoveAll(outgoing)
	return nil
}

func extractZip(r io.Reader, dest string) error {
	tmp, err := os.CreateTemp("", "palworld-save-*.zip")
	if err != nil {
		return err
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()
	if _, err := io.Copy(tmp, r); err != nil {
		return fmt.Errorf("read zip: %w", err)
	}
	zr, err := zip.OpenReader(tmp.Name())
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer func() { _ = zr.Close() }()
	for _, f := range zr.File {
		if err := writeArchiveFile(dest, f.Name, f.FileInfo().IsDir(), func() (io.ReadCloser, error) {
			return f.Open()
		}); err != nil {
			return err
		}
	}
	return nil
}

func extractTarGz(r io.Reader, dest string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer func() { _ = gz.Close() }()
	return extractTar(gz, dest)
}

func extractTar(r io.Reader, dest string) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeDir {
			continue
		}
		isDir := hdr.Typeflag == tar.TypeDir || strings.HasSuffix(hdr.Name, "/")
		if err := writeArchiveFile(dest, hdr.Name, isDir, func() (io.ReadCloser, error) {
			return io.NopCloser(tr), nil
		}); err != nil {
			return err
		}
	}
}

func writeArchiveFile(dest, name string, isDir bool, open func() (io.ReadCloser, error)) error {
	rel, err := sanitizeArchivePath(name)
	if err != nil {
		return err
	}
	if rel == "" {
		return nil
	}
	abs, err := SafeJoin(dest, rel)
	if err != nil {
		return fmt.Errorf("archive path rejected: %w", err)
	}
	if isDir {
		return os.MkdirAll(abs, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	rc, err := open()
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()
	out, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	_, err = io.Copy(out, rc)
	return err
}

func sanitizeArchivePath(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.TrimPrefix(name, "./")
	name = strings.TrimPrefix(name, "/")
	if name == "" || name == "." {
		return "", nil
	}
	for _, prefix := range []string{"pal/saved/", "pal/package/pal/saved/"} {
		if strings.HasPrefix(strings.ToLower(name), prefix) {
			name = name[len(prefix):]
			break
		}
	}
	cleaned := pathCleanSlash(name)
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", errPathEscape
	}
	return cleaned, nil
}

func pathCleanSlash(p string) string {
	parts := strings.Split(p, "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			if len(out) == 0 {
				return ".."
			}
			out = out[:len(out)-1]
			continue
		}
		out = append(out, part)
	}
	return strings.Join(out, "/")
}

func addDirToZip(zw *zip.Writer, dir, prefix string) error {
	if !dirExists(dir) {
		return nil
	}
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name() == incomingDirName || d.Name() == outgoingDirName {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		name := prefix
		if rel != "." {
			name = prefix + "/" + filepath.ToSlash(rel)
		}
		if d.IsDir() {
			_, err := zw.Create(name + "/")
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		fh, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		fh.Name = name
		fh.Method = zip.Deflate
		w, err := zw.CreateHeader(fh)
		if err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(w, in)
		_ = in.Close()
		return copyErr
	})
}

func addConfigDirToZip(zw *zip.Writer, dir, prefix string) error {
	if !dirExists(dir) {
		return nil
	}
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		name := prefix
		if rel != "." {
			name = prefix + "/" + filepath.ToSlash(rel)
		}
		if d.IsDir() {
			_, err := zw.Create(name + "/")
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		fh, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		fh.Name = name
		fh.Method = zip.Deflate
		w, err := zw.CreateHeader(fh)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".ini") {
			raw = redactINISecrets(raw)
		}
		_, err = w.Write(raw)
		return err
	})
}

func redactINISecrets(raw []byte) []byte {
	return iniSecretAssign.ReplaceAll(raw, []byte(`${1}=`+iniRedactedValue))
}

func listDirEntries(abs, relPrefix string) ([]fileEntry, int64, error) {
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, 0, err
	}
	out := make([]fileEntry, 0, len(entries))
	var total int64
	for _, e := range entries {
		name := e.Name()
		if name == "." || name == ".." || name == incomingDirName || name == outgoingDirName {
			continue
		}
		info, infoErr := e.Info()
		size := int64(0)
		dir := e.IsDir()
		if infoErr == nil {
			dir = info.IsDir()
			if dir {
				size = dirSize(filepath.Join(abs, name))
			} else {
				size = info.Size()
			}
		}
		total += size
		child := name
		if relPrefix != "" {
			child = strings.TrimSuffix(relPrefix, "/") + "/" + name
		}
		out = append(out, fileEntry{Name: name, Path: child, Dir: dir, Size: size})
	}
	return out, total, nil
}

func dirSize(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, infoErr := d.Info()
		if infoErr == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func relToSaves(root, abs string) string {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return filepath.Base(abs)
	}
	rel, err := filepath.Rel(rootAbs, abs)
	if err != nil {
		return filepath.Base(abs)
	}
	return filepath.ToSlash(rel)
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for n := n / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
