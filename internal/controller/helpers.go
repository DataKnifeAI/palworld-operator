package controller

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	palworldv1alpha1 "github.com/DataKnifeAI/palworld-operator/api/v1alpha1"
	egv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// managementOptionKeys are emitted first and always override spec.optionSettings.
var managementOptionKeys = []string{
	"ServerName",
	"ServerDescription",
	"ServerPlayerMaxNum",
	"AdminPassword",
	"ServerPassword",
	"PublicPort",
	"PublicIP",
	"RCONEnabled",
	"RCONPort",
	"RESTAPIEnabled",
	"RESTAPIPort",
	"CrossplayPlatforms",
}

var iniNumberRE = regexp.MustCompile(`^-?\d+(\.\d+)?$`)

// communityOptionEnv maps PalWorldSettings.ini OptionSettings keys to
// thijsvanloef/palworld-server-docker environment variables. Unmapped keys
// are skipped — full INI coverage is official-image-primary.
var communityOptionEnv = map[string]string{
	"DayTimeSpeedRate":                   "DAYTIME_SPEEDRATE",
	"NightTimeSpeedRate":                 "NIGHTTIME_SPEEDRATE",
	"ExpRate":                            "EXP_RATE",
	"PalCaptureRate":                     "PAL_CAPTURE_RATE",
	"PalSpawnNumRate":                    "PAL_SPAWN_NUM_RATE",
	"PalDamageRateAttack":                "PAL_DAMAGE_RATE_ATTACK",
	"PalDamageRateDefense":               "PAL_DAMAGE_RATE_DEFENSE",
	"PlayerDamageRateAttack":             "PLAYER_DAMAGE_RATE_ATTACK",
	"PlayerDamageRateDefense":            "PLAYER_DAMAGE_RATE_DEFENSE",
	"PlayerStomachDecreaceRate":          "PLAYER_STOMACH_DECREASE_RATE",
	"PlayerStaminaDecreaceRate":          "PLAYER_STAMINA_DECREASE_RATE",
	"PlayerAutoHPRegeneRate":             "PLAYER_AUTO_HP_REGEN_RATE",
	"PlayerAutoHpRegeneRateInSleep":      "PLAYER_AUTO_HP_REGEN_RATE_IN_SLEEP",
	"PalStomachDecreaceRate":             "PAL_STOMACH_DECREASE_RATE",
	"PalStaminaDecreaceRate":             "PAL_STAMINA_DECREASE_RATE",
	"PalAutoHPRegeneRate":                "PAL_AUTO_HP_REGEN_RATE",
	"PalAutoHpRegeneRateInSleep":         "PAL_AUTO_HP_REGEN_RATE_IN_SLEEP",
	"BuildObjectDamageRate":              "BUILD_OBJECT_DAMAGE_RATE",
	"BuildObjectDeteriorationDamageRate": "BUILD_OBJECT_DETERIORATION_DAMAGE_RATE",
	"CollectionDropRate":                 "COLLECTION_DROP_RATE",
	"CollectionObjectHpRate":             "COLLECTION_OBJECT_HP_RATE",
	"CollectionObjectRespawnSpeedRate":   "COLLECTION_OBJECT_RESPAWN_SPEED_RATE",
	"EnemyDropItemRate":                  "ENEMY_DROP_ITEM_RATE",
	"DeathPenalty":                       "DEATH_PENALTY",
	"bEnableInvaderEnemy":                "ENABLE_INVADER_ENEMY",
	"bEnableNonLoginPenalty":             "ENABLE_NON_LOGIN_PENALTY",
	"WorkSpeedRate":                      "WORK_SPEED_RATE",
	"PalEggDefaultHatchingTime":          "PAL_EGG_DEFAULT_HATCHING_TIME",
	"GuildPlayerMaxNum":                  "GUILD_PLAYER_MAX_NUM",
	"BaseCampMaxNum":                     "BASE_CAMP_MAX_NUM",
	"BaseCampWorkerMaxNum":               "BASE_CAMP_WORKER_MAX_NUM",
	"bIsPvP":                             "IS_PVP",
	"bHardcore":                          "HARDCORE",
	"bPalLost":                           "PAL_LOST",
	"bEnableFastTravel":                  "ENABLE_FAST_TRAVEL",
	"bIsUseBackupSaveData":               "USE_BACKUP_SAVE_DATA",
}

type derivedNames struct {
	pvcName        string
	configMapName  string
	deploymentName string
	serviceName    string
	envoyService   string
	gatewayName    string
	envoyProxyName string
	gameUDPRoute   string
	queryUDPRoute  string
	rconTCPRoute   string
	restTCPRoute   string
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

func credentialsSecretName(server *palworldv1alpha1.PalworldServer) string {
	if server.Spec.CredentialsSecretName != "" {
		return server.Spec.CredentialsSecretName
	}
	return server.Name + credentialsSecretSuffix
}

func defaultSecretKeySelector(name, key string) *corev1.SecretKeySelector {
	return &corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{Name: name},
		Key:                  key,
	}
}

func generatePassword() (string, error) {
	buf := make([]byte, generatedPasswordBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func serverImage(spec palworldv1alpha1.PalworldServerSpec) string {
	if spec.ServerImage != "" {
		return spec.ServerImage
	}
	return defaultServerImage
}

func imageRepository(spec palworldv1alpha1.PalworldServerSpec) string {
	if spec.Update.ImageRepository != "" {
		return strings.TrimSuffix(spec.Update.ImageRepository, "/")
	}
	return defaultImageRepository
}

func updateCheckInterval(spec palworldv1alpha1.PalworldServerSpec) time.Duration {
	if spec.Update.CheckInterval == "" {
		return defaultUpdateCheckInterval
	}
	d, err := time.ParseDuration(spec.Update.CheckInterval)
	if err != nil || d <= 0 {
		return defaultUpdateCheckInterval
	}
	return d
}

func updateOnlyWhenEmpty(spec palworldv1alpha1.PalworldServerSpec) bool {
	return boolValue(spec.Update.OnlyWhenEmpty, true)
}

// dedicatedServerName returns the world pin from spec, else observed status.
func dedicatedServerName(server *palworldv1alpha1.PalworldServer) string {
	if server.Spec.DedicatedServerName != "" {
		return server.Spec.DedicatedServerName
	}
	return server.Status.DedicatedServerName
}

func buildGameUserSettingsINI(name string) string {
	name = strings.TrimSpace(name)
	return fmt.Sprintf("[/Script/Pal.PalGameLocalSettings]\nDedicatedServerName=%s\n", name)
}

func seedSettingsScript() string {
	return fmt.Sprintf(
		`mkdir -p /saves/Config/LinuxServer && cp /settings/%s /saves/%s && if [ -f /settings/%s ]; then cp /settings/%s /saves/%s; fi`,
		settingsConfigKey, settingsRelativePath,
		gameUserSettingsKey, gameUserSettingsKey, gameUserSettingsRelPath,
	)
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

func boolINI(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

// formatOptionSettingValue turns a map value into an OptionSettings literal.
// Callers may pass bare tokens (None), numbers, booleans, quoted strings, or
// parenthesized lists — unknown shapes are quoted and escaped.
func formatOptionSettingValue(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return `""`
	}
	switch strings.ToLower(v) {
	case boolStrTrueLower:
		return "True"
	case boolStrFalseLower:
		return "False"
	}
	if iniNumberRE.MatchString(v) {
		return v
	}
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		return v
	}
	if strings.HasPrefix(v, "(") {
		return v
	}
	if isINIBareToken(v) {
		return v
	}
	return `"` + escapeINI(v) + `"`
}

func isINIBareToken(value string) bool {
	runes := []rune(value)
	if len(runes) == 0 {
		return false
	}
	if !unicode.IsLetter(runes[0]) && runes[0] != '_' {
		return false
	}
	for _, r := range runes[1:] {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			continue
		}
		return false
	}
	return true
}

func buildPalWorldSettingsINI(spec palworldv1alpha1.PalworldServerSpec, adminPassword, serverPassword string) string {
	merged := make(map[string]string, len(spec.OptionSettings)+len(managementOptionKeys))
	for key, value := range spec.OptionSettings {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		merged[key] = formatOptionSettingValue(value)
	}

	name := spec.ServerName
	if name == "" {
		name = "Palworld Server"
	}
	merged["ServerName"] = `"` + escapeINI(name) + `"`
	merged["ServerDescription"] = `"` + escapeINI(spec.ServerDescription) + `"`
	merged["ServerPlayerMaxNum"] = fmt.Sprintf("%d", maxPlayers(spec))
	merged["AdminPassword"] = `"` + escapeINI(adminPassword) + `"`
	merged["ServerPassword"] = `"` + escapeINI(serverPassword) + `"`
	merged["PublicPort"] = fmt.Sprintf("%d", publicPort(spec))
	merged["PublicIP"] = `"` + escapeINI(publicIP(spec)) + `"`
	merged["RCONEnabled"] = boolINI(rconEnabled(spec))
	merged["RCONPort"] = fmt.Sprintf("%d", rconPort(spec))
	merged["RESTAPIEnabled"] = boolINI(restEnabled(spec))
	merged["RESTAPIPort"] = fmt.Sprintf("%d", restPort(spec))
	merged["CrossplayPlatforms"] = `"` + escapeINI(crossplayPlatforms(spec)) + `"`

	opts := make([]string, 0, len(merged))
	seen := make(map[string]struct{}, len(managementOptionKeys))
	for _, key := range managementOptionKeys {
		if value, ok := merged[key]; ok {
			opts = append(opts, key+"="+value)
			seen[key] = struct{}{}
		}
	}
	extras := make([]string, 0, len(merged))
	for key := range merged {
		if _, ok := seen[key]; ok {
			continue
		}
		extras = append(extras, key)
	}
	sort.Strings(extras)
	for _, key := range extras {
		opts = append(opts, key+"="+merged[key])
	}
	return fmt.Sprintf("[/Script/Pal.PalGameWorldSettings]\nOptionSettings=(%s)\n", strings.Join(opts, ","))
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
	env = append(env, communityOptionSettingsEnv(spec)...)
	return env
}

// communityOptionSettingsEnv maps known optionSettings keys to community-image
// env vars. Management CR fields already set above win for overlapping concerns.
func communityOptionSettingsEnv(spec palworldv1alpha1.PalworldServerSpec) []corev1.EnvVar {
	if len(spec.OptionSettings) == 0 {
		return nil
	}
	keys := make([]string, 0, len(spec.OptionSettings))
	for key := range spec.OptionSettings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var env []corev1.EnvVar
	for _, key := range keys {
		envName, ok := communityOptionEnv[key]
		if !ok {
			continue
		}
		env = append(env, corev1.EnvVar{
			Name:  envName,
			Value: communityEnvValue(spec.OptionSettings[key]),
		})
	}
	return env
}

func communityEnvValue(value string) string {
	v := strings.TrimSpace(value)
	switch strings.ToLower(v) {
	case boolStrTrueLower:
		return boolStrTrueLower
	case boolStrFalseLower:
		return boolStrFalseLower
	}
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		return v[1 : len(v)-1]
	}
	return v
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
