package platformconfig

import (
	"github.com/pkg/errors"

	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/installconfig/alibabacloud"
	config "github.com/openshift/installer/pkg/asset/installconfig/alibabacloud"
	"github.com/openshift/installer/pkg/types"
)

type AlibabaCloudPlatformConfig struct {
	PlatformConfig
}

func (a *AlibabaCloudPlatformConfig) CredsCheck(ic *installconfig.InstallConfig) error {
	_, err := ic.AlibabaCloud.Client()
	if err != nil {
		return errors.Wrap(err, "creating AlibabaCloud Cloud session")
	}
	return nil
}

func (a *AlibabaCloudPlatformConfig) SetPlatform(p *types.Platform) error {
	var err error
	p.AlibabaCloud, err = config.Platform()
	return err
}

func (a *AlibabaCloudPlatformConfig) PermissionCheck(ic *installconfig.InstallConfig) error {
	return nil
}

func (a *AlibabaCloudPlatformConfig) ProvisionCheck(ic *installconfig.InstallConfig) error {
	client, err := ic.AlibabaCloud.Client()
	if err != nil {
		return err
	}
	return config.ValidateForProvisioning(client, ic.Config, ic.AlibabaCloud)
}

func (a *AlibabaCloudPlatformConfig) InstallConfigCheck(ic *installconfig.InstallConfig) error {
	ic.AlibabaCloud = config.NewMetadata(ic.Config.AlibabaCloud.Region, ic.Config.AlibabaCloud.VSwitchIDs)
	client, err := ic.AlibabaCloud.Client()
	if err != nil {
		return err
	}
	return alibabacloud.Validate(client, ic.Config)
}

func (a *AlibabaCloudPlatformConfig) BaseDomainCheck(bd *installconfig.BaseDomain, p *installconfig.Platform) error {
	var err error
	bd.BaseDomain, err = config.GetBaseDomain()
	return err
}

func init() {
	MapPlatforms["alibabacloud"] = &AlibabaCloudPlatformConfig{}
}
