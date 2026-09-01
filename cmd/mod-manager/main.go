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

	"github.com/DataKnifeAI/palworld-operator/internal/modmanager"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	listen := flag.String("listen", envOr("MOD_MANAGER_LISTEN", ":8088"), "listen address")
	root := flag.String("root", envOr("MOD_MANAGER_ROOT", "/mods"), "mods PVC mount root")
	user := flag.String("user", envOr("MOD_MANAGER_USER", modmanager.DefaultUser), "basic auth username")
	flag.Parse()

	password := os.Getenv("MOD_MANAGER_PASSWORD")
	if password == "" {
		log.Fatal("MOD_MANAGER_PASSWORD is required")
	}

	srv, err := modmanager.New(modmanager.Config{
		Root:     *root,
		User:     *user,
		Password: password,
		Restarter: &modmanager.DeploymentRestarter{
			Namespace: os.Getenv("MOD_MANAGER_NAMESPACE"),
			Name:      os.Getenv("MOD_MANAGER_DEPLOYMENT"),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("mod manager listening on %s root=%s", *listen, *root)
	if err := http.ListenAndServe(*listen, srv); err != nil {
		log.Fatal(err)
	}
}
