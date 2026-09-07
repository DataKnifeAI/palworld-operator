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
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	palworldv1alpha1 "github.com/DataKnifeAI/palworld-operator/api/v1alpha1"
	"k8s.io/utils/ptr"
)

const (
	errSettingsDisabled  = "settings API is not configured"
	defaultMaxPlayersAPI = 4
	defaultCrossplayAPI  = "(Steam,Xbox,PS5,Mac)"
	maxPlayersLimit      = 32
	liveRedacted         = "[redacted]"
	settingsININame      = "PalWorldSettings.ini"
	maxSettingsBodyBytes = 1 << 16
	platformSteam        = "Steam"
	platformXbox         = "Xbox"
	platformPS5          = "PS5"
	platformMac          = "Mac"
)

var (
	reservedOptionKeys = map[string]struct{}{
		"ServerName": {}, "ServerDescription": {}, "ServerPlayerMaxNum": {},
		"AdminPassword": {}, "ServerPassword": {}, "PublicPort": {}, "PublicIP": {},
		"RCONEnabled": {}, "RCONPort": {}, "RESTAPIEnabled": {}, "RESTAPIPort": {},
		"CrossplayPlatforms": {},
	}
	secretLiveKeys = map[string]struct{}{
		"ServerPassword": {}, "AdminPassword": {},
	}
	deathPenaltyVals   = map[string]struct{}{"None": {}, "Item": {}, "ItemAndEquipment": {}, "All": {}}
	randomizerVals     = map[string]struct{}{"None": {}, "Region": {}, "All": {}}
	boolOptionVals     = map[string]struct{}{"True": {}, "False": {}}
	optNumberRE        = regexp.MustCompile(`^-?\d+(\.\d+)?$`)
	optionSettingsLine = regexp.MustCompile(`(?i)OptionSettings\s*=\s*\((.*)\)\s*$`)
)

type serverProfile struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	MaxPlayers  int32  `json:"maxPlayers"`
	Steam       bool   `json:"steam"`
	Xbox        bool   `json:"xbox"`
	PS5         bool   `json:"ps5"`
	Mac         bool   `json:"mac"`
	Community   bool   `json:"community"`
}

type settingsResponse struct {
	Profile     serverProfile     `json:"profile"`
	Options     map[string]string `json:"options"`
	Live        map[string]string `json:"live"`
	Credentials credStatus        `json:"credentials"`
}

type settingsApplyRequest struct {
	Profile *serverProfile    `json:"profile"`
	Options map[string]string `json:"options"`
}

type credStatus struct {
	Join  credFlag `json:"join"`
	Admin credFlag `json:"admin"`
}

type credFlag struct {
	Set bool `json:"set"`
}

func (s *Server) handleSettingsGet(w http.ResponseWriter, r *http.Request) {
	if !s.updatesConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errSettingsDisabled})
		return
	}
	ctx := r.Context()
	server, err := s.cr.Get(ctx)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, s.buildSettings(ctx, server))
}

func (s *Server) handleSettingsApply(w http.ResponseWriter, r *http.Request) {
	if !s.updatesConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errSettingsDisabled})
		return
	}
	var req settingsApplyRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxSettingsBodyBytes)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errInvalidJSON})
		return
	}
	if req.Profile == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "profile is required"})
		return
	}
	if fields := validateProfile(*req.Profile); len(fields) > 0 {
		writeJSON(w, http.StatusBadRequest, settingsError{Error: "Not applied.", Fields: fields})
		return
	}
	opts, fields := sanitizeOptions(req.Options)
	if len(fields) > 0 {
		writeJSON(w, http.StatusBadRequest, settingsError{Error: "Not applied.", Fields: fields})
		return
	}
	ctx := r.Context()
	server, err := s.cr.Get(ctx)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	applyProfileToSpec(&server.Spec, *req.Profile)
	server.Spec.OptionSettings = opts
	if err := s.cr.Update(ctx, server); err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	if s.restarter == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "restart is not configured"})
		return
	}
	if err := s.restarter.Restart(ctx); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "restart failed"})
		return
	}
	writeJSON(w, http.StatusOK, actionResponse{
		Status:  "ok",
		Message: "Profile and game settings applied. Recreate requested.",
	})
}

func (s *Server) buildSettings(ctx context.Context, server *palworldv1alpha1.PalworldServer) settingsResponse {
	opts := map[string]string{}
	for k, v := range server.Spec.OptionSettings {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if _, reserved := reservedOptionKeys[k]; reserved {
			continue
		}
		opts[k] = v
	}
	return settingsResponse{
		Profile:     profileFromSpec(server.Spec),
		Options:     opts,
		Live:        s.liveSettings(ctx),
		Credentials: s.credStatus(ctx, server),
	}
}

func profileFromSpec(spec palworldv1alpha1.PalworldServerSpec) serverProfile {
	max := spec.MaxPlayers
	if max <= 0 {
		max = defaultMaxPlayersAPI
	}
	cross := spec.CrossplayPlatforms
	if strings.TrimSpace(cross) == "" {
		cross = defaultCrossplayAPI
	}
	steam, xbox, ps5, mac := parseCrossplay(cross)
	community := false
	if spec.Community.Enabled != nil {
		community = *spec.Community.Enabled
	}
	return serverProfile{
		Name:        spec.ServerName,
		Description: spec.ServerDescription,
		MaxPlayers:  max,
		Steam:       steam,
		Xbox:        xbox,
		PS5:         ps5,
		Mac:         mac,
		Community:   community,
	}
}

func applyProfileToSpec(spec *palworldv1alpha1.PalworldServerSpec, p serverProfile) {
	spec.ServerName = strings.TrimSpace(p.Name)
	spec.ServerDescription = strings.TrimSpace(p.Description)
	spec.MaxPlayers = p.MaxPlayers
	spec.CrossplayPlatforms = formatCrossplay(p)
	spec.Community.Enabled = ptr.To(p.Community)
}

func parseCrossplay(raw string) (steam, xbox, ps5, mac bool) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "(")
	s = strings.TrimSuffix(s, ")")
	if s == "" {
		return true, true, true, true
	}
	for _, part := range strings.Split(s, ",") {
		switch strings.TrimSpace(part) {
		case platformSteam:
			steam = true
		case platformXbox:
			xbox = true
		case platformPS5:
			ps5 = true
		case platformMac:
			mac = true
		}
	}
	return steam, xbox, ps5, mac
}

func formatCrossplay(p serverProfile) string {
	parts := make([]string, 0, 4)
	if p.Steam {
		parts = append(parts, platformSteam)
	}
	if p.Xbox {
		parts = append(parts, platformXbox)
	}
	if p.PS5 {
		parts = append(parts, platformPS5)
	}
	if p.Mac {
		parts = append(parts, platformMac)
	}
	if len(parts) == 0 {
		return ""
	}
	return "(" + strings.Join(parts, ",") + ")"
}

func validateProfile(p serverProfile) map[string]string {
	if p.MaxPlayers < 1 || p.MaxPlayers > maxPlayersLimit {
		return map[string]string{"maxPlayers": "1–32"}
	}
	return nil
}

func sanitizeOptions(in map[string]string) (map[string]string, map[string]string) {
	fields := map[string]string{}
	out := map[string]string{}
	for key, raw := range in {
		key = strings.TrimSpace(key)
		val := strings.TrimSpace(raw)
		if key == "" || val == "" {
			continue
		}
		if _, reserved := reservedOptionKeys[key]; reserved {
			continue
		}
		if msg := validateOptionValue(key, val); msg != "" {
			fields[key] = msg
			continue
		}
		out[key] = val
	}
	if len(fields) > 0 {
		return nil, fields
	}
	return out, nil
}

func validateOptionValue(key, val string) string {
	switch key {
	case "DeathPenalty":
		if _, ok := deathPenaltyVals[val]; !ok {
			return "None, Item, ItemAndEquipment, or All"
		}
	case "RandomizerType":
		if _, ok := randomizerVals[val]; !ok {
			return "None, Region, or All"
		}
	default:
		if strings.HasPrefix(key, "b") && len(key) > 1 {
			if _, ok := boolOptionVals[val]; !ok {
				return "True or False"
			}
			return ""
		}
		if optNumberRE.MatchString(val) {
			n, err := strconv.ParseFloat(val, 64)
			if err != nil || n < 0 {
				return "number ≥ 0"
			}
			return ""
		}
	}
	return ""
}

func (s *Server) liveSettings(ctx context.Context) map[string]string {
	if s.restConfigured() {
		raw, err := s.restGet(ctx, "/v1/api/settings")
		if err == nil {
			if parsed := parseLiveSettings(raw); len(parsed) > 0 {
				return redactLive(parsed)
			}
		}
	}
	if parsed := s.liveSettingsFromINI(); len(parsed) > 0 {
		return redactLive(parsed)
	}
	return map[string]string{}
}

func parseLiveSettings(raw json.RawMessage) map[string]string {
	out := map[string]string{}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return out
	}
	if nested, ok := obj["settings"]; ok {
		switch v := nested.(type) {
		case string:
			return parseOptionSettingsTuple(v)
		case map[string]any:
			obj = v
		}
	}
	for k, v := range obj {
		if k == "settings" {
			continue
		}
		out[k] = stringifyLive(v)
	}
	if blob, ok := out["OptionSettings"]; ok && strings.Contains(blob, "=") {
		return parseOptionSettingsTuple(blob)
	}
	return out
}

func stringifyLive(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "True"
		}
		return "False"
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return t.String()
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

func parseOptionSettingsTuple(raw string) map[string]string {
	s := strings.TrimSpace(raw)
	if i := strings.Index(s, "OptionSettings="); i >= 0 {
		s = s[i+len("OptionSettings="):]
	}
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		s = s[1 : len(s)-1]
	}
	out := map[string]string{}
	var key strings.Builder
	var val strings.Builder
	inKey := true
	quote := byte(0)
	depth := 0
	flush := func() {
		k := strings.TrimSpace(key.String())
		v := strings.TrimSpace(val.String())
		if k != "" {
			if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
				v = v[1 : len(v)-1]
			}
			out[k] = v
		}
		key.Reset()
		val.Reset()
		inKey = true
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if inKey {
				key.WriteByte(c)
			} else {
				val.WriteByte(c)
			}
			if c == quote && (i == 0 || s[i-1] != '\\') {
				quote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
			if inKey {
				key.WriteByte(c)
			} else {
				val.WriteByte(c)
			}
			continue
		}
		if c == '(' {
			depth++
			if !inKey {
				val.WriteByte(c)
			}
			continue
		}
		if c == ')' && depth > 0 {
			depth--
			if !inKey {
				val.WriteByte(c)
			}
			continue
		}
		if c == '=' && inKey {
			inKey = false
			continue
		}
		if c == ',' && depth == 0 {
			flush()
			continue
		}
		if inKey {
			key.WriteByte(c)
		} else {
			val.WriteByte(c)
		}
	}
	flush()
	return out
}

func redactLive(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		if _, secret := secretLiveKeys[k]; secret {
			out[k] = liveRedacted
			continue
		}
		out[k] = v
	}
	return out
}

func (s *Server) liveSettingsFromINI() map[string]string {
	_, config, err := s.resolveSaveRoots()
	if err != nil {
		return nil
	}
	raw, err := os.ReadFile(filepath.Join(config, settingsININame))
	if err != nil {
		return nil
	}
	for _, line := range strings.Split(string(raw), "\n") {
		m := optionSettingsLine.FindStringSubmatch(strings.TrimSpace(line))
		if len(m) == 2 {
			return parseOptionSettingsTuple("(" + m[1] + ")")
		}
	}
	return nil
}
