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
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
)

const errNotPak = "only .pak files are supported"

type diskUsage struct {
	Used  int64 `json:"used"`
	Free  int64 `json:"free"`
	Total int64 `json:"total"`
}

func isPakName(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".pak")
}

func diskUsageOf(path string) (diskUsage, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return diskUsage{}, err
	}
	bsize := uint64(st.Bsize)
	total := int64(st.Blocks * bsize)
	free := int64(st.Bavail * bsize)
	used := int64((st.Blocks - st.Bfree) * bsize)
	return diskUsage{Used: used, Free: free, Total: total}, nil
}

func humanBytes(n int64) string {
	if n < 0 {
		n = 0
	}
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	case n < 1024*1024*1024:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	default:
		return fmt.Sprintf("%.1f GB", float64(n)/(1024*1024*1024))
	}
}

func spaceError(free int64) string {
	return "file is larger than " + humanBytes(free) + " free on the mods PVC"
}
