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

package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/DataKnifeAI/palworld-operator/internal/modmanager"
)

func envOr(keys []string, fallback string) string {
	for _, key := range keys {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return fallback
}

func main() {
	listen := flag.String("listen", envOr([]string{"SERVER_MANAGER_LISTEN", "MOD_MANAGER_LISTEN"}, ":8088"), "listen address")
	root := flag.String("root", envOr([]string{"SERVER_MANAGER_ROOT", "MOD_MANAGER_ROOT"}, ""), "mods PVC mount root (empty disables Mods tab files)")
	saves := flag.String("saves", envOr([]string{"SERVER_MANAGER_SAVES"}, "/saves"), "world saves PVC mount (Pal/Saved)")
	user := flag.String("user", envOr([]string{"SERVER_MANAGER_USER", "MOD_MANAGER_USER"}, modmanager.DefaultUser), "basic auth username")
	restBase := flag.String("rest-base", envOr([]string{"SERVER_MANAGER_REST_BASE"}, "http://127.0.0.1:8212"), "Palworld REST base (sidecar localhost)")
	flag.Parse()

	password := envOr([]string{"SERVER_MANAGER_PASSWORD", "MOD_MANAGER_PASSWORD"}, "")
	if password == "" {
		log.Fatal("SERVER_MANAGER_PASSWORD (or MOD_MANAGER_PASSWORD) is required")
	}

	namespace := envOr([]string{"SERVER_MANAGER_NAMESPACE", "MOD_MANAGER_NAMESPACE"}, "")
	deployment := envOr([]string{"SERVER_MANAGER_DEPLOYMENT", "MOD_MANAGER_DEPLOYMENT"}, "")

	srv, err := modmanager.New(modmanager.Config{
		Root:      *root,
		SavesRoot: *saves,
		User:      *user,
		Password:  password,
		RESTBase:  *restBase,
		Restarter: &modmanager.DeploymentRestarter{
			Namespace: namespace,
			Name:      deployment,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("server manager listening on %s mods=%s saves=%s rest=%s", *listen, *root, *saves, *restBase)
	httpSrv := &http.Server{
		Addr:              *listen,
		Handler:           srv,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	if err := httpSrv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
