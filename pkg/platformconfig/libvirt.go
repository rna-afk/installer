package platformconfig

import (
	"github.com/openshift/installer/pkg/asset/installconfig"
	config "github.com/openshift/installer/pkg/asset/installconfig/libvirt"
	"github.com/openshift/installer/pkg/types"
)

type LibvirtCloudPlatformConfig struct {
	PlatformConfig
}

func (a *LibvirtCloudPlatformConfig) CredsCheck(ic *installconfig.InstallConfig) error {
	return nil
}

func (a *LibvirtCloudPlatformConfig) SetPlatform(p *types.Platform) error {
	var err error
	p.Libvirt, err = config.Platform()
	return err
}

func (a *LibvirtCloudPlatformConfig) PermissionCheck(ic *installconfig.InstallConfig) error {
	return nil
}

func (a *LibvirtCloudPlatformConfig) ProvisionCheck(ic *installconfig.InstallConfig) error {
	return nil
}

func (a *LibvirtCloudPlatformConfig) InstallConfigCheck(ic *installconfig.InstallConfig) error {
	return nil
}

func (a *LibvirtCloudPlatformConfig) BaseDomainCheck(bd *installconfig.BaseDomain, p *installconfig.Platform) error {
	return nil
}

func init() {
	MapPlatforms["libvirt"] = &LibvirtCloudPlatformConfig{}
}
