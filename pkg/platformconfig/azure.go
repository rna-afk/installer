package platformconfig

import (
	"github.com/openshift/installer/pkg/asset/installconfig"
	config "github.com/openshift/installer/pkg/asset/installconfig/azure"
	"github.com/openshift/installer/pkg/types"
	"github.com/openshift/installer/pkg/types/azure"
)

type AzureCloudPlatformConfig struct {
	PlatformConfig
}

func (a *AzureCloudPlatformConfig) CredsCheck(ic *installconfig.InstallConfig) error {
	_, err := ic.Azure.Session()
	return err
}

func (a *AzureCloudPlatformConfig) SetPlatform(p *types.Platform) error {
	var err error
	p.Azure, err = config.Platform()
	return err
}

func (a *AzureCloudPlatformConfig) PermissionCheck(ic *installconfig.InstallConfig) error {
	return nil
}

func (a *AzureCloudPlatformConfig) ProvisionCheck(ic *installconfig.InstallConfig) error {
	dnsConfig, err := ic.Azure.DNSConfig()
	if err != nil {
		return err
	}
	err = config.ValidatePublicDNS(ic.Config, dnsConfig)
	if err != nil {
		return err
	}
	client, err := ic.Azure.Client()
	if err != nil {
		return err
	}
	return config.ValidateForProvisioning(client, ic.Config)
}

func (a *AzureCloudPlatformConfig) InstallConfigCheck(ic *installconfig.InstallConfig) error {
	client, err := ic.Azure.Client()
	if err != nil {
		return err
	}
	return config.Validate(client, ic.Config)
}

func (a *AzureCloudPlatformConfig) BaseDomainCheck(bd *installconfig.BaseDomain, p *installconfig.Platform) error {
	// Create client using public cloud because install config has not been generated yet.
	ssn, err := config.GetSession(azure.PublicCloud, "")
	if err != nil {
		return err
	}
	azureDNS := config.NewDNSConfig(ssn)
	zone, err := azureDNS.GetDNSZone()
	if err != nil {
		return err
	}
	bd.BaseDomain = zone.Name
	return p.Azure.SetBaseDomain(zone.ID)
}

func init() {
	MapPlatforms["azure"] = &AzureCloudPlatformConfig{}
}
