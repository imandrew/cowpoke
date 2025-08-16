package app

import (
	"log/slog"
	"time"

	"cowpoke/internal/adapters/http"
	"cowpoke/internal/domain"
	"cowpoke/internal/services/auth"
	"cowpoke/internal/services/kubeconfig"
	"cowpoke/internal/services/rancher"
	"cowpoke/internal/services/sync"
)

const (
	// defaultHTTPTimeout is the default timeout for HTTP requests to Rancher.
	defaultHTTPTimeout = 30 * time.Second
)

// RancherServiceFactory implements the RancherServiceFactory interface.
type RancherServiceFactory struct {
	logger     *slog.Logger
	fileSystem domain.FileSystemAdapter
}

// NewRancherServiceFactory creates a new Rancher service factory.
func NewRancherServiceFactory(logger *slog.Logger, fileSystem domain.FileSystemAdapter) *RancherServiceFactory {
	return &RancherServiceFactory{
		logger:     logger,
		fileSystem: fileSystem,
	}
}

// CreateSecureServices creates all Rancher services with secure TLS verification.
func (rsf *RancherServiceFactory) CreateSecureServices(
	configProvider domain.ConfigProvider,
) domain.RancherServices {
	return rsf.createServices(false, configProvider) // secure TLS
}

// CreateInsecureServices creates all Rancher services that skip TLS verification.
func (rsf *RancherServiceFactory) CreateInsecureServices(
	configProvider domain.ConfigProvider,
) domain.RancherServices {
	return rsf.createServices(true, configProvider) // insecure TLS
}

// createServices is the internal implementation for creating services with the specified TLS configuration.
func (rsf *RancherServiceFactory) createServices(
	insecureSkipTLS bool,
	configProvider domain.ConfigProvider,
) domain.RancherServices {
	// Create HTTP client with the specified configuration
	httpAdapter := http.NewAdapter(defaultHTTPTimeout, insecureSkipTLS, rsf.logger)

	// Create services that depend on the HTTP client
	rancherAuthenticator := auth.NewAuthenticator(httpAdapter, rsf.logger)
	clusterLister := rancher.NewClusterLister(httpAdapter, rsf.logger)
	kubeconfigFetcher := rancher.NewKubeconfigFetcher(httpAdapter, rsf.logger)

	// Create kubeconfig manager for the sync operations
	kubeconfigDir, err := configProvider.GetKubeconfigDir()
	if err != nil {
		rsf.logger.Error("Failed to get kubeconfig directory", "error", err)
		// Return minimal services without kubeconfig functionality
		return domain.RancherServices{
			RancherAuthenticator: rancherAuthenticator,
			ClusterLister:        clusterLister,
			KubeconfigFetcher:    kubeconfigFetcher,
		}
	}
	kubeconfigManager, err := kubeconfig.NewManager(rsf.fileSystem, kubeconfigDir, rsf.logger)
	if err != nil {
		rsf.logger.Error("Failed to create kubeconfig manager", "error", err)
		// Return minimal services without kubeconfig functionality
		return domain.RancherServices{
			RancherAuthenticator: rancherAuthenticator,
			ClusterLister:        clusterLister,
			KubeconfigFetcher:    kubeconfigFetcher,
		}
	}

	// Create sync manager that orchestrates all services
	kubeconfigSyncer := sync.NewManager(
		rancherAuthenticator,
		clusterLister,
		kubeconfigFetcher,
		kubeconfigManager, // KubeconfigWriter
		kubeconfigManager, // KubeconfigMerger
		kubeconfigManager, // KubeconfigCleaner
		configProvider,
		rsf.logger,
	)

	return domain.RancherServices{
		RancherAuthenticator: rancherAuthenticator,
		ClusterLister:        clusterLister,
		KubeconfigFetcher:    kubeconfigFetcher,
		KubeconfigWriter:     kubeconfigManager,
		KubeconfigMerger:     kubeconfigManager,
		KubeconfigCleaner:    kubeconfigManager,
		KubeconfigSyncer:     kubeconfigSyncer,
	}
}
