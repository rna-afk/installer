package platformconfig

import (
	"github.com/pkg/errors"

	"github.com/openshift/installer/pkg/asset/installconfig"
	config "github.com/openshift/installer/pkg/asset/installconfig/ovirt"
	"github.com/openshift/installer/pkg/types"
)

type OvirtCloudPlatformConfig struct {
	PlatformConfig
}

func (a *OvirtCloudPlatformConfig) CredsCheck(ic *installconfig.InstallConfig) error {
	con, err := config.NewConnection()
	if err != nil {
		return errors.Wrap(err, "creating Engine connection")
	}
	err = con.Test()
	if err != nil {
		return errors.Wrap(err, "testing Engine connection")
	}
	return nil
}

func (a *OvirtCloudPlatformConfig) SetPlatform(p *types.Platform) error {
	var err error
	p.Ovirt, err = config.Platform()
	return err
}

func (a *OvirtCloudPlatformConfig) PermissionCheck(ic *installconfig.InstallConfig) error {
	return nil
}

func (a *OvirtCloudPlatformConfig) ProvisionCheck(ic *installconfig.InstallConfig) error {
	return config.ValidateForProvisioning(ic.Config)
}

func (a *OvirtCloudPlatformConfig) InstallConfigCheck(ic *installconfig.InstallConfig) error {
	return config.Validate(ic.Config)
}

func (a *OvirtCloudPlatformConfig) BaseDomainCheck(bd *installconfig.BaseDomain, p *installconfig.Platform) error {
	return nil
}

func init() {
	MapPlatforms["ovirt"] = &OvirtCloudPlatformConfig{}
}
