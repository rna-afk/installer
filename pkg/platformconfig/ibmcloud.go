package platformconfig

import (
	"github.com/pkg/errors"

	"github.com/openshift/installer/pkg/asset/installconfig"
	config "github.com/openshift/installer/pkg/asset/installconfig/ibmcloud"
	"github.com/openshift/installer/pkg/types"
)

type IBMCloudPlatformConfig struct {
	PlatformConfig
}

func (a *IBMCloudPlatformConfig) CredsCheck(ic *installconfig.InstallConfig) error {
	_, err := config.NewClient()
	if err != nil {
		return errors.Wrap(err, "creating IBM Cloud session")
	}
	return nil
}

func (a *IBMCloudPlatformConfig) SetPlatform(p *types.Platform) error {
	var err error
	p.IBMCloud, err = config.Platform()
	return err
}

func (a *IBMCloudPlatformConfig) PermissionCheck(ic *installconfig.InstallConfig) error {
	return nil
}

func (a *IBMCloudPlatformConfig) ProvisionCheck(ic *installconfig.InstallConfig) error {
	client, err := config.NewClient()
	if err != nil {
		return err
	}
	return config.ValidatePreExitingPublicDNS(client, ic.Config, ic.IBMCloud)
}

func (a *IBMCloudPlatformConfig) InstallConfigCheck(ic *installconfig.InstallConfig) error {
	ic.IBMCloud = config.NewMetadata(ic.Config.BaseDomain)
	client, err := config.NewClient()
	if err != nil {
		return err
	}
	return config.Validate(client, ic.Config)
}

func (a *IBMCloudPlatformConfig) BaseDomainCheck(bd *installconfig.BaseDomain, p *installconfig.Platform) error {
	zone, err := config.GetDNSZone()
	if err != nil {
		return err
	}
	bd.BaseDomain = zone.Name
	return nil
}

func init() {
	MapPlatforms["ibmcloud"] = &IBMCloudPlatformConfig{}
}
