package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PhasePending = "Pending"
	PhaseRunning = "Running"
	PhaseFailed  = "Failed"
)

// GatewayConfig configures Envoy Gateway exposure for Palworld game traffic.
// Matches the DataKnife prd-apps game-servers pattern used by windrose-operator.
type GatewayConfig struct {
	// Address is the external IP assigned to this server (Kube-VIP or MetalLB).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^([0-9]{1,3}\.){3}[0-9]{1,3}$`
	Address string `json:"address"`

	// ClassName is the GatewayClass used for the Envoy Gateway controller.
	// +kubebuilder:default="envoy"
	// +optional
	ClassName string `json:"className,omitempty"`

	// GatewayName overrides the Gateway resource name.
	// Default: {base-name}-gateway where base-name strips a trailing "-server" suffix.
	// +optional
	GatewayName string `json:"gatewayName,omitempty"`

	// EnvoyProxyName overrides the EnvoyProxy resource name.
	// Default: game-{base-name}-kubevip.
	// +optional
	EnvoyProxyName string `json:"envoyProxyName,omitempty"`

	// ExternalTrafficPolicy for the Envoy LoadBalancer service.
	// +kubebuilder:validation:Enum=Cluster;Local
	// +kubebuilder:default=Cluster
	// +optional
	ExternalTrafficPolicy corev1.ServiceExternalTrafficPolicy `json:"externalTrafficPolicy,omitempty"`
}

// RCONConfig controls remote console access (required for graceful Docker stop/save).
type RCONConfig struct {
	// Enabled toggles RCON. Default true for graceful shutdown support.
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// Port is the RCON TCP listen port.
	// +kubebuilder:default=25575
	// +kubebuilder:validation:Minimum=1024
	// +kubebuilder:validation:Maximum=65535
	// +optional
	Port int32 `json:"port,omitempty"`
}

// RESTAPIConfig controls the Palworld REST API (default port 8212).
// Prefer ClusterIP-only exposure; do not public-route unless intentionally secured.
type RESTAPIConfig struct {
	// Enabled toggles the REST API.
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// Port is the REST API TCP listen port.
	// +kubebuilder:default=8212
	// +kubebuilder:validation:Minimum=1024
	// +kubebuilder:validation:Maximum=65535
	// +optional
	Port int32 `json:"port,omitempty"`

	// ExposeViaGateway when true creates a TCPRoute for the REST port.
	// Default false — keep admin API internal.
	// +kubebuilder:default=false
	// +optional
	ExposeViaGateway *bool `json:"exposeViaGateway,omitempty"`
}

// CommunityConfig controls Steam community server browser listing.
type CommunityConfig struct {
	// Enabled shows the server in the community browser (use with a password).
	// +kubebuilder:default=false
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// PublicIP overrides auto-detected public IP (often set to gateway.address).
	// +optional
	PublicIP string `json:"publicIP,omitempty"`

	// PublicPort overrides advertised public port (usually gamePort).
	// +optional
	PublicPort int32 `json:"publicPort,omitempty"`
}

// PalworldServerSpec defines the desired state of a Palworld dedicated game server.
// Default image is the official Pocketpair package (ghcr.io/pocketpairjp/palserver).
// Settings map to PalWorldSettings.ini / CLI args (official) or community-image
// environment variables. See docs/PALWORLD_SERVER.md and
// https://docs.palworldgame.com/settings-and-operation/configuration/
type PalworldServerSpec struct {
	// ServerImage is the Palworld dedicated server container image.
	// Defaults to the official Pocketpair image. Override with a Harbor mirror
	// or a community image (e.g. thijsvanloef/palworld-server-docker) if needed.
	// +kubebuilder:default="ghcr.io/pocketpairjp/palserver:latest"
	// +optional
	ServerImage string `json:"serverImage,omitempty"`

	// ImagePullPolicy for the game server container.
	// +kubebuilder:default=IfNotPresent
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets for private registries.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// NodeSelector pins the game server pod to specific nodes.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Gateway configures Envoy Gateway exposure (required).
	Gateway GatewayConfig `json:"gateway"`

	// ServerName is the display name for the dedicated server.
	// +optional
	ServerName string `json:"serverName,omitempty"`

	// ServerDescription is shown in the server browser.
	// +optional
	ServerDescription string `json:"serverDescription,omitempty"`

	// MaxPlayers is the maximum number of simultaneous players (1–32).
	// When spec.resources is unset, pod CPU/memory are auto-selected from this value.
	// +kubebuilder:default=4
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=32
	// +optional
	MaxPlayers int32 `json:"maxPlayers,omitempty"`

	// GamePort is the primary UDP game port.
	// +kubebuilder:default=8211
	// +kubebuilder:validation:Minimum=1024
	// +kubebuilder:validation:Maximum=65535
	// +optional
	GamePort int32 `json:"gamePort,omitempty"`

	// QueryPort is the Steam query UDP port.
	// +kubebuilder:default=27015
	// +kubebuilder:validation:Minimum=1024
	// +kubebuilder:validation:Maximum=65535
	// +optional
	QueryPort int32 `json:"queryPort,omitempty"`

	// RCON configures remote administration.
	// +optional
	RCON RCONConfig `json:"rcon,omitempty"`

	// RESTAPI configures the Palworld REST API.
	// +optional
	RESTAPI RESTAPIConfig `json:"restAPI,omitempty"`

	// Community configures community server browser listing.
	// +optional
	Community CommunityConfig `json:"community,omitempty"`

	// Multithreading enables multi-threaded server mode (~4 threads useful).
	// +kubebuilder:default=true
	// +optional
	Multithreading *bool `json:"multithreading,omitempty"`

	// UpdateOnBoot updates/installs server files on container start.
	// Relevant primarily for community SteamCMD-based images; the official
	// Pocketpair image is versioned via the image tag.
	// +kubebuilder:default=true
	// +optional
	UpdateOnBoot *bool `json:"updateOnBoot,omitempty"`

	// CrossplayPlatforms lists allowed platforms, e.g. "(Steam,Xbox,PS5,Mac)".
	// +optional
	CrossplayPlatforms string `json:"crossplayPlatforms,omitempty"`

	// GenerateSecrets when true creates an Opaque Secret with random strong
	// passwords for keys server-password (join) and admin-password (RCON/admin)
	// if the Secret is missing or those keys are empty. Existing non-empty keys
	// are never overwritten. Secret name defaults to {metadata.name}-secrets
	// (override with credentialsSecretName). When false/omitted, provide
	// adminPasswordSecretRef and serverPasswordSecretRef yourself (bring-your-own).
	// +optional
	GenerateSecrets bool `json:"generateSecrets,omitempty"`

	// CredentialsSecretName overrides the auto-generated Secret name when
	// generateSecrets is true. Default: {metadata.name}-secrets.
	// +optional
	CredentialsSecretName string `json:"credentialsSecretName,omitempty"`

	// AdminPasswordSecretRef points to a Secret key used as ADMIN_PASSWORD.
	// Required for bring-your-own credentials; optional when generateSecrets is true
	// (defaults to credentials Secret key admin-password).
	// +optional
	AdminPasswordSecretRef *corev1.SecretKeySelector `json:"adminPasswordSecretRef,omitempty"`

	// ServerPasswordSecretRef points to a Secret key used as SERVER_PASSWORD.
	// Required for bring-your-own credentials; optional when generateSecrets is true
	// (defaults to credentials Secret key server-password).
	// +optional
	ServerPasswordSecretRef *corev1.SecretKeySelector `json:"serverPasswordSecretRef,omitempty"`

	// StorageSize is the PVC capacity for world saves (official mount:
	// /pal/Package/Pal/Saved; community image typically /palworld).
	// +kubebuilder:default="50Gi"
	// +optional
	StorageSize string `json:"storageSize,omitempty"`

	// StorageClassName selects the StorageClass for the saves PVC.
	// +optional
	StorageClassName string `json:"storageClassName,omitempty"`

	// Resources overrides auto-selected CPU/memory. When unset, tiers derive from maxPlayers.
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// TerminationGracePeriodSeconds allows graceful RCON save on stop.
	// +kubebuilder:default=60
	// +optional
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`
}

// PalworldServerStatus defines the observed state of PalworldServer.
type PalworldServerStatus struct {
	// Phase is Pending, Running, or Failed.
	// +optional
	Phase string `json:"phase,omitempty"`

	// Ready is true when the game server pod is ready.
	// +optional
	Ready bool `json:"ready,omitempty"`

	// ConnectionAddress is the IP clients should use.
	// +optional
	ConnectionAddress string `json:"connectionAddress,omitempty"`

	// ConnectionPort is the UDP game port clients should use.
	// +optional
	ConnectionPort int32 `json:"connectionPort,omitempty"`

	// Message is a human-readable status detail.
	// +optional
	Message string `json:"message,omitempty"`

	// CredentialsSecretName is the Secret that holds join/admin passwords.
	// Never contains plaintext passwords — use kubectl to read Secret data.
	// +optional
	CredentialsSecretName string `json:"credentialsSecretName,omitempty"`

	// CredentialsGenerated is true when spec.generateSecrets created or manages
	// the credentials Secret (passwords are not written into status).
	// +optional
	CredentialsGenerated bool `json:"credentialsGenerated,omitempty"`

	// ObservedGeneration is the last reconciled generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=ps;palworld
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Address",type=string,JSONPath=`.status.connectionAddress`
// +kubebuilder:printcolumn:name="Port",type=integer,JSONPath=`.status.connectionPort`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// PalworldServer is the Schema for the palworldservers API.
type PalworldServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PalworldServerSpec   `json:"spec,omitempty"`
	Status PalworldServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PalworldServerList contains a list of PalworldServer.
type PalworldServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PalworldServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PalworldServer{}, &PalworldServerList{})
}
