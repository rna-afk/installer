package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/openshift/installer/pkg/asset/installconfig"
	assetstore "github.com/openshift/installer/pkg/asset/store"
	"github.com/openshift/installer/pkg/asset/tls"
	"github.com/openshift/installer/pkg/clusterapi"
	serialgather "github.com/openshift/installer/pkg/gather"
	"github.com/openshift/installer/pkg/gather/ssh"
	"github.com/openshift/installer/pkg/infrastructure"
	infra "github.com/openshift/installer/pkg/infrastructure/platform"

	_ "github.com/openshift/installer/pkg/gather/aws"
	_ "github.com/openshift/installer/pkg/gather/azure"
	_ "github.com/openshift/installer/pkg/gather/gcp"
)

type gatherOptions struct {
	Bootstrap string
	Masters   []string
	SSHKeys   []string
}

func runGatherBootstrap(ctx context.Context, directory string, options gatherOptions) (string, error) {
	assetStore, err := assetstore.NewStore(directory)
	if err != nil {
		return "", fmt.Errorf("failed to create asset store: %w", err)
	}

	bootstrapSSHKeyPair := &tls.BootstrapSSHKeyPair{}
	if err := assetStore.Fetch(ctx, bootstrapSSHKeyPair); err != nil {
		return "", fmt.Errorf("failed to fetch %s: %w", bootstrapSSHKeyPair.Name(), err)
	}
	tmpfile, err := os.CreateTemp("", "bootstrap-ssh")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write(bootstrapSSHKeyPair.Private()); err != nil {
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		return "", err
	}

	sshKeys := append([]string{}, options.SSHKeys...)
	sshKeys = append(sshKeys, tmpfile.Name())

	ha := &infrastructure.HostAddresses{
		Bootstrap: options.Bootstrap,
		Port:      22,
		Masters:   options.Masters,
	}

	if ha.Bootstrap == "" && len(ha.Masters) == 0 {
		config := &installconfig.InstallConfig{}
		if err := assetStore.Fetch(ctx, config); err != nil {
			return "", fmt.Errorf("failed to fetch %s: %w", config.Name(), err)
		}

		provider, err := infra.ProviderForPlatform(config.Config.Platform.Name(), config.Config.EnabledFeatureGates())
		if err != nil {
			return "", fmt.Errorf("error getting infrastructure provider: %w", err)
		}
		if err = provider.ExtractHostAddresses(directory, config.Config, ha); err != nil {
			logrus.Warnf("Failed to extract host addresses: %s", err.Error())
		}
	}

	if ha.Bootstrap == "" {
		return "", errors.New("must provide bootstrap host address")
	}

	return gatherBootstrapLogs(ha.Bootstrap, ha.Port, ha.Masters, directory, sshKeys)
}

func gatherBootstrapLogs(bootstrap string, port int, masters []string, directory string, sshKeys []string) (string, error) {
	gatherID := time.Now().Format("20060102150405")
	archives := map[string]string{}

	if capiManifestsBundlePath, err := gatherCAPIArtifacts(directory, gatherID); err != nil {
		logrus.Infof("Failed to gather Cluster API manifests: %s", err.Error())
	} else {
		archives[capiManifestsBundlePath] = "clusterapi"
	}

	serialLogBundle := filepath.Join(directory, fmt.Sprintf("serial-log-bundle-%s.tar.gz", gatherID))
	serialLogBundlePath, err := filepath.Abs(serialLogBundle)
	if err != nil {
		return "", fmt.Errorf("failed to stat log file: %w", err)
	}

	consoleGather, err := serialgather.New(logrus.StandardLogger(), serialLogBundlePath, bootstrap, masters, directory)
	if err != nil {
		logrus.Infof("Skipping VM console logs gather: %s", err.Error())
	} else {
		logrus.Info("Pulling VM console logs")
		if err := consoleGather.Run(); err != nil {
			logrus.Infof("Failed to gather VM console logs: %s", err.Error())
		} else {
			archives[serialLogBundlePath] = "serial"
		}
	}

	clusterLogBundlePath, err := pullLogsFromBootstrap(gatherID, bootstrap, port, masters, directory, sshKeys)
	if err != nil {
		logrus.Infof("Failed to gather bootstrap logs: %s", err.Error())
	} else {
		archives[clusterLogBundlePath] = ""
	}

	if len(archives) == 0 {
		return "", fmt.Errorf("failed to gather VM console and bootstrap logs")
	}

	logBundlePath := filepath.Join(directory, fmt.Sprintf("log-bundle-%s.tar.gz", gatherID))
	if err := serialgather.CombineArchives(logBundlePath, archives); err != nil {
		return "", fmt.Errorf("failed to combine archives: %w", err)
	}

	return logBundlePath, nil
}

func pullLogsFromBootstrap(gatherID string, bootstrap string, port int, masters []string, directory string, sshKeys []string) (string, error) {
	logrus.Info("Pulling debug logs from the bootstrap machine")
	client, err := ssh.NewClient("core", net.JoinHostPort(bootstrap, strconv.Itoa(port)), sshKeys)
	if err != nil {
		if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ETIMEDOUT) {
			return "", fmt.Errorf("failed to connect to the bootstrap machine: %w", err)
		}
		return "", fmt.Errorf("failed to create SSH client: %w", err)
	}

	if err := ssh.Run(client, fmt.Sprintf("/usr/local/bin/installer-gather.sh --id %s %s", gatherID, strings.Join(masters, " "))); err != nil {
		return "", fmt.Errorf("failed to run remote command: %w", err)
	}

	file := filepath.Join(directory, fmt.Sprintf("cluster-log-bundle-%s.tar.gz", gatherID))
	if err := ssh.PullFileTo(client, fmt.Sprintf("/home/core/log-bundle-%s.tar.gz", gatherID), file); err != nil {
		return "", fmt.Errorf("failed to pull log file from remote: %w", err)
	}

	clusterLogBundlePath, err := filepath.Abs(file)
	if err != nil {
		return "", fmt.Errorf("failed to stat log file: %w", err)
	}

	return clusterLogBundlePath, nil
}

func gatherCAPIArtifacts(directory, gatherID string) (string, error) {
	logrus.Infoln("Pulling Cluster API artifacts")
	dir, err := filepath.Abs(directory)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for %s: %w", directory, err)
	}

	capiDir := filepath.Join(dir, clusterapi.ArtifactsDir)
	if _, err := os.Stat(capiDir); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("either Cluster API manifests not generated or terraform provision")
		}
		return "", fmt.Errorf("failed to stat Cluster API output directory: %w", err)
	}

	bundleDir := filepath.Join(dir, fmt.Sprintf("capi-artifacts-bundle-%s", gatherID))
	if err := os.Symlink(capiDir, bundleDir); err != nil {
		return "", fmt.Errorf("failed to copy Cluster API artifacts: %w", err)
	}
	defer os.Remove(bundleDir)

	manifests, err := filepath.Glob(filepath.Join(bundleDir, "*.yaml"))
	if err != nil {
		return "", fmt.Errorf("failed to gather Cluster API manifests: %w", err)
	}
	logs, err := filepath.Glob(filepath.Join(bundleDir, "*.log"))
	if err != nil {
		return "", fmt.Errorf("failed to gather Cluster API control plane logs: %w", err)
	}

	capiArtifactsBundlePath := fmt.Sprintf("%s.tar.gz", bundleDir)
	if err := serialgather.CreateArchive(append(manifests, logs...), capiArtifactsBundlePath); err != nil {
		return "", fmt.Errorf("failed to create clusterapi bundle file: %w", err)
	}
	return capiArtifactsBundlePath, nil
}
