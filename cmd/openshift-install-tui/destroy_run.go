package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sirupsen/logrus"

	assetstore "github.com/openshift/installer/pkg/asset/store"
	targetassets "github.com/openshift/installer/pkg/asset/targets"
	"github.com/openshift/installer/pkg/clusterapi"
	"github.com/openshift/installer/pkg/destroy"
	quotaasset "github.com/openshift/installer/pkg/destroy/quota"

	_ "github.com/openshift/installer/pkg/destroy/aws"
	_ "github.com/openshift/installer/pkg/destroy/azure"
	_ "github.com/openshift/installer/pkg/destroy/baremetal"
	_ "github.com/openshift/installer/pkg/destroy/gcp"
	_ "github.com/openshift/installer/pkg/destroy/ibmcloud"
	_ "github.com/openshift/installer/pkg/destroy/nutanix"
	_ "github.com/openshift/installer/pkg/destroy/openstack"
	_ "github.com/openshift/installer/pkg/destroy/ovirt"
	_ "github.com/openshift/installer/pkg/destroy/powervc"
	_ "github.com/openshift/installer/pkg/destroy/powervs"
	_ "github.com/openshift/installer/pkg/destroy/vsphere"
)

func (m *Model) startDestroy(returnTo screen) tea.Cmd {
	return m.runTaskUncancellable("Destroy cluster", returnTo, func(context.Context) (string, error) {
		reportQuota := os.Getenv("OPENSHIFT_INSTALL_REPORT_QUOTA_FOOTPRINT") == "true"
		if err := runDestroyCluster(m.dir, reportQuota); err != nil {
			return "", err
		}
		return "Cluster destroyed.", nil
	})
}

func runDestroyCluster(directory string, reportQuota bool) error {
	destroyer, err := destroy.New(logrus.StandardLogger(), directory)
	if err != nil {
		return fmt.Errorf("failed while preparing to destroy cluster: %w", err)
	}
	quota, err := destroyer.Run()
	if err != nil {
		return fmt.Errorf("failed to destroy cluster: %w", err)
	}

	if reportQuota {
		if err := quotaasset.WriteQuota(directory, quota); err != nil {
			return fmt.Errorf("failed to record quota: %w", err)
		}
	}

	store, err := assetstore.NewStore(directory)
	if err != nil {
		return fmt.Errorf("failed to create asset store: %w", err)
	}
	for _, asset := range targetassets.Cluster {
		if err := store.Destroy(asset); err != nil {
			return fmt.Errorf("failed to destroy asset %q: %w", asset.Name(), err)
		}
	}

	if err := store.DestroyState(); err != nil {
		return fmt.Errorf("failed to remove state file: %w", err)
	}

	tfstateFiles, err := filepath.Glob(filepath.Join(directory, "*.tfstate"))
	if err != nil {
		return fmt.Errorf("failed to glob for tfstate files: %w", err)
	}
	tfvarsFiles, err := filepath.Glob(filepath.Join(directory, "*.tfvars.json"))
	if err != nil {
		return fmt.Errorf("failed to glob for tfvars files: %w", err)
	}
	for _, file := range append(tfstateFiles, tfvarsFiles...) {
		if err := os.Remove(file); err != nil {
			return fmt.Errorf("failed to remove terraform file %q: %w", file, err)
		}
	}

	capiArtifactsDir := filepath.Join(directory, clusterapi.ArtifactsDir)
	if err := os.RemoveAll(capiArtifactsDir); err != nil {
		logrus.Warnf("failed to remove %s: %v", capiArtifactsDir, err)
	}

	return nil
}
