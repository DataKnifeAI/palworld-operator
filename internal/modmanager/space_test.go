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

import "testing"

func TestIsPakName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		want bool
	}{
		{"IslandKeep.pak", true},
		{"IslandKeep.PAK", true},
		{"notes.txt", false},
		{"mod.zip", false},
		{"mod.pak.bak", false},
		{"", false},
	} {
		if got := isPakName(tc.name); got != tc.want {
			t.Fatalf("isPakName(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestHumanBytes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		n    int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1536, "1.5 KB"},
		{42*1024*1024 + 512*1024, "42.5 MB"},
		{2 * 1024 * 1024 * 1024, "2.0 GB"},
	} {
		if got := humanBytes(tc.n); got != tc.want {
			t.Fatalf("humanBytes(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

func TestSpaceError(t *testing.T) {
	t.Parallel()
	got := spaceError(7 * 1024 * 1024 * 1024)
	if got != "file is larger than 7.0 GB free on the mods PVC" {
		t.Fatalf("spaceError = %q", got)
	}
}
