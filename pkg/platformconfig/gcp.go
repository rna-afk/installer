package platformconfig

import (
	"context"

	"github.com/pkg/errors"

	"github.com/openshift/installer/pkg/asset/installconfig"
	config "github.com/openshift/installer/pkg/asset/installconfig/gcp"
	"github.com/openshift/installer/pkg/types"
)

type GCPCloudPlatformConfig struct {
	PlatformConfig
}

func (a *GCPCloudPlatformConfig) CredsCheck(ic *installconfig.InstallConfig) error {
	var err error
	_, err = config.GetSession(context.TODO())
	if err != nil {
		return errors.Wrap(err, "creating GCP session")
	}
	return nil
}

func (a *GCPCloudPlatformConfig) SetPlatform(p *types.Platform) error {
	var err error
	p.GCP, err = config.Platform()
	return err
}

func (a *GCPCloudPlatformConfig) PermissionCheck(ic *installconfig.InstallConfig) error {
	client, err := config.NewClient(context.TODO())
	if err != nil {
		return err
	}

	if err = config.ValidateEnabledServices(context.TODO(), client, ic.Config.GCP.ProjectID); err != nil {
		return errors.Wrap(err, "failed to validate services in this project")
	}
	return nil
}

func (a *GCPCloudPlatformConfig) ProvisionCheck(ic *installconfig.InstallConfig) error {
	client, err := config.NewClient(context.TODO())
	if err != nil {
		return err
	}
	err = config.ValidatePreExitingPublicDNS(client, ic.Config)
	if err != nil {
		return err
	}
	return nil
}

func (a *GCPCloudPlatformConfig) InstallConfigCheck(ic *installconfig.InstallConfig) error {
	client, err := config.NewClient(context.TODO())
	if err != nil {
		return err
	}
	return config.Validate(client, ic.Config)
}

func (a *GCPCloudPlatformConfig) BaseDomainCheck(bd *installconfig.BaseDomain, p *installconfig.Platform) error {
	var err error
	bd.BaseDomain, err = config.GetBaseDomain(p.GCP.ProjectID)

	// We are done if success (err == nil) or an err besides forbidden/throttling
	if !(config.IsForbidden(err) || config.IsThrottled(err)) {
		return err
	}
	return nil
}

func init() {
	MapPlatforms["gcp"] = &GCPCloudPlatformConfig{}
}
