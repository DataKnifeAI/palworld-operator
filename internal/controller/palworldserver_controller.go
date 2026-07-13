package controller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	palworldv1alpha1 "github.com/DataKnifeAI/palworld-operator/api/v1alpha1"
	egv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
)

// PalworldServerReconciler reconciles a PalworldServer object.
type PalworldServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=palworld.dataknife.ai,resources=palworldservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=palworld.dataknife.ai,resources=palworldservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=palworld.dataknife.ai,resources=palworldservers/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=tcproutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=udproutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.envoyproxy.io,resources=envoyproxies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *PalworldServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	server := &palworldv1alpha1.PalworldServer{}
	if err := r.Get(ctx, req.NamespacedName, server); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !controllerutil.ContainsFinalizer(server, finalizerName) {
		controllerutil.AddFinalizer(server, finalizerName)
		if err := r.Update(ctx, server); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if !server.DeletionTimestamp.IsZero() {
		controllerutil.RemoveFinalizer(server, finalizerName)
		if err := r.Update(ctx, server); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	names := deriveNames(server)

	adminPassword, serverPassword, credentialsSecret, credentialsGenerated, err := r.resolvePasswords(ctx, server)
	if err != nil {
		return r.failStatus(ctx, server, err)
	}

	if err := r.reconcilePVC(ctx, server, names.pvcName); err != nil {
		return r.failStatus(ctx, server, err)
	}
	if err := r.reconcileConfigMap(ctx, server, names.configMapName, adminPassword, serverPassword); err != nil {
		return r.failStatus(ctx, server, err)
	}
	if err := r.reconcileDeployment(ctx, server, names, adminPassword, serverPassword); err != nil {
		return r.failStatus(ctx, server, err)
	}
	if err := r.reconcileClusterIPService(ctx, server, names.serviceName, serverLabels(server.Name)); err != nil {
		return r.failStatus(ctx, server, err)
	}
	if err := r.reconcileClusterIPService(ctx, server, names.envoyService, envoyBackendLabels(server.Name)); err != nil {
		return r.failStatus(ctx, server, err)
	}
	if err := r.reconcileEnvoyGateway(ctx, server, names); err != nil {
		return r.failStatus(ctx, server, err)
	}

	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: names.deploymentName, Namespace: server.Namespace}, deployment); err != nil {
		return r.failStatus(ctx, server, err)
	}

	gateway := &gatewayv1.Gateway{}
	if err := r.Get(ctx, types.NamespacedName{Name: names.gatewayName, Namespace: server.Namespace}, gateway); err != nil {
		return r.failStatus(ctx, server, err)
	}

	ready := deployment.Status.ReadyReplicas > 0
	phase := palworldv1alpha1.PhasePending
	message := "Waiting for game server pod"
	if ready {
		phase = palworldv1alpha1.PhaseRunning
		message = "Game server is running"
	}

	server.Status.Phase = phase
	server.Status.Ready = ready
	server.Status.ConnectionPort = gamePort(server.Spec)
	server.Status.ConnectionAddress = connectionAddressFromGateway(server, gateway)
	server.Status.Message = message
	server.Status.CredentialsSecretName = credentialsSecret
	server.Status.CredentialsGenerated = credentialsGenerated
	server.Status.ObservedGeneration = server.Generation
	meta.SetStatusCondition(&server.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             conditionStatus(ready),
		Reason:             phase,
		Message:            message,
		ObservedGeneration: server.Generation,
		LastTransitionTime: metav1.NewTime(time.Now()),
	})

	if err := r.Status().Update(ctx, server); err != nil {
		log.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	if !ready {
		return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
	}
	return ctrl.Result{}, nil
}

func (r *PalworldServerReconciler) resolvePasswords(
	ctx context.Context,
	server *palworldv1alpha1.PalworldServer,
) (adminPassword, serverPassword, credentialsSecret string, credentialsGenerated bool, err error) {
	credentialsGenerated = server.Spec.GenerateSecrets
	if server.Spec.GenerateSecrets {
		credentialsSecret = credentialsSecretName(server)
		if err = r.reconcileCredentialsSecret(ctx, server, credentialsSecret); err != nil {
			return "", "", "", false, err
		}
	}

	adminRef := server.Spec.AdminPasswordSecretRef
	if adminRef == nil && server.Spec.GenerateSecrets {
		adminRef = defaultSecretKeySelector(credentialsSecret, secretKeyAdminPassword)
	}
	serverRef := server.Spec.ServerPasswordSecretRef
	if serverRef == nil && server.Spec.GenerateSecrets {
		serverRef = defaultSecretKeySelector(credentialsSecret, secretKeyServerPassword)
	}

	if adminRef != nil {
		adminPassword, err = r.readSecretKey(ctx, server.Namespace, adminRef)
		if err != nil {
			return "", "", "", false, fmt.Errorf("adminPasswordSecretRef: %w", err)
		}
		if credentialsSecret == "" {
			credentialsSecret = adminRef.Name
		}
	}
	if serverRef != nil {
		serverPassword, err = r.readSecretKey(ctx, server.Namespace, serverRef)
		if err != nil {
			return "", "", "", false, fmt.Errorf("serverPasswordSecretRef: %w", err)
		}
		if credentialsSecret == "" {
			credentialsSecret = serverRef.Name
		}
	}
	return adminPassword, serverPassword, credentialsSecret, credentialsGenerated, nil
}

func (r *PalworldServerReconciler) reconcileCredentialsSecret(
	ctx context.Context,
	server *palworldv1alpha1.PalworldServer,
	name string,
) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: server.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		if err := controllerutil.SetControllerReference(server, secret, r.Scheme); err != nil {
			return err
		}
		secret.Labels = serverLabels(server.Name)
		secret.Type = corev1.SecretTypeOpaque
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		for _, key := range []string{secretKeyAdminPassword, secretKeyServerPassword} {
			if len(secret.Data[key]) > 0 {
				continue
			}
			password, genErr := generatePassword()
			if genErr != nil {
				return fmt.Errorf("generate %s: %w", key, genErr)
			}
			secret.Data[key] = []byte(password)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("reconcile credentials Secret: %w", err)
	}
	logf.FromContext(ctx).V(1).Info("reconciled credentials Secret", "operation", op, "name", name)
	return nil
}

func (r *PalworldServerReconciler) readSecretKey(
	ctx context.Context,
	namespace string,
	ref *corev1.SecretKeySelector,
) (string, error) {
	if ref == nil || ref.Name == "" || ref.Key == "" {
		return "", fmt.Errorf("secret name and key are required")
	}
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: namespace}, secret); err != nil {
		return "", err
	}
	value, ok := secret.Data[ref.Key]
	if !ok {
		if ref.Optional != nil && *ref.Optional {
			return "", nil
		}
		return "", fmt.Errorf("key %q not found in secret %q", ref.Key, ref.Name)
	}
	return string(value), nil
}

func (r *PalworldServerReconciler) reconcilePVC(ctx context.Context, server *palworldv1alpha1.PalworldServer, name string) error {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: server.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, pvc, func() error {
		if err := controllerutil.SetControllerReference(server, pvc, r.Scheme); err != nil {
			return err
		}
		pvc.Labels = serverLabels(server.Name)
		if pvc.Spec.AccessModes == nil {
			pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
		}
		if pvc.Spec.Resources.Requests == nil {
			pvc.Spec.Resources.Requests = corev1.ResourceList{
				corev1.ResourceStorage: resourceQuantity(storageSize(server.Spec)),
			}
		}
		if server.Spec.StorageClassName != "" {
			pvc.Spec.StorageClassName = &server.Spec.StorageClassName
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("reconcile PVC: %w", err)
	}
	logf.FromContext(ctx).V(1).Info("reconciled PVC", "operation", op, "name", name)
	return nil
}

func (r *PalworldServerReconciler) reconcileConfigMap(
	ctx context.Context,
	server *palworldv1alpha1.PalworldServer,
	name, adminPassword, serverPassword string,
) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: server.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		if err := controllerutil.SetControllerReference(server, configMap, r.Scheme); err != nil {
			return err
		}
		configMap.Labels = serverLabels(server.Name)
		configMap.Data = map[string]string{
			settingsConfigKey: buildPalWorldSettingsINI(server.Spec, adminPassword, serverPassword),
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("reconcile ConfigMap: %w", err)
	}
	logf.FromContext(ctx).V(1).Info("reconciled ConfigMap", "operation", op, "name", name)
	return nil
}

func (r *PalworldServerReconciler) reconcileDeployment(
	ctx context.Context,
	server *palworldv1alpha1.PalworldServer,
	names derivedNames,
	adminPassword, serverPassword string,
) error {
	replicas := int32(1)
	runAsUser := containerUser
	grace := terminationGrace(server.Spec)
	community := isCommunityImage(server.Spec)
	mountPath := savedMountPath(server.Spec)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.deploymentName,
			Namespace: server.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		if err := controllerutil.SetControllerReference(server, deployment, r.Scheme); err != nil {
			return err
		}

		deployment.Labels = serverLabels(server.Name)
		deployment.Spec.Replicas = &replicas
		deployment.Spec.Strategy = appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}
		deployment.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: serverLabels(server.Name),
		}
		deployment.Spec.Template.Labels = serverLabels(server.Name)

		container := corev1.Container{
			Name:            "palworld",
			Image:           serverImage(server.Spec),
			ImagePullPolicy: imagePullPolicy(server.Spec),
			Ports:           containerPorts(server.Spec),
			Resources:       defaultResources(server.Spec),
			VolumeMounts: []corev1.VolumeMount{
				{Name: "saves", MountPath: mountPath},
			},
		}

		volumes := []corev1.Volume{
			{
				Name: "saves",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: names.pvcName,
					},
				},
			},
			{
				Name: "settings",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: names.configMapName},
					},
				},
			},
		}

		var initContainers []corev1.Container
		if community {
			container.Env = communityEnv(server.Spec, adminPassword, serverPassword)
			container.SecurityContext = &corev1.SecurityContext{
				RunAsUser:                &runAsUser,
				RunAsNonRoot:             boolPtr(true),
				AllowPrivilegeEscalation: boolPtr(false),
			}
		} else {
			container.Args = officialCommandArgs(server.Spec)
			initContainers = []corev1.Container{
				{
					Name:  "seed-settings",
					Image: initContainerImage,
					Command: []string{
						"sh", "-c",
						fmt.Sprintf(
							"mkdir -p /saves/Config/LinuxServer && cp /settings/%s /saves/%s",
							settingsConfigKey, settingsRelativePath,
						),
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "saves", MountPath: "/saves"},
						{Name: "settings", MountPath: "/settings"},
					},
				},
			}
		}

		podSpec := corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				FSGroup: &runAsUser,
			},
			TerminationGracePeriodSeconds: &grace,
			InitContainers:                initContainers,
			Containers:                    []corev1.Container{container},
			Volumes:                       volumes,
		}

		if len(server.Spec.ImagePullSecrets) > 0 {
			podSpec.ImagePullSecrets = server.Spec.ImagePullSecrets
		}
		if len(server.Spec.NodeSelector) > 0 {
			podSpec.NodeSelector = server.Spec.NodeSelector
		}

		deployment.Spec.Template.Spec = podSpec
		return nil
	})
	if err != nil {
		return fmt.Errorf("reconcile Deployment: %w", err)
	}
	logf.FromContext(ctx).V(1).Info("reconciled Deployment", "operation", op, "name", names.deploymentName)
	return nil
}

func (r *PalworldServerReconciler) reconcileClusterIPService(
	ctx context.Context,
	server *palworldv1alpha1.PalworldServer,
	name string,
	labels map[string]string,
) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: server.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		if err := controllerutil.SetControllerReference(server, service, r.Scheme); err != nil {
			return err
		}

		service.Labels = labels
		service.Spec.Type = corev1.ServiceTypeClusterIP
		service.Spec.Selector = serverLabels(server.Name)
		service.Spec.Ports = gameServicePorts(server.Spec)
		return nil
	})
	if err != nil {
		return fmt.Errorf("reconcile Service %s: %w", name, err)
	}
	logf.FromContext(ctx).V(1).Info("reconciled Service", "operation", op, "name", name)
	return nil
}

func (r *PalworldServerReconciler) failStatus(ctx context.Context, server *palworldv1alpha1.PalworldServer, err error) (ctrl.Result, error) {
	server.Status.Phase = palworldv1alpha1.PhaseFailed
	server.Status.Ready = false
	server.Status.Message = err.Error()
	server.Status.ObservedGeneration = server.Generation
	meta.SetStatusCondition(&server.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             palworldv1alpha1.PhaseFailed,
		Message:            err.Error(),
		ObservedGeneration: server.Generation,
		LastTransitionTime: metav1.NewTime(time.Now()),
	})
	if statusErr := r.Status().Update(ctx, server); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{}, err
}

func (r *PalworldServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&palworldv1alpha1.PalworldServer{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&gatewayv1.Gateway{}).
		Owns(&gatewayv1alpha2.TCPRoute{}).
		Owns(&gatewayv1alpha2.UDPRoute{}).
		Owns(&egv1a1.EnvoyProxy{}).
		Complete(r)
}

func serverLabels(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       name,
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/managed-by": "palworld-operator",
	}
}

func envoyBackendLabels(instanceName string) map[string]string {
	labels := serverLabels(instanceName)
	labels["app.kubernetes.io/component"] = "envoy-backend"
	return labels
}

func conditionStatus(ready bool) metav1.ConditionStatus {
	if ready {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}

func boolPtr(value bool) *bool {
	return &value
}
