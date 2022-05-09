package platformconfig

import (
	"github.com/openshift/installer/pkg/asset/installconfig"
	config "github.com/openshift/installer/pkg/asset/installconfig/baremetal"
	"github.com/openshift/installer/pkg/types"
)

type BaremetalCloudPlatformConfig struct {
	PlatformConfig
}

func (a *BaremetalCloudPlatformConfig) CredsCheck(ic *installconfig.InstallConfig) error {
	return nil
}

func (a *BaremetalCloudPlatformConfig) SetPlatform(p *types.Platform) error {
	var err error
	p.BareMetal, err = config.Platform()
	return err
}

func (a *BaremetalCloudPlatformConfig) PermissionCheck(ic *installconfig.InstallConfig) error {
	return nil
}

func (a *BaremetalCloudPlatformConfig) ProvisionCheck(ic *installconfig.InstallConfig) error {
	return config.ValidateProvisioning(ic.Config)
}

func (a *BaremetalCloudPlatformConfig) InstallConfigCheck(ic *installconfig.InstallConfig) error {
	return nil
}

func (a *BaremetalCloudPlatformConfig) BaseDomainCheck(bd *installconfig.BaseDomain, p *installconfig.Platform) error {
	return nil
}

func init() {
	MapPlatforms["baremetal"] = &BaremetalCloudPlatformConfig{}
}
