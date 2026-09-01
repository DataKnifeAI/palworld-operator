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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var (
	errPathEscape = errors.New("path escapes mods root")
	errEmptyName  = errors.New("empty file name")
)

// SafeJoin resolves rel against root and rejects path traversal and symlink
// escapes. rel is treated as relative to root; absolute paths are rejected.
func SafeJoin(root, rel string) (string, error) {
	if strings.ContainsRune(rel, 0) {
		return "", errPathEscape
	}
	if filepath.IsAbs(rel) {
		return "", errPathEscape
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("mods root: %w", err)
	}
	rootAbs, err = filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", fmt.Errorf("mods root: %w", err)
	}

	cleaned := filepath.Clean(rel)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", errPathEscape
	}

	if cleaned == "." {
		return rootAbs, nil
	}

	current := rootAbs
	for i, part := range strings.Split(cleaned, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			return "", errPathEscape
		}
		next := filepath.Join(current, part)
		if !within(rootAbs, next) {
			return "", errPathEscape
		}
		info, err := os.Lstat(next)
		if errors.Is(err, fs.ErrNotExist) {
			rest := strings.Split(cleaned, string(filepath.Separator))[i:]
			out := current
			for _, p := range rest {
				if p == "" || p == "." {
					continue
				}
				out = filepath.Join(out, p)
			}
			if !within(rootAbs, out) {
				return "", errPathEscape
			}
			return out, nil
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, evalErr := filepath.EvalSymlinks(next)
			if evalErr != nil {
				return "", errPathEscape
			}
			if !within(rootAbs, target) {
				return "", errPathEscape
			}
			current = target
			continue
		}
		current = next
	}
	return current, nil
}

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

func safeBaseName(name string) (string, error) {
	base := filepath.Base(name)
	if base == "." || base == string(filepath.Separator) || base == "" || base == ".." {
		return "", errEmptyName
	}
	if strings.Contains(base, "..") {
		return "", errPathEscape
	}
	return base, nil
}
