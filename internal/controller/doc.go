// Package controller will contain the PalworldServer reconciler.
//
// Implement in TASKS.md T3–T7, reusing patterns from
// github.com/DataKnifeAI/windrose-operator/internal/controller:
//   - Deployment + PVC + ClusterIP Services
//   - Envoy Gateway / EnvoyProxy / TCPRoute / UDPRoute
//   - Finalizers, status, resource auto-selection
package controller
