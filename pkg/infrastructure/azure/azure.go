package azure

import (
	"github.com/openshift/installer/pkg/infrastructure/clusterapi"
)

// const (
// 	tfVarsFileName         = "terraform.tfvars.json"
// 	tfPlatformVarsFileName = "terraform.platform.auto.tfvars.json"
// 	clusterOutputFileName  = "cluster.azuresdk.vars.json"

// 	defaultDescription = "Created by Openshift Installer"
// 	ownedTagKey        = "kubernetes.io/cluster/%s"
// 	ownedTagValue      = "owned"
// )

// InfraProvider is the Azure SDK infra provider.
type InfraProvider struct{}

// InitializeProvider initializes the Azure SDK provider.
func InitializeProvider() clusterapi.Provider {
	return &InfraProvider{}
}

// Name returns the name of the provider.
func (p *InfraProvider) Name() string {
	return "azure"
}
