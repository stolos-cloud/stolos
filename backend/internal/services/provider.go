package services

// Provider defines the interface for cloud provider services
type Provider interface {
	GetTerraformBackendConfig() (map[string]string, error)

	GetProviderName() string

	// returns the configured region/location for the provider
	GetRegion() string

	// checks if the provider is properly configured
	IsConfigured() bool
}
