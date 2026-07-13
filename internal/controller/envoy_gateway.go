package controller

import (
	"context"
	"fmt"

	palworldv1alpha1 "github.com/DataKnifeAI/palworld-operator/api/v1alpha1"
	egv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

func (r *PalworldServerReconciler) reconcileEnvoyGateway(
	ctx context.Context,
	server *palworldv1alpha1.PalworldServer,
	names derivedNames,
) error {
	if server.Spec.Gateway.Address == "" {
		return fmt.Errorf("spec.gateway.address is required")
	}

	if err := r.reconcileEnvoyProxy(ctx, server, names); err != nil {
		return err
	}
	if err := r.reconcileGateway(ctx, server, names); err != nil {
		return err
	}
	if err := r.reconcileUDPRoute(ctx, server, names, names.gameUDPRoute, gatewayListenerGameUDP, gamePort(server.Spec)); err != nil {
		return err
	}
	if err := r.reconcileUDPRoute(ctx, server, names, names.queryUDPRoute, gatewayListenerQueryUDP, queryPort(server.Spec)); err != nil {
		return err
	}
	// RCON stays ClusterIP-only by default for security.
	if restExposeViaGateway(server.Spec) && restEnabled(server.Spec) {
		if err := r.reconcileTCPRoute(ctx, server, names, names.restTCPRoute, gatewayListenerRESTTCP, restPort(server.Spec)); err != nil {
			return err
		}
	}
	return nil
}

func (r *PalworldServerReconciler) reconcileEnvoyProxy(
	ctx context.Context,
	server *palworldv1alpha1.PalworldServer,
	names derivedNames,
) error {
	envoyProxy := &egv1a1.EnvoyProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.envoyProxyName,
			Namespace: server.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, envoyProxy, func() error {
		if err := controllerutil.SetControllerReference(server, envoyProxy, r.Scheme); err != nil {
			return err
		}
		envoyProxy.Labels = gatewayLabels(server.Name)
		policy := envoyExternalTrafficPolicy(server.Spec)
		envoyProxy.Spec = egv1a1.EnvoyProxySpec{
			Logging: egv1a1.ProxyLogging{
				Level: map[egv1a1.ProxyLogComponent]egv1a1.LogLevel{
					egv1a1.LogComponentDefault: egv1a1.LogLevelWarn,
				},
			},
			Provider: &egv1a1.EnvoyProxyProvider{
				Type: egv1a1.EnvoyProxyProviderTypeKubernetes,
				Kubernetes: &egv1a1.EnvoyProxyKubernetesProvider{
					EnvoyService: &egv1a1.KubernetesServiceSpec{
						Type:                  ptr.To(egv1a1.ServiceTypeLoadBalancer),
						LoadBalancerIP:        ptr.To(server.Spec.Gateway.Address),
						ExternalTrafficPolicy: ptr.To(policy),
					},
				},
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("reconcile EnvoyProxy: %w", err)
	}
	logf.FromContext(ctx).V(1).Info("reconciled EnvoyProxy", "operation", op, "name", names.envoyProxyName)
	return nil
}

func (r *PalworldServerReconciler) reconcileGateway(
	ctx context.Context,
	server *palworldv1alpha1.PalworldServer,
	names derivedNames,
) error {
	listeners := []gatewayv1.Listener{
		{
			Name:     gatewayv1.SectionName(gatewayListenerGameUDP),
			Port:     gamePort(server.Spec),
			Protocol: gatewayv1.UDPProtocolType,
			AllowedRoutes: &gatewayv1.AllowedRoutes{
				Namespaces: &gatewayv1.RouteNamespaces{
					From: ptr.To(gatewayv1.NamespacesFromSame),
				},
			},
		},
		{
			Name:     gatewayv1.SectionName(gatewayListenerQueryUDP),
			Port:     queryPort(server.Spec),
			Protocol: gatewayv1.UDPProtocolType,
			AllowedRoutes: &gatewayv1.AllowedRoutes{
				Namespaces: &gatewayv1.RouteNamespaces{
					From: ptr.To(gatewayv1.NamespacesFromSame),
				},
			},
		},
	}
	if restExposeViaGateway(server.Spec) && restEnabled(server.Spec) {
		listeners = append(listeners, gatewayv1.Listener{
			Name:     gatewayv1.SectionName(gatewayListenerRESTTCP),
			Port:     restPort(server.Spec),
			Protocol: gatewayv1.TCPProtocolType,
			AllowedRoutes: &gatewayv1.AllowedRoutes{
				Namespaces: &gatewayv1.RouteNamespaces{
					From: ptr.To(gatewayv1.NamespacesFromSame),
				},
			},
		})
	}

	gateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.gatewayName,
			Namespace: server.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, gateway, func() error {
		if err := controllerutil.SetControllerReference(server, gateway, r.Scheme); err != nil {
			return err
		}
		gateway.Labels = gatewayLabels(server.Name)
		gateway.Spec = gatewayv1.GatewaySpec{
			GatewayClassName: gatewayv1.ObjectName(gatewayClassName(server.Spec)),
			Addresses: []gatewayv1.GatewaySpecAddress{
				{
					Type:  ptr.To(gatewayv1.IPAddressType),
					Value: server.Spec.Gateway.Address,
				},
			},
			Infrastructure: &gatewayv1.GatewayInfrastructure{
				ParametersRef: &gatewayv1.LocalParametersReference{
					Group: gatewayv1.Group(egv1a1.GroupVersion.Group),
					Kind:  gatewayv1.Kind("EnvoyProxy"),
					Name:  names.envoyProxyName,
				},
			},
			Listeners: listeners,
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("reconcile Gateway: %w", err)
	}
	logf.FromContext(ctx).V(1).Info("reconciled Gateway", "operation", op, "name", names.gatewayName)
	return nil
}

func (r *PalworldServerReconciler) reconcileUDPRoute(
	ctx context.Context,
	server *palworldv1alpha1.PalworldServer,
	names derivedNames,
	routeName, listenerName string,
	port int32,
) error {
	udpRoute := &gatewayv1alpha2.UDPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeName,
			Namespace: server.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, udpRoute, func() error {
		if err := controllerutil.SetControllerReference(server, udpRoute, r.Scheme); err != nil {
			return err
		}
		udpRoute.Labels = gatewayLabels(server.Name)
		udpRoute.Spec = gatewayv1alpha2.UDPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name:        gatewayv1.ObjectName(names.gatewayName),
						Namespace:   ptr.To(gatewayv1.Namespace(server.Namespace)),
						SectionName: ptr.To(gatewayv1.SectionName(listenerName)),
					},
				},
			},
			Rules: []gatewayv1alpha2.UDPRouteRule{
				{
					BackendRefs: []gatewayv1alpha2.BackendRef{
						{
							BackendObjectReference: gatewayv1.BackendObjectReference{
								Name: gatewayv1.ObjectName(names.envoyService),
								Port: ptr.To(port),
							},
						},
					},
				},
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("reconcile UDPRoute %s: %w", routeName, err)
	}
	logf.FromContext(ctx).V(1).Info("reconciled UDPRoute", "operation", op, "name", routeName)
	return nil
}

func (r *PalworldServerReconciler) reconcileTCPRoute(
	ctx context.Context,
	server *palworldv1alpha1.PalworldServer,
	names derivedNames,
	routeName, listenerName string,
	port int32,
) error {
	tcpRoute := &gatewayv1alpha2.TCPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeName,
			Namespace: server.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, tcpRoute, func() error {
		if err := controllerutil.SetControllerReference(server, tcpRoute, r.Scheme); err != nil {
			return err
		}
		tcpRoute.Labels = gatewayLabels(server.Name)
		tcpRoute.Spec = gatewayv1alpha2.TCPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name:        gatewayv1.ObjectName(names.gatewayName),
						Namespace:   ptr.To(gatewayv1.Namespace(server.Namespace)),
						SectionName: ptr.To(gatewayv1.SectionName(listenerName)),
					},
				},
			},
			Rules: []gatewayv1alpha2.TCPRouteRule{
				{
					BackendRefs: []gatewayv1alpha2.BackendRef{
						{
							BackendObjectReference: gatewayv1.BackendObjectReference{
								Name: gatewayv1.ObjectName(names.envoyService),
								Port: ptr.To(port),
							},
						},
					},
				},
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("reconcile TCPRoute %s: %w", routeName, err)
	}
	logf.FromContext(ctx).V(1).Info("reconciled TCPRoute", "operation", op, "name", routeName)
	return nil
}

func gatewayLabels(instanceName string) map[string]string {
	labels := serverLabels(instanceName)
	labels["app.kubernetes.io/component"] = "envoy-gateway"
	return labels
}

func connectionAddressFromGateway(server *palworldv1alpha1.PalworldServer, gateway *gatewayv1.Gateway) string {
	if gateway != nil && len(gateway.Status.Addresses) > 0 {
		return gateway.Status.Addresses[0].Value
	}
	return server.Spec.Gateway.Address
}
