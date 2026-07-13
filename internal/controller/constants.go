package controller

const (
	finalizerName = "palworld.dataknife.ai/finalizer"

	defaultServerImage              = "ghcr.io/pocketpairjp/palserver:latest"
	defaultGatewayClassName         = "envoy"
	defaultGamePort           int32 = 8211
	defaultQueryPort          int32 = 27015
	defaultRCONPort           int32 = 25575
	defaultRESTPort           int32 = 8212
	defaultMaxPlayers         int32 = 4
	defaultStorageSize              = "50Gi"
	defaultTerminationGrace   int64 = 60
	defaultCrossplayPlatforms       = "(Steam,Xbox,PS5,Mac)"

	containerUser = int64(1000)

	officialSavedMountPath  = "/pal/Package/Pal/Saved"
	communitySavedMountPath = "/palworld"
	settingsConfigKey       = "PalWorldSettings.ini"
	settingsRelativePath    = "Config/LinuxServer/PalWorldSettings.ini"

	gatewayListenerGameUDP  = "game-udp"
	gatewayListenerQueryUDP = "query-udp"
	gatewayListenerRESTTCP  = "rest-tcp"

	initContainerImage = "busybox:1.37"

	secretKeyAdminPassword  = "admin-password"
	secretKeyServerPassword = "server-password"
	credentialsSecretSuffix = "-secrets"
	generatedPasswordBytes  = 24
)
