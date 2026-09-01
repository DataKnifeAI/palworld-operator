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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const restartedAtAnnotation = "kubectl.kubernetes.io/restartedAt"

// Restarter rolls the Palworld game Deployment (Recreate).
type Restarter interface {
	Restart(ctx context.Context) error
}

// DeploymentRestarter PATCHes a namespaced Deployment using in-cluster config.
// The sidecar ServiceAccount must be limited to that one Deployment.
type DeploymentRestarter struct {
	Namespace string
	Name      string
}

// Restart annotates the pod template so Kubernetes Recreate-rolls the Deployment.
func (d *DeploymentRestarter) Restart(ctx context.Context) error {
	if d.Namespace == "" || d.Name == "" {
		return fmt.Errorf("namespace and deployment name are required")
	}
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("in-cluster config: %w", err)
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]string{
						restartedAtAnnotation: time.Now().UTC().Format(time.RFC3339Nano),
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}
	_, err = client.AppsV1().Deployments(d.Namespace).Patch(
		ctx, d.Name, types.StrategicMergePatchType, payload, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("patch deployment %s/%s: %w", d.Namespace, d.Name, err)
	}
	return nil
}
