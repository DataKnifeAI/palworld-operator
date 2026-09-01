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

package controller

import (
	"context"
	"fmt"

	palworldv1alpha1 "github.com/DataKnifeAI/palworld-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func modManagerSidecar(
	spec palworldv1alpha1.PalworldServerSpec,
	names derivedNames,
	adminRef *corev1.SecretKeySelector,
) corev1.Container {
	port := modManagerPort(spec)
	runAsUser := containerUser
	return corev1.Container{
		Name:            containerModManager,
		Image:           modManagerImage(spec),
		ImagePullPolicy: imagePullPolicy(spec),
		Command:         []string{modManagerBinary},
		Ports: []corev1.ContainerPort{
			{Name: portNameModManager, ContainerPort: port, Protocol: corev1.ProtocolTCP},
		},
		Env: []corev1.EnvVar{
			{Name: envModManagerRoot, Value: modManagerMountPath},
			{Name: envModManagerListen, Value: fmt.Sprintf(":%d", port)},
			{Name: envModManagerUser, Value: palworldAdminUser},
			{Name: envModManagerDeployment, Value: names.deploymentName},
			{
				Name: envModManagerNamespace,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"},
				},
			},
			{
				Name: envModManagerPassword,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: adminRef,
				},
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: volumeMods, MountPath: modManagerMountPath},
		},
		Resources: podResources("10m", "32Mi", "200m", "128Mi"),
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:                &runAsUser,
			RunAsNonRoot:             boolPtr(true),
			AllowPrivilegeEscalation: boolPtr(false),
		},
	}
}

func (r *PalworldServerReconciler) reconcileModManagerRBAC(
	ctx context.Context,
	server *palworldv1alpha1.PalworldServer,
	names derivedNames,
) error {
	if !modManagerEnabled(server.Spec) {
		return r.deleteModManagerRBAC(ctx, server, names)
	}

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.modManagerSA,
			Namespace: server.Namespace,
		},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, sa, func() error {
		if err := controllerutil.SetControllerReference(server, sa, r.Scheme); err != nil {
			return err
		}
		sa.Labels = serverLabels(server.Name)
		return nil
	}); err != nil {
		return fmt.Errorf("reconcile mod-manager ServiceAccount: %w", err)
	}

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.modManagerRole,
			Namespace: server.Namespace,
		},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, role, func() error {
		if err := controllerutil.SetControllerReference(server, role, r.Scheme); err != nil {
			return err
		}
		role.Labels = serverLabels(server.Name)
		role.Rules = []rbacv1.PolicyRule{
			{
				APIGroups:     []string{"apps"},
				Resources:     []string{"deployments"},
				ResourceNames: []string{names.deploymentName},
				Verbs:         []string{"get", "patch", "update"},
			},
		}
		return nil
	}); err != nil {
		return fmt.Errorf("reconcile mod-manager Role: %w", err)
	}

	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.modManagerRole,
			Namespace: server.Namespace,
		},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, binding, func() error {
		if err := controllerutil.SetControllerReference(server, binding, r.Scheme); err != nil {
			return err
		}
		binding.Labels = serverLabels(server.Name)
		binding.Subjects = []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      names.modManagerSA,
				Namespace: server.Namespace,
			},
		}
		binding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     names.modManagerRole,
		}
		return nil
	}); err != nil {
		return fmt.Errorf("reconcile mod-manager RoleBinding: %w", err)
	}

	logf.FromContext(ctx).V(1).Info("reconciled mod-manager RBAC", "sa", names.modManagerSA)
	return nil
}

func (r *PalworldServerReconciler) deleteModManagerRBAC(
	ctx context.Context,
	server *palworldv1alpha1.PalworldServer,
	names derivedNames,
) error {
	objs := []client.Object{
		&rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: names.modManagerRole, Namespace: server.Namespace}},
		&rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: names.modManagerRole, Namespace: server.Namespace}},
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: names.modManagerSA, Namespace: server.Namespace}},
	}
	for _, obj := range objs {
		if err := r.deleteIfExists(ctx, obj); err != nil {
			return err
		}
	}
	return nil
}

func (r *PalworldServerReconciler) reconcileHTTPRoute(
	ctx context.Context,
	server *palworldv1alpha1.PalworldServer,
	names derivedNames,
) error {
	port := modManagerPort(server.Spec)
	httpRoute := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.modManagerHTTPRoute,
			Namespace: server.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, httpRoute, func() error {
		if err := controllerutil.SetControllerReference(server, httpRoute, r.Scheme); err != nil {
			return err
		}
		httpRoute.Labels = gatewayLabels(server.Name)
		httpRoute.Spec = gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name:        gatewayv1.ObjectName(names.gatewayName),
						Namespace:   ptr.To(gatewayv1.Namespace(server.Namespace)),
						SectionName: ptr.To(gatewayv1.SectionName(gatewayListenerModManagerHTTP)),
					},
				},
			},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: gatewayv1.ObjectName(names.envoyService),
									Port: ptr.To(port),
								},
							},
						},
					},
				},
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("reconcile HTTPRoute %s: %w", names.modManagerHTTPRoute, err)
	}
	logf.FromContext(ctx).V(1).Info("reconciled HTTPRoute", "operation", op, "name", names.modManagerHTTPRoute)
	return nil
}

func (r *PalworldServerReconciler) deleteIfExists(ctx context.Context, obj client.Object) error {
	err := r.Delete(ctx, obj)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}
