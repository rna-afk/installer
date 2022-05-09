package platformconfig

import (
	"github.com/pkg/errors"

	"github.com/openshift/installer/pkg/asset/installconfig"
	config "github.com/openshift/installer/pkg/asset/installconfig/openstack"
	"github.com/openshift/installer/pkg/types"
)

type OpenstackCloudPlatformConfig struct {
	PlatformConfig
}

func (a *OpenstackCloudPlatformConfig) CredsCheck(ic *installconfig.InstallConfig) error {
	var err error
	_, err = config.GetSession(ic.Config.Platform.OpenStack.Cloud)
	if err != nil {
		return errors.Wrap(err, "creating OpenStack session")
	}
	return nil
}

func (a *OpenstackCloudPlatformConfig) SetPlatform(p *types.Platform) error {
	var err error
	p.OpenStack, err = config.Platform()
	return err
}

func (a *OpenstackCloudPlatformConfig) PermissionCheck(ic *installconfig.InstallConfig) error {
	return nil
}

func (a *OpenstackCloudPlatformConfig) ProvisionCheck(ic *installconfig.InstallConfig) error {
	return config.ValidateForProvisioning(ic.Config)
}

func (a *OpenstackCloudPlatformConfig) InstallConfigCheck(ic *installconfig.InstallConfig) error {
	return config.Validate(ic.Config)
}

func (a *OpenstackCloudPlatformConfig) BaseDomainCheck(bd *installconfig.BaseDomain, p *installconfig.Platform) error {
	return nil
}

func init() {
	MapPlatforms["openstack"] = &OpenstackCloudPlatformConfig{}
}
