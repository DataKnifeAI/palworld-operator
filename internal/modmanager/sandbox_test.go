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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeJoinAllowsRelativePaths(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "paks", "~WorkshopMods"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "PalModSettings.ini"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []string{
		"",
		".",
		"PalModSettings.ini",
		"Workshop/MyMod/Info.json",
		"paks/~WorkshopMods/foo.pak",
		"paks/LogicMods",
		"paks/~WorkshopMods/../LogicMods/bar.pak",
	}
	for _, rel := range cases {
		got, err := SafeJoin(root, rel)
		if err != nil {
			t.Fatalf("SafeJoin(%q) unexpected error: %v", rel, err)
		}
		if !within(root, got) && got != root {
			// EvalSymlinks may change root; re-resolve
			rootAbs, _ := filepath.EvalSymlinks(root)
			if !within(rootAbs, got) {
				t.Fatalf("SafeJoin(%q) = %q escapes %q", rel, got, root)
			}
		}
	}
}

func TestSafeJoinRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	_ = os.WriteFile(filepath.Join(outside, "secret"), []byte("nope"), 0o600)

	cases := []string{
		"/etc/passwd",
		"../etc/passwd",
		"..",
		"foo/../../../etc/passwd",
		"Workshop/foo/../../../etc/passwd",
		filepath.Join("..", filepath.Base(outside), "secret"),
		"\x00evil",
	}
	for _, rel := range cases {
		got, err := SafeJoin(root, rel)
		if err == nil {
			t.Fatalf("SafeJoin(%q) = %q, want error", rel, got)
		}
	}
}

func TestSafeJoinRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret")
	if err := os.WriteFile(secret, []byte("nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escape")
	if err := os.Symlink(secret, link); err != nil {
		t.Fatal(err)
	}

	got, err := SafeJoin(root, "escape")
	if err == nil {
		t.Fatalf("symlink escape allowed: %q", got)
	}
	if !strings.Contains(err.Error(), "escapes") && err != errPathEscape {
		t.Fatalf("expected escape error, got %v", err)
	}

	if err := os.Mkdir(filepath.Join(root, "okdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	dirLink := filepath.Join(root, "outsidelink")
	if err := os.Symlink(outside, dirLink); err != nil {
		t.Fatal(err)
	}
	got, err = SafeJoin(root, "outsidelink/secret")
	if err == nil {
		t.Fatalf("dir symlink escape allowed: %q", got)
	}
}

func TestSafeJoinNewFileUnderRoot(t *testing.T) {
	root := t.TempDir()
	got, err := SafeJoin(root, "paks/~WorkshopMods/new.pak")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "paks", "~WorkshopMods", "new.pak")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSafeBaseName(t *testing.T) {
	got, err := safeBaseName("foo.pak")
	if err != nil || got != "foo.pak" {
		t.Fatalf("got %q %v", got, err)
	}
	got, err = safeBaseName("/tmp/foo.pak")
	if err != nil || got != "foo.pak" {
		t.Fatalf("base of abs = %q %v", got, err)
	}
	if _, err := safeBaseName(".."); err == nil {
		t.Fatal("expected error for ..")
	}
	if _, err := safeBaseName(""); err == nil {
		t.Fatal("expected error for empty")
	}
}
