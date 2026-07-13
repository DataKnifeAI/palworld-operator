package controller

import (
	"fmt"
	"strings"

	palworldv1alpha1 "github.com/DataKnifeAI/palworld-operator/api/v1alpha1"
	egv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type derivedNames struct {
	pvcName         string
	configMapName   string
	deploymentName  string
	serviceName     string
	envoyService    string
	gatewayName     string
	envoyProxyName  string
	gameUDPRoute    string
	queryUDPRoute   string
	rconTCPRoute    string
	restTCPRoute    string
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func gatewayBaseName(name string) string {
	if strings.HasSuffix(name, "-server") {
		return strings.TrimSuffix(name, "-server")
	}
	return name
}

func deriveNames(server *palworldv1alpha1.PalworldServer) derivedNames {
	base := gatewayBaseName(server.Name)
	names := derivedNames{
		pvcName:        server.Name + "-files",
		configMapName:  server.Name + "-config",
		deploymentName: server.Name,
		serviceName:    server.Name,
		envoyService:   server.Name + "-envoy",
		gatewayName:    base + "-gateway",
		envoyProxyName: "game-" + base + "-kubevip",
		gameUDPRoute:   base + "-game-udp",
		queryUDPRoute:  base + "-query-udp",
		rconTCPRoute:   base + "-rcon-tcp",
		restTCPRoute:   base + "-rest-tcp",
	}
	if server.Spec.Gateway.GatewayName != "" {
		names.gatewayName = server.Spec.Gateway.GatewayName
	}
	if server.Spec.Gateway.EnvoyProxyName != "" {
		names.envoyProxyName = server.Spec.Gateway.EnvoyProxyName
	}
	return names
}

func serverImage(spec palworldv1alpha1.PalworldServerSpec) string {
	if spec.ServerImage != "" {
		return spec.ServerImage
	}
	return defaultServerImage
}

func isCommunityImage(spec palworldv1alpha1.PalworldServerSpec) bool {
	image := strings.ToLower(serverImage(spec))
	return strings.Contains(image, "thijsvanloef") || strings.Contains(image, "palworld-server-docker")
}

func savedMountPath(spec palworldv1alpha1.PalworldServerSpec) string {
	if isCommunityImage(spec) {
		return communitySavedMountPath
	}
	return officialSavedMountPath
}

func imagePullPolicy(spec palworldv1alpha1.PalworldServerSpec) corev1.PullPolicy {
	if spec.ImagePullPolicy != "" {
		return spec.ImagePullPolicy
	}
	return corev1.PullIfNotPresent
}

func gatewayClassName(spec palworldv1alpha1.PalworldServerSpec) string {
	if spec.Gateway.ClassName != "" {
		return spec.Gateway.ClassName
	}
	return defaultGatewayClassName
}

func externalTrafficPolicy(spec palworldv1alpha1.PalworldServerSpec) corev1.ServiceExternalTrafficPolicy {
	if spec.Gateway.ExternalTrafficPolicy != "" {
		return spec.Gateway.ExternalTrafficPolicy
	}
	return corev1.ServiceExternalTrafficPolicyCluster
}

func envoyExternalTrafficPolicy(spec palworldv1alpha1.PalworldServerSpec) egv1a1.ServiceExternalTrafficPolicy {
	if externalTrafficPolicy(spec) == corev1.ServiceExternalTrafficPolicyLocal {
		return egv1a1.ServiceExternalTrafficPolicyLocal
	}
	return egv1a1.ServiceExternalTrafficPolicyCluster
}

func gamePort(spec palworldv1alpha1.PalworldServerSpec) int32 {
	if spec.GamePort != 0 {
		return spec.GamePort
	}
	return defaultGamePort
}

func queryPort(spec palworldv1alpha1.PalworldServerSpec) int32 {
	if spec.QueryPort != 0 {
		return spec.QueryPort
	}
	return defaultQueryPort
}

func rconPort(spec palworldv1alpha1.PalworldServerSpec) int32 {
	if spec.RCON.Port != 0 {
		return spec.RCON.Port
	}
	return defaultRCONPort
}

func restPort(spec palworldv1alpha1.PalworldServerSpec) int32 {
	if spec.RESTAPI.Port != 0 {
		return spec.RESTAPI.Port
	}
	return defaultRESTPort
}

func rconEnabled(spec palworldv1alpha1.PalworldServerSpec) bool {
	return boolValue(spec.RCON.Enabled, true)
}

func restEnabled(spec palworldv1alpha1.PalworldServerSpec) bool {
	return boolValue(spec.RESTAPI.Enabled, true)
}

func restExposeViaGateway(spec palworldv1alpha1.PalworldServerSpec) bool {
	return boolValue(spec.RESTAPI.ExposeViaGateway, false)
}

func communityEnabled(spec palworldv1alpha1.PalworldServerSpec) bool {
	return boolValue(spec.Community.Enabled, false)
}

func maxPlayers(spec palworldv1alpha1.PalworldServerSpec) int32 {
	if spec.MaxPlayers != 0 {
		return spec.MaxPlayers
	}
	return defaultMaxPlayers
}

func storageSize(spec palworldv1alpha1.PalworldServerSpec) string {
	if spec.StorageSize != "" {
		return spec.StorageSize
	}
	return defaultStorageSize
}

func terminationGrace(spec palworldv1alpha1.PalworldServerSpec) int64 {
	if spec.TerminationGracePeriodSeconds != nil {
		return *spec.TerminationGracePeriodSeconds
	}
	return defaultTerminationGrace
}

func crossplayPlatforms(spec palworldv1alpha1.PalworldServerSpec) string {
	if spec.CrossplayPlatforms != "" {
		return spec.CrossplayPlatforms
	}
	return defaultCrossplayPlatforms
}

func publicIP(spec palworldv1alpha1.PalworldServerSpec) string {
	if spec.Community.PublicIP != "" {
		return spec.Community.PublicIP
	}
	if communityEnabled(spec) {
		return spec.Gateway.Address
	}
	return ""
}

func publicPort(spec palworldv1alpha1.PalworldServerSpec) int32 {
	if spec.Community.PublicPort != 0 {
		return spec.Community.PublicPort
	}
	return gamePort(spec)
}

func defaultResources(spec palworldv1alpha1.PalworldServerSpec) corev1.ResourceRequirements {
	if spec.Resources != nil {
		return *spec.Resources
	}
	return resourcesForPlayerCount(maxPlayers(spec))
}

// resourcesForPlayerCount returns conservative CPU/memory for Palworld on small nodes.
// Requests stay schedulable on ~8Gi worker nodes; limits allow burst.
func resourcesForPlayerCount(count int32) corev1.ResourceRequirements {
	switch {
	case count <= 4:
		return podResources("1", "3Gi", "4", "6Gi")
	case count <= 8:
		return podResources("2", "4Gi", "4", "7Gi")
	case count <= 16:
		return podResources("2", "5Gi", "6", "7Gi")
	default:
		return podResources("4", "6Gi", "8", "7Gi")
	}
}

func podResources(cpuRequest, memRequest, cpuLimit, memLimit string) corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resourceQuantity(cpuRequest),
			corev1.ResourceMemory: resourceQuantity(memRequest),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resourceQuantity(cpuLimit),
			corev1.ResourceMemory: resourceQuantity(memLimit),
		},
	}
}

func resourceQuantity(value string) resource.Quantity {
	return resource.MustParse(value)
}

func escapeINI(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return replacer.Replace(value)
}

func buildPalWorldSettingsINI(spec palworldv1alpha1.PalworldServerSpec, adminPassword, serverPassword string) string {
	name := spec.ServerName
	if name == "" {
		name = "Palworld Server"
	}
	opts := []string{
		fmt.Sprintf(`ServerName="%s"`, escapeINI(name)),
		fmt.Sprintf(`ServerDescription="%s"`, escapeINI(spec.ServerDescription)),
		fmt.Sprintf("ServerPlayerMaxNum=%d", maxPlayers(spec)),
		fmt.Sprintf(`AdminPassword="%s"`, escapeINI(adminPassword)),
		fmt.Sprintf(`ServerPassword="%s"`, escapeINI(serverPassword)),
		fmt.Sprintf("PublicPort=%d", publicPort(spec)),
		fmt.Sprintf(`PublicIP="%s"`, escapeINI(publicIP(spec))),
		fmt.Sprintf("RCONEnabled=%s", boolINI(rconEnabled(spec))),
		fmt.Sprintf("RCONPort=%d", rconPort(spec)),
		fmt.Sprintf("RESTAPIEnabled=%s", boolINI(restEnabled(spec))),
		fmt.Sprintf("RESTAPIPort=%d", restPort(spec)),
		fmt.Sprintf(`CrossplayPlatforms="%s"`, escapeINI(crossplayPlatforms(spec))),
	}
	return fmt.Sprintf("[/Script/Pal.PalGameWorldSettings]\nOptionSettings=(%s)\n", strings.Join(opts, ","))
}

func boolINI(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

func officialCommandArgs(spec palworldv1alpha1.PalworldServerSpec) []string {
	args := []string{fmt.Sprintf("-port=%d", gamePort(spec))}
	if boolValue(spec.Multithreading, true) {
		args = append(args,
			"-useperfthreads",
			"-NoAsyncLoadingThread",
			"-UseMultithreadForDS",
		)
	}
	return args
}

func communityEnv(spec palworldv1alpha1.PalworldServerSpec, adminPassword, serverPassword string) []corev1.EnvVar {
	name := spec.ServerName
	if name == "" {
		name = "Palworld Server"
	}
	env := []corev1.EnvVar{
		{Name: "PUID", Value: "1000"},
		{Name: "PGID", Value: "1000"},
		{Name: "PORT", Value: fmt.Sprintf("%d", gamePort(spec))},
		{Name: "QUERY_PORT", Value: fmt.Sprintf("%d", queryPort(spec))},
		{Name: "PLAYERS", Value: fmt.Sprintf("%d", maxPlayers(spec))},
		{Name: "SERVER_NAME", Value: name},
		{Name: "SERVER_DESCRIPTION", Value: spec.ServerDescription},
		{Name: "ADMIN_PASSWORD", Value: adminPassword},
		{Name: "SERVER_PASSWORD", Value: serverPassword},
		{Name: "RCON_ENABLED", Value: fmt.Sprintf("%t", rconEnabled(spec))},
		{Name: "RCON_PORT", Value: fmt.Sprintf("%d", rconPort(spec))},
		{Name: "REST_API_ENABLED", Value: fmt.Sprintf("%t", restEnabled(spec))},
		{Name: "REST_API_PORT", Value: fmt.Sprintf("%d", restPort(spec))},
		{Name: "MULTITHREADING", Value: fmt.Sprintf("%t", boolValue(spec.Multithreading, true))},
		{Name: "COMMUNITY", Value: fmt.Sprintf("%t", communityEnabled(spec))},
		{Name: "UPDATE_ON_BOOT", Value: fmt.Sprintf("%t", boolValue(spec.UpdateOnBoot, true))},
		{Name: "CROSSPLAY_PLATFORMS", Value: crossplayPlatforms(spec)},
	}
	if ip := publicIP(spec); ip != "" {
		env = append(env, corev1.EnvVar{Name: "PUBLIC_IP", Value: ip})
	}
	env = append(env, corev1.EnvVar{Name: "PUBLIC_PORT", Value: fmt.Sprintf("%d", publicPort(spec))})
	return env
}

func gameServicePorts(spec palworldv1alpha1.PalworldServerSpec) []corev1.ServicePort {
	ports := []corev1.ServicePort{
		{
			Name:       "game-udp",
			Port:       gamePort(spec),
			TargetPort: intstr.FromInt32(gamePort(spec)),
			Protocol:   corev1.ProtocolUDP,
		},
		{
			Name:       "query-udp",
			Port:       queryPort(spec),
			TargetPort: intstr.FromInt32(queryPort(spec)),
			Protocol:   corev1.ProtocolUDP,
		},
	}
	if rconEnabled(spec) {
		ports = append(ports, corev1.ServicePort{
			Name:       "rcon-tcp",
			Port:       rconPort(spec),
			TargetPort: intstr.FromInt32(rconPort(spec)),
			Protocol:   corev1.ProtocolTCP,
		})
	}
	if restEnabled(spec) {
		ports = append(ports, corev1.ServicePort{
			Name:       "rest-tcp",
			Port:       restPort(spec),
			TargetPort: intstr.FromInt32(restPort(spec)),
			Protocol:   corev1.ProtocolTCP,
		})
	}
	return ports
}

func containerPorts(spec palworldv1alpha1.PalworldServerSpec) []corev1.ContainerPort {
	ports := []corev1.ContainerPort{
		{Name: "game-udp", ContainerPort: gamePort(spec), Protocol: corev1.ProtocolUDP},
		{Name: "query-udp", ContainerPort: queryPort(spec), Protocol: corev1.ProtocolUDP},
	}
	if rconEnabled(spec) {
		ports = append(ports, corev1.ContainerPort{
			Name: "rcon-tcp", ContainerPort: rconPort(spec), Protocol: corev1.ProtocolTCP,
		})
	}
	if restEnabled(spec) {
		ports = append(ports, corev1.ContainerPort{
			Name: "rest-tcp", ContainerPort: restPort(spec), Protocol: corev1.ProtocolTCP,
		})
	}
	return ports
}
