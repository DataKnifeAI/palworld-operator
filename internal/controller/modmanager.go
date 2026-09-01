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

func serverManagerSidecar(
	spec palworldv1alpha1.PalworldServerSpec,
	names derivedNames,
	adminRef *corev1.SecretKeySelector,
) corev1.Container {
	port := serverManagerPort(spec)
	runAsUser := containerUser
	env := []corev1.EnvVar{
		{Name: envServerManagerListen, Value: fmt.Sprintf(":%d", port)},
		{Name: envServerManagerUser, Value: palworldAdminUser},
		{Name: envServerManagerDeployment, Value: names.deploymentName},
		{Name: envServerManagerSaves, Value: serverManagerSavesPath},
		{
			Name: envServerManagerNamespace,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"},
			},
		},
		{
			Name: envServerManagerPassword,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: adminRef,
			},
		},
	}
	if restEnabled(spec) {
		env = append(env, corev1.EnvVar{
			Name:  envServerManagerRESTBase,
			Value: fmt.Sprintf("http://127.0.0.1:%d", restPort(spec)),
		})
	}
	mounts := []corev1.VolumeMount{
		{Name: volumeSaves, MountPath: serverManagerSavesPath},
	}
	if modsEnabled(spec) {
		env = append(env, corev1.EnvVar{Name: envServerManagerRoot, Value: serverManagerModsPath})
		mounts = append(mounts, corev1.VolumeMount{Name: volumeMods, MountPath: serverManagerModsPath})
	}
	return corev1.Container{
		Name:            containerServerManager,
		Image:           serverManagerImage(spec),
		ImagePullPolicy: imagePullPolicy(spec),
		Command:         []string{serverManagerBinary},
		Ports: []corev1.ContainerPort{
			{Name: portNameServerManager, ContainerPort: port, Protocol: corev1.ProtocolTCP},
		},
		Env:          env,
		VolumeMounts: mounts,
		Resources:    podResources("10m", "32Mi", "200m", "256Mi"),
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
	if !serverManagerEnabled(server.Spec) {
		return r.deleteServerManagerRBAC(ctx, server, names)
	}

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.serverManagerSA,
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
		return fmt.Errorf("reconcile server-manager ServiceAccount: %w", err)
	}

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.serverManagerRole,
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
		return fmt.Errorf("reconcile server-manager Role: %w", err)
	}

	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.serverManagerRole,
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
				Name:      names.serverManagerSA,
				Namespace: server.Namespace,
			},
		}
		binding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     names.serverManagerRole,
		}
		return nil
	}); err != nil {
		return fmt.Errorf("reconcile server-manager RoleBinding: %w", err)
	}

	logf.FromContext(ctx).V(1).Info("reconciled server-manager RBAC", "sa", names.serverManagerSA)
	return nil
}

func (r *PalworldServerReconciler) deleteServerManagerRBAC(
	ctx context.Context,
	server *palworldv1alpha1.PalworldServer,
	names derivedNames,
) error {
	objs := []client.Object{
		&rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: names.serverManagerRole, Namespace: server.Namespace}},
		&rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: names.serverManagerRole, Namespace: server.Namespace}},
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: names.serverManagerSA, Namespace: server.Namespace}},
		&rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: server.Name + legacyModManagerSASuffix, Namespace: server.Namespace}},
		&rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: server.Name + legacyModManagerSASuffix, Namespace: server.Namespace}},
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: server.Name + legacyModManagerSASuffix, Namespace: server.Namespace}},
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
	port := serverManagerPort(server.Spec)
	httpRoute := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.serverManagerHTTPRoute,
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
						SectionName: ptr.To(gatewayv1.SectionName(gatewayListenerServerManagerHTTP)),
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
		return fmt.Errorf("reconcile HTTPRoute %s: %w", names.serverManagerHTTPRoute, err)
	}
	logf.FromContext(ctx).V(1).Info("reconciled HTTPRoute", "operation", op, "name", names.serverManagerHTTPRoute)
	return nil
}

func (r *PalworldServerReconciler) deleteIfExists(ctx context.Context, obj client.Object) error {
	err := r.Delete(ctx, obj)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}
