package platformconfig

import (
	"github.com/openshift/installer/pkg/asset/installconfig"
	config "github.com/openshift/installer/pkg/asset/installconfig/nutanix"
	"github.com/openshift/installer/pkg/types"
)

type NutanixCloudPlatformConfig struct {
	PlatformConfig
}

func (a *NutanixCloudPlatformConfig) CredsCheck(ic *installconfig.InstallConfig) error {
	return nil
}

func (a *NutanixCloudPlatformConfig) SetPlatform(p *types.Platform) error {
	var err error
	p.Nutanix, err = config.Platform()
	return err
}

func (a *NutanixCloudPlatformConfig) PermissionCheck(ic *installconfig.InstallConfig) error {
	return nil
}

func (a *NutanixCloudPlatformConfig) ProvisionCheck(ic *installconfig.InstallConfig) error {
	return config.ValidateForProvisioning(ic.Config)
}

func (a *NutanixCloudPlatformConfig) InstallConfigCheck(ic *installconfig.InstallConfig) error {
	return config.Validate(ic.Config)
}

func (a *NutanixCloudPlatformConfig) BaseDomainCheck(bd *installconfig.BaseDomain, p *installconfig.Platform) error {
	return nil
}

func init() {
	MapPlatforms["nutanix"] = &NutanixCloudPlatformConfig{}
}
