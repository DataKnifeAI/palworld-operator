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

	palworldv1alpha1 "github.com/DataKnifeAI/palworld-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ServerCR reads and updates the PalworldServer for this sidecar.
type ServerCR interface {
	Get(ctx context.Context) (*palworldv1alpha1.PalworldServer, error)
	Update(ctx context.Context, server *palworldv1alpha1.PalworldServer) error
}

// SecretStore reads and writes one namespaced Secret (rotate one key only).
type SecretStore interface {
	GetSecret(ctx context.Context, name string) (*corev1.Secret, error)
	UpdateSecret(ctx context.Context, secret *corev1.Secret) error
}

// RuntimeCR uses in-cluster config to get/update one PalworldServer.
type RuntimeCR struct {
	client    client.Client
	namespace string
	name      string
}

// NewRuntimeCR builds an in-cluster client for namespace/name.
func NewRuntimeCR(namespace, name string) (*RuntimeCR, error) {
	if namespace == "" || name == "" {
		return nil, fmt.Errorf("namespace and CR name are required")
	}
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("in-cluster config: %w", err)
	}
	scheme := runtime.NewScheme()
	if err := palworldv1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	return &RuntimeCR{client: c, namespace: namespace, name: name}, nil
}

// Get returns the PalworldServer.
func (r *RuntimeCR) Get(ctx context.Context) (*palworldv1alpha1.PalworldServer, error) {
	out := &palworldv1alpha1.PalworldServer{}
	if err := r.client.Get(ctx, types.NamespacedName{Namespace: r.namespace, Name: r.name}, out); err != nil {
		return nil, fmt.Errorf("get PalworldServer %s/%s: %w", r.namespace, r.name, err)
	}
	return out, nil
}

// Update writes the PalworldServer spec.
func (r *RuntimeCR) Update(ctx context.Context, server *palworldv1alpha1.PalworldServer) error {
	if err := r.client.Update(ctx, server); err != nil {
		return fmt.Errorf("update PalworldServer %s/%s: %w", r.namespace, r.name, err)
	}
	return nil
}

// GetSecret returns a Secret in the sidecar namespace.
func (r *RuntimeCR) GetSecret(ctx context.Context, name string) (*corev1.Secret, error) {
	out := &corev1.Secret{}
	if err := r.client.Get(ctx, types.NamespacedName{Namespace: r.namespace, Name: name}, out); err != nil {
		return nil, fmt.Errorf("get Secret %s/%s: %w", r.namespace, name, err)
	}
	return out, nil
}

// UpdateSecret writes a Secret. Callers must change only the intended key.
func (r *RuntimeCR) UpdateSecret(ctx context.Context, secret *corev1.Secret) error {
	if err := r.client.Update(ctx, secret); err != nil {
		return fmt.Errorf("update Secret %s/%s: %w", r.namespace, secret.Name, err)
	}
	return nil
}
