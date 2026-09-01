package controller

import "time"

const (
	finalizerName = "palworld.dataknife.ai/finalizer"

	defaultServerImage               = "ghcr.io/pocketpairjp/palserver:latest"
	defaultImageRepository           = "ghcr.io/pocketpairjp/palserver"
	defaultServerManagerImage        = "harbor.dataknife.net/library/palworld-operator:latest"
	defaultGatewayClassName          = "envoy"
	defaultGamePort            int32 = 8211
	defaultQueryPort           int32 = 27015
	defaultRCONPort            int32 = 25575
	defaultRESTPort            int32 = 8212
	defaultServerManagerPort   int32 = 8088
	defaultMaxPlayers          int32 = 4
	defaultStorageSize               = "50Gi"
	defaultModsStorageSize           = "10Gi"
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

	officialSavedMountPath     = "/pal/Package/Pal/Saved"
	communitySavedMountPath    = "/palworld"
	officialModsMountPath      = "/pal/Package/Mods"
	communityModsMountPath     = "/palworld/Mods"
	officialPaksRoot           = "/pal/Package/Pal/Content/Paks"
	communityPaksRoot          = "/palworld/Pal/Content/Paks"
	paksWorkshopModsDir        = "~WorkshopMods"
	paksLogicModsDir           = "LogicMods"
	paksOverlayWorkshopSub     = "paks/~WorkshopMods"
	paksOverlayLogicSub        = "paks/LogicMods"
	volumeSaves                = "saves"
	volumeSettings             = "settings"
	volumeMods                 = "mods"
	modsPVCSuffix              = "-mods"
	serverManagerSASuffix      = "-manager"
	legacyModManagerSASuffix   = "-mod-manager"
	containerServerManager     = "server-manager"
	portNameServerManager      = "server-manager"
	serverManagerModsPath      = "/mods"
	serverManagerSavesPath     = "/saves"
	serverManagerBinary        = "/server-manager"
	envServerManagerRoot       = "SERVER_MANAGER_ROOT"
	envServerManagerSaves      = "SERVER_MANAGER_SAVES"
	envServerManagerListen     = "SERVER_MANAGER_LISTEN"
	envServerManagerUser       = "SERVER_MANAGER_USER"
	envServerManagerPassword   = "SERVER_MANAGER_PASSWORD"
	envServerManagerNamespace  = "SERVER_MANAGER_NAMESPACE"
	envServerManagerDeployment = "SERVER_MANAGER_DEPLOYMENT"
	envServerManagerRESTBase   = "SERVER_MANAGER_REST_BASE"
	workshopSubdir             = "Workshop"
	workshopDirArgPrefix       = "-workshopdir="
	seedModsInitName           = "seed-mods"
	settingsConfigKey          = "PalWorldSettings.ini"
	settingsRelativePath       = "Config/LinuxServer/PalWorldSettings.ini"
	gameUserSettingsKey        = "GameUserSettings.ini"
	gameUserSettingsRelPath    = "Config/LinuxServer/GameUserSettings.ini"
	palModSettingsKey          = "PalModSettings.ini"
	palModSettingsSection      = "[PalModSettings]"
	seedModsMountPath          = serverManagerModsPath

	gatewayListenerGameUDP           = "game-udp"
	gatewayListenerQueryUDP          = "query-udp"
	gatewayListenerRESTTCP           = "rest-tcp"
	gatewayListenerServerManagerHTTP = "server-manager-http"

	palworldAdminUser = "admin"

	initContainerImage = "busybox:1.37"

	secretKeyAdminPassword  = "admin-password"
	secretKeyServerPassword = "server-password"
	credentialsSecretSuffix = "-secrets"
	generatedPasswordBytes  = 24

	// Boolean string forms used when parsing optionSettings / community env.
	boolStrTrueLower  = "true"
	boolStrFalseLower = "false"
	// INI / OptionSettings boolean and bare-token literals (goconst).
	boolStrTrueINI  = "True"
	boolStrFalseINI = "False"
	iniBareNone     = "None"

	// OptionSettings keys referenced from maps and tests (goconst).
	optionKeyExpRate               = "ExpRate"
	optionKeyWorkSpeedRate         = "WorkSpeedRate"
	optionKeyEnableNonLoginPenalty = "bEnableNonLoginPenalty"
)
