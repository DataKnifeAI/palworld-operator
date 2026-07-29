package controller

import "time"

const (
	finalizerName = "palworld.dataknife.ai/finalizer"

	defaultServerImage               = "ghcr.io/pocketpairjp/palserver:latest"
	defaultImageRepository           = "ghcr.io/pocketpairjp/palserver"
	defaultGatewayClassName          = "envoy"
	defaultGamePort            int32 = 8211
	defaultQueryPort           int32 = 27015
	defaultRCONPort            int32 = 25575
	defaultRESTPort            int32 = 8212
	defaultMaxPlayers          int32 = 4
	defaultStorageSize               = "50Gi"
	defaultTerminationGrace    int64 = 60
	defaultCrossplayPlatforms        = "(Steam,Xbox,PS5,Mac)"
	defaultUpdateCheckInterval       = 6 * time.Hour
	defaultUpdateTimeZone            = "UTC"
	defaultNotifyLeadTime            = 60 * time.Minute // max of default multi-stage schedule
	tagCacheTTL                      = 30 * time.Minute
	updateRequeueBusy                = 2 * time.Minute
	updateRequeueSoon                = 30 * time.Second

	// Named notify-schedule keys (goconst); default timeline uses all of these.
	notifyKey60m = "60m"
	notifyKey30m = "30m"
	notifyKey15m = "15m"
	notifyKey5m  = "5m"
	notifyKey1m  = "1m"
	notifyKey30s = "30s"
	notifyKey10s = "10s"

	containerUser = int64(1000)

	officialSavedMountPath  = "/pal/Package/Pal/Saved"
	communitySavedMountPath = "/palworld"
	volumeSaves             = "saves"
	volumeSettings          = "settings"
	settingsConfigKey       = "PalWorldSettings.ini"
	settingsRelativePath    = "Config/LinuxServer/PalWorldSettings.ini"
	gameUserSettingsKey     = "GameUserSettings.ini"
	gameUserSettingsRelPath = "Config/LinuxServer/GameUserSettings.ini"

	gatewayListenerGameUDP  = "game-udp"
	gatewayListenerQueryUDP = "query-udp"
	gatewayListenerRESTTCP  = "rest-tcp"

	initContainerImage = "busybox:1.37"

	secretKeyAdminPassword  = "admin-password"
	secretKeyServerPassword = "server-password"
	credentialsSecretSuffix = "-secrets"
	generatedPasswordBytes  = 24
)
