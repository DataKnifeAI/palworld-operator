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
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	palworldv1alpha1 "github.com/DataKnifeAI/palworld-operator/api/v1alpha1"
)

const (
	errSecretsDisabled      = "credentials rotate is not configured"
	secretKeyJoin           = "server-password"
	secretKeyAdmin          = "admin-password"
	credKindJoin            = "join"
	credKindAdmin           = "admin"
	credentialsSecretSuffix = "-secrets"
	generatedPasswordBytes  = 24
	maxRotateBodyBytes      = 4096
)

type rotateRequest struct {
	Key string `json:"key"`
}

type rotateResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Key      string `json:"key"`
	Password string `json:"password"`
}

func (s *Server) handleCredentialsRotate(w http.ResponseWriter, r *http.Request) {
	if s.secrets == nil || !s.updatesConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errSecretsDisabled})
		return
	}
	var req rotateRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxRotateBodyBytes)).Decode(&req); err != nil && err != io.EOF {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errInvalidJSON})
		return
	}
	kind := strings.ToLower(strings.TrimSpace(req.Key))
	if kind != credKindJoin && kind != credKindAdmin {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "key must be join or admin"})
		return
	}
	ctx := r.Context()
	server, err := s.cr.Get(ctx)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}
	name, key := credentialTarget(server, kind)
	if name == "" || key == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "credential Secret is not configured"})
		return
	}
	secret, err := s.secrets.GetSecret(ctx, name)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: "secret read failed"})
		return
	}
	password, err := generatePassword()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "rotate failed"})
		return
	}
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}
	secret.Data[key] = []byte(password)
	if err := s.secrets.UpdateSecret(ctx, secret); err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: "secret update failed"})
		return
	}
	log.Printf("server manager rotated %s password", kind)
	if s.restarter == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errRestartNotConfigured})
		return
	}
	if err := s.restarter.Restart(ctx); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: errRestartFailed})
		return
	}
	writeJSON(w, http.StatusOK, rotateResponse{
		Status:   "ok",
		Message:  "Rotated " + kind + ". Recreate requested. Copy once if needed.",
		Key:      kind,
		Password: password,
	})
}

func (s *Server) credStatus(ctx context.Context, server *palworldv1alpha1.PalworldServer) credStatus {
	joinSet, adminSet := secretKeyPresent(ctx, s.secrets, server, credKindJoin), secretKeyPresent(ctx, s.secrets, server, credKindAdmin)
	return credStatus{
		Join:  credFlag{Set: joinSet},
		Admin: credFlag{Set: adminSet},
	}
}

func secretKeyPresent(ctx context.Context, store SecretStore, server *palworldv1alpha1.PalworldServer, kind string) bool {
	if store == nil || server == nil {
		return false
	}
	name, key := credentialTarget(server, kind)
	if name == "" || key == "" {
		return false
	}
	secret, err := store.GetSecret(ctx, name)
	if err != nil || secret == nil {
		return false
	}
	return len(secret.Data[key]) > 0
}

func credentialTarget(server *palworldv1alpha1.PalworldServer, kind string) (name, key string) {
	if server == nil {
		return "", ""
	}
	switch kind {
	case credKindJoin:
		if ref := server.Spec.ServerPasswordSecretRef; ref != nil && ref.Name != "" && ref.Key != "" {
			return ref.Name, ref.Key
		}
		return credentialsSecretName(server), secretKeyJoin
	case credKindAdmin:
		if ref := server.Spec.AdminPasswordSecretRef; ref != nil && ref.Name != "" && ref.Key != "" {
			return ref.Name, ref.Key
		}
		return credentialsSecretName(server), secretKeyAdmin
	default:
		return "", ""
	}
}

func credentialsSecretName(server *palworldv1alpha1.PalworldServer) string {
	if server.Spec.CredentialsSecretName != "" {
		return server.Spec.CredentialsSecretName
	}
	return server.Name + credentialsSecretSuffix
}

func generatePassword() (string, error) {
	buf := make([]byte, generatedPasswordBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
