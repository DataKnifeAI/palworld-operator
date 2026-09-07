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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	palworldv1alpha1 "github.com/DataKnifeAI/palworld-operator/api/v1alpha1"
	"github.com/DataKnifeAI/palworld-operator/internal/controller"
	"k8s.io/utils/ptr"
)

const (
	errUpdatesDisabled     = "update API is not configured"
	errNoUpdateAvailable   = "no update available"
	forceCountdownDefault  = 10 * time.Second
	notifyPlaceholderOpen  = "{"
	maxUpdateSettingsBytes = 1 << 16
)

var forceCountdown = forceCountdownDefault

type updateSettings struct {
	AutoUpdateImage  bool   `json:"autoUpdateImage"`
	CheckInterval    string `json:"checkInterval"`
	CheckSchedule    string `json:"checkSchedule"`
	ApplySchedule    string `json:"applySchedule"`
	TimeZone         string `json:"timeZone"`
	ImageRepository  string `json:"imageRepository"`
	OnlyWhenEmpty    *bool  `json:"onlyWhenEmpty"`
	NotifyPlayers    bool   `json:"notifyPlayers"`
	NotifySchedule   string `json:"notifySchedule"`
	NotifyMessage    string `json:"notifyMessage"`
	NotifyMessageTip string `json:"notifyMessageHint,omitempty"`
}

type updatesResponse struct {
	Pinned          string         `json:"pinned"`
	Latest          string         `json:"latest"`
	LatestImage     string         `json:"latestImage"`
	UpdateAvailable bool           `json:"updateAvailable"`
	Settings        updateSettings `json:"settings"`
}

type settingsError struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}

func (s *Server) updatesConfigured() bool {
	return s.cr != nil
}

func (s *Server) tagLister() controller.TagLister {
	if s.tags != nil {
		return s.tags
	}
	return &controller.GHCRTagLister{}
}

func (s *Server) handleUpdatesGet(w http.ResponseWriter, r *http.Request) {
	s.writeUpdates(w, r.Context(), false)
}

func (s *Server) handleUpdatesCheck(w http.ResponseWriter, r *http.Request) {
	s.writeUpdates(w, r.Context(), true)
}

func (s *Server) writeUpdates(w http.ResponseWriter, ctx context.Context, fresh bool) {
	if !s.updatesConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errUpdatesDisabled})
		return
	}
	server, err := s.cr.Get(ctx)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	out, err := s.buildUpdates(ctx, server, fresh)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleUpdatesSave(w http.ResponseWriter, r *http.Request) {
	if !s.updatesConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errUpdatesDisabled})
		return
	}
	var req updateSettings
	if err := json.NewDecoder(io.LimitReader(r.Body, maxUpdateSettingsBytes)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	cfg, fields := validateAndBuildUpdate(req)
	if len(fields) > 0 {
		writeJSON(w, http.StatusBadRequest, settingsError{Error: "Not saved.", Fields: fields})
		return
	}
	ctx := r.Context()
	server, err := s.cr.Get(ctx)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	server.Spec.Update = cfg
	if err := s.cr.Update(ctx, server); err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	out, err := s.buildUpdates(ctx, server, false)
	if err != nil {
		writeJSON(w, http.StatusOK, actionResponse{Status: "ok", Message: "Saved."})
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleUpdatesReset(w http.ResponseWriter, r *http.Request) {
	if !s.updatesConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errUpdatesDisabled})
		return
	}
	ctx := r.Context()
	server, err := s.cr.Get(ctx)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	server.Spec.Update = palworldv1alpha1.UpdateConfig{}
	if err := s.cr.Update(ctx, server); err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	out, err := s.buildUpdates(ctx, server, false)
	if err != nil {
		writeJSON(w, http.StatusOK, actionResponse{Status: "ok", Message: "Defaults restored."})
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleUpdatesForce(w http.ResponseWriter, r *http.Request) {
	if !s.updatesConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errUpdatesDisabled})
		return
	}
	ctx := r.Context()
	server, err := s.cr.Get(ctx)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	out, err := s.buildUpdates(ctx, server, true)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	if !out.UpdateAvailable || out.LatestImage == "" {
		writeJSON(w, http.StatusConflict, errorResponse{Error: errNoUpdateAvailable})
		return
	}
	if !s.restConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errRESTDisabled})
		return
	}
	startMsg := forceAnnounceText(server.Spec, out.Latest, out.LatestImage, forceCountdown)
	if err := s.restPost(ctx, "/v1/api/announce", map[string]string{"message": startMsg}); err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	wait := forceCountdown
	if wait > 0 {
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			writeJSON(w, http.StatusRequestTimeout, errorResponse{Error: "force update canceled"})
			return
		case <-timer.C:
		}
	}
	nowMsg := forceAnnounceText(server.Spec, out.Latest, out.LatestImage, 0)
	if err := s.restPost(ctx, "/v1/api/announce", map[string]string{"message": nowMsg}); err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	server, err = s.cr.Get(ctx)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	server.Spec.ServerImage = out.LatestImage
	if err := s.cr.Update(ctx, server); err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, actionResponse{Status: "ok", Message: "Updated.", Image: out.LatestImage})
}

func (s *Server) buildUpdates(ctx context.Context, server *palworldv1alpha1.PalworldServer, fresh bool) (updatesResponse, error) {
	pinned := controller.ServerImage(server.Spec)
	repo := controller.ImageRepository(server.Spec)
	latest := strings.TrimSpace(server.Status.LatestAvailableVersion)
	if fresh || latest == "" {
		var tags []string
		var err error
		if lister, ok := s.tagLister().(interface {
			ListTagsFresh(context.Context, string) ([]string, error)
		}); ok && fresh {
			tags, err = lister.ListTagsFresh(ctx, repo)
		} else {
			tags, err = s.tagLister().ListTags(ctx, repo)
		}
		if err != nil {
			if latest == "" {
				return updatesResponse{}, err
			}
		} else if tag, ok := controller.NewestPalVersionTag(tags); ok {
			latest = tag
		}
	}
	latestImage := ""
	if latest != "" {
		latestImage = controller.FormatImageRef(repo, latest)
	}
	settings := settingsFromSpec(server.Spec.Update)
	settings.NotifyMessageTip = notifyMessageHint(settings.NotifyMessage)
	return updatesResponse{
		Pinned:          pinned,
		Latest:          latest,
		LatestImage:     latestImage,
		UpdateAvailable: controller.ShouldUpdateImage(pinned, server.Status.RunningVersion, latest),
		Settings:        settings,
	}, nil
}

func settingsFromSpec(cfg palworldv1alpha1.UpdateConfig) updateSettings {
	sched := strings.Join(cfg.NotifySchedule, ", ")
	return updateSettings{
		AutoUpdateImage: cfg.AutoUpdateImage,
		CheckInterval:   cfg.CheckInterval,
		CheckSchedule:   cfg.CheckSchedule,
		ApplySchedule:   cfg.ApplySchedule,
		TimeZone:        cfg.TimeZone,
		ImageRepository: cfg.ImageRepository,
		OnlyWhenEmpty:   cfg.OnlyWhenEmpty,
		NotifyPlayers:   cfg.NotifyPlayers,
		NotifySchedule:  sched,
		NotifyMessage:   cfg.NotifyMessage,
	}
}

func validateAndBuildUpdate(in updateSettings) (palworldv1alpha1.UpdateConfig, map[string]string) {
	fields := map[string]string{}
	in.CheckInterval = strings.TrimSpace(in.CheckInterval)
	in.CheckSchedule = strings.TrimSpace(in.CheckSchedule)
	in.ApplySchedule = strings.TrimSpace(in.ApplySchedule)
	in.TimeZone = strings.TrimSpace(in.TimeZone)
	in.ImageRepository = strings.TrimSpace(in.ImageRepository)
	in.NotifySchedule = strings.TrimSpace(in.NotifySchedule)
	in.NotifyMessage = strings.TrimSpace(in.NotifyMessage)

	if in.CheckInterval != "" {
		d, err := time.ParseDuration(in.CheckInterval)
		if err != nil || d <= 0 {
			fields["checkInterval"] = "Need a Go duration (6h, 1h30m)"
		}
	}
	if in.CheckSchedule != "" {
		if err := controller.ValidateCronExpr(in.CheckSchedule); err != nil {
			fields["checkSchedule"] = "Need 5-field cron or @hourly"
		}
	}
	if in.ApplySchedule != "" {
		if err := controller.ValidateCronExpr(in.ApplySchedule); err != nil {
			fields["applySchedule"] = "Need 5-field cron or @hourly"
		}
	}
	if in.TimeZone != "" {
		if _, err := time.LoadLocation(in.TimeZone); err != nil {
			fields["timeZone"] = "Need IANA zone (UTC, America/Los_Angeles)"
		}
	}
	if in.ImageRepository != "" {
		if err := controller.ValidateImageRepository(in.ImageRepository); err != nil {
			fields["imageRepository"] = "Need host/path"
		}
	}
	var notifyKeys []string
	if in.NotifySchedule != "" {
		for _, tok := range strings.Split(in.NotifySchedule, ",") {
			tok = strings.TrimSpace(tok)
			if tok == "" {
				fields["notifySchedule"] = "Each item needs a Go duration"
				break
			}
			d, err := time.ParseDuration(tok)
			if err != nil || d <= 0 {
				fields["notifySchedule"] = "Each item needs a Go duration"
				break
			}
			notifyKeys = append(notifyKeys, tok)
		}
	}

	if len(fields) > 0 {
		return palworldv1alpha1.UpdateConfig{}, fields
	}
	out := palworldv1alpha1.UpdateConfig{
		AutoUpdateImage: in.AutoUpdateImage,
		CheckInterval:   in.CheckInterval,
		CheckSchedule:   in.CheckSchedule,
		ApplySchedule:   in.ApplySchedule,
		TimeZone:        in.TimeZone,
		ImageRepository: in.ImageRepository,
		NotifyPlayers:   in.NotifyPlayers,
		NotifySchedule:  notifyKeys,
		NotifyMessage:   in.NotifyMessage,
	}
	if in.OnlyWhenEmpty != nil && !*in.OnlyWhenEmpty {
		out.OnlyWhenEmpty = ptr.To(false)
	}
	return out, nil
}

func notifyMessageHint(msg string) string {
	if !strings.Contains(msg, notifyPlaceholderOpen) {
		return ""
	}
	known := map[string]struct{}{"version": {}, "image": {}, "remaining": {}}
	rest := msg
	for {
		i := strings.Index(rest, "{")
		if i < 0 {
			return ""
		}
		rest = rest[i+1:]
		j := strings.Index(rest, "}")
		if j < 0 {
			return "Unclosed {"
		}
		tok := rest[:j]
		if _, ok := known[tok]; !ok {
			return "Unknown {" + tok + "}"
		}
		rest = rest[j+1:]
	}
}

func forceAnnounceText(spec palworldv1alpha1.PalworldServerSpec, version, image string, remaining time.Duration) string {
	remain := "now"
	if remaining > 0 {
		remain = remaining.String()
	}
	tmpl := strings.TrimSpace(spec.Update.NotifyMessage)
	if tmpl == "" {
		if remaining > 0 {
			return fmt.Sprintf("[Server] Update %s — restart in %s", version, remain)
		}
		return fmt.Sprintf("[Server] Update %s — restarting now", version)
	}
	out := tmpl
	out = strings.ReplaceAll(out, "{version}", version)
	out = strings.ReplaceAll(out, "{image}", image)
	out = strings.ReplaceAll(out, "{remaining}", remain)
	return out
}
