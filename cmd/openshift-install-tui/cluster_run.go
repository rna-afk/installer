package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"

	installercommand "github.com/openshift/installer/cmd/openshift-install/command"
	"github.com/openshift/installer/pkg/asset/installconfig"
	assetstore "github.com/openshift/installer/pkg/asset/store"
	destroybootstrap "github.com/openshift/installer/pkg/destroy/bootstrap"
	"github.com/openshift/installer/pkg/types"
)

// runClusterPost runs the post-provisioning phases of `create cluster`.
func runClusterPost(ctx context.Context, dir string, skipPasswordPrint bool) error {
	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(dir, "auth", "kubeconfig"))
	if err != nil {
		return fmt.Errorf("loading kubeconfig: %w", err)
	}

	installercommand.RootOpts.Dir = dir
	installercommand.SkipPasswordPrintFlag = skipPasswordPrint

	if err := handleUnreachableAPIServer(ctx, config, dir); err != nil {
		return fmt.Errorf("unable to handle API server override: %w", err)
	}

	if err := runWaitForBootstrapComplete(ctx, config, dir); err != nil {
		if logErr := installercommand.LogClusterOperatorConditions(ctx, config); logErr != nil {
			logrus.Error("Attempted to gather ClusterOperator status after installation failure: ", logErr)
		}
		return fmt.Errorf("bootstrap failed to complete: %w", err)
	}

	if os.Getenv("OPENSHIFT_INSTALL_PRESERVE_BOOTSTRAP") == "" {
		logrus.Info("Destroying the bootstrap resources...")
		if err := destroybootstrap.Destroy(ctx, dir); err != nil {
			return err
		}
	} else {
		logrus.Warn("OPENSHIFT_INSTALL_PRESERVE_BOOTSTRAP is set, not destroying bootstrap resources. " +
			"Warning: this should only be used for debugging purposes, and poses a risk to cluster stability.")
	}

	store, err := assetstore.NewStore(dir)
	if err != nil {
		return err
	}

	var installConfig *types.InstallConfig
	if asset, err := store.Load(&installconfig.InstallConfig{}); err == nil && asset != nil {
		installConfig = asset.(*installconfig.InstallConfig).Config
	}

	options := waitOptionsForInstallConfig(installConfig)
	if err := installercommand.WaitForInstallComplete(ctx, config, options); err != nil {
		if logErr := installercommand.LogClusterOperatorConditions(ctx, config); logErr != nil {
			logrus.Error("Attempted to gather ClusterOperator status after installation failure: ", logErr)
		}
		installercommand.LogTroubleshootingLink()
		return err
	}

	return nil
}
