package azure

import (
	"context"
	"fmt"

	"github.com/openshift/installer/pkg/infrastructure/clusterapi"
	"github.com/sirupsen/logrus"
)

// const (
// 	tfVarsFileName         = "terraform.tfvars.json"
// 	tfPlatformVarsFileName = "terraform.platform.auto.tfvars.json"
// 	clusterOutputFileName  = "cluster.azuresdk.vars.json"

// 	defaultDescription = "Created by Openshift Installer"
// 	ownedTagKey        = "kubernetes.io/cluster/%s"
// 	ownedTagValue      = "owned"
// )

// InfraProvider is the AWS SDK infra provider.
type InfraProvider struct{}

// InitializeProvider initializes the AWS SDK provider.
func InitializeProvider() clusterapi.Provider {
	return &InfraProvider{}
}

func (p *InfraProvider) Name() string {
	return "azure"
}

func (p *InfraProvider) PreProvision(ctx context.Context, in clusterapi.PreProvisionInput) error {
	logrus.Infof("Creating User Assigned Identity")
	identityName := fmt.Sprintf("%s-identity", in.InfraID)

	return nil
}
