package domain

// RancherServices contains all services needed for Rancher operations.
type RancherServices struct {
	RancherAuthenticator RancherAuthenticator
	ClusterLister        ClusterLister
	KubeconfigFetcher    KubeconfigFetcher
	KubeconfigWriter     KubeconfigWriter
	KubeconfigMerger     KubeconfigMerger
	KubeconfigCleaner    KubeconfigCleaner
	KubeconfigSyncer     KubeconfigSyncer
}

// RancherServiceFactory creates Rancher-related services with the appropriate HTTP configuration.
type RancherServiceFactory interface {
	// CreateSecureServices creates all Rancher services with secure TLS verification.
	CreateSecureServices(configProvider ConfigProvider) RancherServices
	// CreateInsecureServices creates all Rancher services that skip TLS verification.
	CreateInsecureServices(configProvider ConfigProvider) RancherServices
}
