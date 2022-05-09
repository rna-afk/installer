package platformconfig

import (
	"fmt"

	installconfig "github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/types"
)

type PlatformConfig interface {
	CredsCheck(ic *installconfig.InstallConfig) error
	SetPlatform(p *types.Platform) error
	PermissionCheck(ic *installconfig.InstallConfig) error
	ProvisionCheck(ic *installconfig.InstallConfig) error
	InstallConfigCheck(ic *installconfig.InstallConfig) error
	BaseDomainCheck(bd *installconfig.BaseDomain, p *installconfig.Platform) error
}

type AbstractPlatformConfig struct {
	PlatformConfig
}

func (a *AbstractPlatformConfig) CredsCheck(ic *installconfig.InstallConfig) error {
	return fmt.Errorf("unknown platform type %q", ic.Config.Platform.Name())
}

func (a *AbstractPlatformConfig) SetPlatform(p *types.Platform) error {
	return a.CredsCheck(nil)
}

func (a *AbstractPlatformConfig) PermissionCheck(ic *installconfig.InstallConfig) error {
	return a.CredsCheck(nil)
}

func (a *AbstractPlatformConfig) ProvisionCheck(ic *installconfig.InstallConfig) error {
	return a.CredsCheck(nil)
}

func (a *AbstractPlatformConfig) InstallConfigCheck(ic *installconfig.InstallConfig) error {
	return a.CredsCheck(nil)
}

func (a *AbstractPlatformConfig) BaseDomainCheck(bd *installconfig.BaseDomain, p *installconfig.Platform) error {
	return nil
}

var MapPlatforms map[string]PlatformConfig
var CurrentPlatform PlatformConfig

func init() {
	CurrentPlatform = &AbstractPlatformConfig{}
}

func SetCurrentPlatform(cloudName string) {
	if value, found := MapPlatforms[cloudName]; found {
		CurrentPlatform = value
	}
}
