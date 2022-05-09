package platformconfig

import (
	"github.com/openshift/installer/pkg/asset/installconfig"
	config "github.com/openshift/installer/pkg/asset/installconfig/vsphere"
	"github.com/openshift/installer/pkg/types"
)

type VsphereCloudPlatformConfig struct {
	PlatformConfig
}

func (a *VsphereCloudPlatformConfig) CredsCheck(ic *installconfig.InstallConfig) error {
	return nil
}

func (a *VsphereCloudPlatformConfig) SetPlatform(p *types.Platform) error {
	var err error
	p.VSphere, err = config.Platform()
	return err
}

func (a *VsphereCloudPlatformConfig) PermissionCheck(ic *installconfig.InstallConfig) error {
	return nil
}

func (a *VsphereCloudPlatformConfig) ProvisionCheck(ic *installconfig.InstallConfig) error {
	return config.ValidateForProvisioning(ic.Config)
}

func (a *VsphereCloudPlatformConfig) InstallConfigCheck(ic *installconfig.InstallConfig) error {
	return config.Validate(ic.Config)
}

func (a *VsphereCloudPlatformConfig) BaseDomainCheck(bd *installconfig.BaseDomain, p *installconfig.Platform) error {
	return nil
}

func init() {
	MapPlatforms["vsphere"] = &VsphereCloudPlatformConfig{}
}
