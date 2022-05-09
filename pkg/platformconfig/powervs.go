package platformconfig

import (
	"github.com/pkg/errors"

	"github.com/openshift/installer/pkg/asset/installconfig"
	config "github.com/openshift/installer/pkg/asset/installconfig/powervs"
	"github.com/openshift/installer/pkg/types"
)

type PowerVSCloudPlatformConfig struct {
	PlatformConfig
}

func (a *PowerVSCloudPlatformConfig) CredsCheck(ic *installconfig.InstallConfig) error {
	_, err := config.NewClient()
	if err != nil {
		return errors.Wrap(err, "creating IBM Cloud session")
	}
	return nil
}

func (a *PowerVSCloudPlatformConfig) SetPlatform(p *types.Platform) error {
	var err error
	p.PowerVS, err = config.Platform()
	return err
}

func (a *PowerVSCloudPlatformConfig) PermissionCheck(ic *installconfig.InstallConfig) error {
	return nil
}

func (a *PowerVSCloudPlatformConfig) ProvisionCheck(ic *installconfig.InstallConfig) error {
	client, err := config.NewClient()
	if err != nil {
		return err
	}
	return config.ValidatePreExistingPublicDNS(client, ic.Config, ic.PowerVS)
}

func (a *PowerVSCloudPlatformConfig) InstallConfigCheck(ic *installconfig.InstallConfig) error {
	return config.Validate(ic.Config)
}

func (a *PowerVSCloudPlatformConfig) BaseDomainCheck(bd *installconfig.BaseDomain, p *installconfig.Platform) error {
	zone, err := config.GetDNSZone()
	if err != nil {
		return err
	}
	bd.BaseDomain = zone.Name
	return nil
}

func init() {
	MapPlatforms["powervs"] = &PowerVSCloudPlatformConfig{}
}
