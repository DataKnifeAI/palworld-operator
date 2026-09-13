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
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	rebootCountdownDefault = 10 * time.Second
	rebootAnnounceBase     = "Server changes applied and server rebooting"
	errAnnounceCanceled    = "announce countdown canceled"
)

var rebootCountdown = rebootCountdownDefault

func rebootAnnounceText(remaining time.Duration) string {
	if remaining > 0 {
		return fmt.Sprintf("%s — restart in %s", rebootAnnounceBase, remaining)
	}
	return rebootAnnounceBase + " — restarting now"
}

// announceRebootCountdown sends the generic REST announce, waits 10s, then
// announces again at 0. Used by every portal Recreate except Shutdown and
// spec.update auto-notify. Writes the HTTP error and returns false on failure.
func (s *Server) announceRebootCountdown(w http.ResponseWriter, ctx context.Context) bool {
	if !s.restConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errRESTDisabled})
		return false
	}
	startMsg := rebootAnnounceText(rebootCountdown)
	if err := s.restPost(ctx, "/v1/api/announce", map[string]string{restMessageField: startMsg}); err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return false
	}
	if rebootCountdown > 0 {
		timer := time.NewTimer(rebootCountdown)
		select {
		case <-ctx.Done():
			timer.Stop()
			writeJSON(w, http.StatusRequestTimeout, errorResponse{Error: errAnnounceCanceled})
			return false
		case <-timer.C:
		}
	}
	nowMsg := rebootAnnounceText(0)
	if err := s.restPost(ctx, "/v1/api/announce", map[string]string{restMessageField: nowMsg}); err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return false
	}
	return true
}
