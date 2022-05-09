package platformconfig

import (
	"context"

	"github.com/pkg/errors"

	"github.com/openshift/installer/pkg/asset/installconfig"
	config "github.com/openshift/installer/pkg/asset/installconfig/aws"
	"github.com/openshift/installer/pkg/types"
	"github.com/openshift/installer/pkg/types/aws"
)

type AWSCloudPlatformConfig struct {
	PlatformConfig
}

func (a *AWSCloudPlatformConfig) CredsCheck(ic *installconfig.InstallConfig) error {
	_, err := ic.AWS.Session(context.TODO())
	return err
}

func (a *AWSCloudPlatformConfig) SetPlatform(p *types.Platform) error {
	var err error
	p.AWS, err = config.Platform()
	return err
}

func (a *AWSCloudPlatformConfig) PermissionCheck(ic *installconfig.InstallConfig) error {
	permissionGroups := []config.PermissionGroup{config.PermissionCreateBase}
	usingExistingVPC := len(ic.Config.AWS.Subnets) != 0

	if !usingExistingVPC {
		permissionGroups = append(permissionGroups, config.PermissionCreateNetworking)
	}

	// Add delete permissions for non-C2S installs.
	if !aws.IsSecretRegion(ic.Config.AWS.Region) {
		permissionGroups = append(permissionGroups, config.PermissionDeleteBase)
		if usingExistingVPC {
			permissionGroups = append(permissionGroups, config.PermissionDeleteSharedNetworking)
		} else {
			permissionGroups = append(permissionGroups, config.PermissionDeleteNetworking)
		}
	}

	ssn, err := ic.AWS.Session(context.TODO())
	if err != nil {
		return err
	}

	err = config.ValidateCreds(ssn, permissionGroups, ic.Config.Platform.AWS.Region)
	if err != nil {
		return errors.Wrap(err, "validate AWS credentials")
	}
	return nil
}

func (a *AWSCloudPlatformConfig) ProvisionCheck(ic *installconfig.InstallConfig) error {
	session, err := ic.AWS.Session(context.TODO())
	if err != nil {
		return err
	}
	return config.ValidateForProvisioning(session, ic.Config, ic.AWS)
}

func (a *AWSCloudPlatformConfig) InstallConfigCheck(ic *installconfig.InstallConfig) error {
	ic.AWS = config.NewMetadata(ic.Config.Platform.AWS.Region, ic.Config.Platform.AWS.Subnets, ic.Config.AWS.ServiceEndpoints)
	return config.Validate(context.TODO(), ic.AWS, ic.Config)
}

func (a *AWSCloudPlatformConfig) BaseDomainCheck(bd *installconfig.BaseDomain, p *installconfig.Platform) error {
	var err error
	bd.BaseDomain, err = config.GetBaseDomain()
	return err
}

func init() {
	MapPlatforms["aws"] = &AWSCloudPlatformConfig{}
}
