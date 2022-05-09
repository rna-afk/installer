package platformconfig

import (
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/types"
	"github.com/openshift/installer/pkg/types/none"
)

type NoneCloudPlatformConfig struct {
	PlatformConfig
}

func (a *NoneCloudPlatformConfig) CredsCheck(ic *installconfig.InstallConfig) error {
	return nil
}

func (a *NoneCloudPlatformConfig) SetPlatform(p *types.Platform) error {
	var err error
	p.None = &none.Platform{}
	return err
}

func (a *NoneCloudPlatformConfig) PermissionCheck(ic *installconfig.InstallConfig) error {
	return nil
}

func (a *NoneCloudPlatformConfig) ProvisionCheck(ic *installconfig.InstallConfig) error {
	return nil
}

func (a *NoneCloudPlatformConfig) InstallConfigCheck(ic *installconfig.InstallConfig) error {
	return nil
}

func (a *NoneCloudPlatformConfig) BaseDomainCheck(bd *installconfig.BaseDomain, p *installconfig.Platform) error {
	return nil
}

func init() {
	MapPlatforms["none"] = &NoneCloudPlatformConfig{}
}
