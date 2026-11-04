package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	clientwatch "k8s.io/client-go/tools/watch"

	operatorv1 "github.com/openshift/api/operator/v1"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	installercommand "github.com/openshift/installer/cmd/openshift-install/command"
	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/asset/lbconfig"
	assetstore "github.com/openshift/installer/pkg/asset/store"
	"github.com/openshift/installer/pkg/types"
	"github.com/openshift/installer/pkg/types/aws"
	"github.com/openshift/installer/pkg/types/azure"
	"github.com/openshift/installer/pkg/types/baremetal"
	"github.com/openshift/installer/pkg/types/dns"
	"github.com/openshift/installer/pkg/types/gcp"
	"github.com/openshift/installer/pkg/types/vsphere"
	baremetalutils "github.com/openshift/installer/pkg/utils/baremetal"
)

func waitOptionsForInstallConfig(ic *types.InstallConfig) installercommand.WaitOptions {
	options := installercommand.WaitOptions{
		VerifyFIPS: false,
	}
	if ic != nil {
		switch ic.Platform.Name() {
		case baremetal.Name:
			options.ExtendTimeoutForBaremetal = true
		case aws.Name:
			if ic.AWS != nil && ic.AWS.UserProvisionedDNS == dns.UserProvisionedDNSEnabled {
				options.UserProvisionedDNSEnabled = true
			}
		}
	}
	return options
}

func runWaitForBootstrapComplete(ctx context.Context, config *rest.Config, directory string) error {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "creating a Kubernetes client")
	}

	discovery := client.Discovery()
	apiTimeout := 20 * time.Minute
	untilTime := time.Now().Add(apiTimeout)
	timezone, _ := untilTime.Zone()
	logrus.Infof("Waiting up to %v (until %v %s) for the Kubernetes API at %s...",
		apiTimeout, untilTime.Format(time.Kitchen), timezone, config.Host)

	apiContext, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

	if assetStore, err := assetstore.NewStore(directory); err == nil {
		installercommand.CheckIfAgentCommand(assetStore)
	}

	var lastErr error
	err = wait.PollUntilContextCancel(apiContext, 2*time.Second, true, func(_ context.Context) (bool, error) {
		version, err := discovery.ServerVersion()
		if err == nil {
			logrus.Infof("API %s up", version)
			return true, nil
		}
		lastErr = err
		logrus.Debugf("Still waiting for the Kubernetes API: %v", err)
		return false, nil
	})
	if err != nil {
		if lastErr != nil {
			return errors.Wrap(lastErr, "failed waiting for Kubernetes API")
		}
		return errors.Wrap(err, "failed waiting for Kubernetes API")
	}

	var platformName string
	var installConfig *installconfig.InstallConfig
	if assetStore, err := assetstore.NewStore(directory); err == nil {
		if ic, err := assetStore.Load(&installconfig.InstallConfig{}); err == nil && ic != nil {
			installConfig = ic.(*installconfig.InstallConfig)
			platformName = installConfig.Config.Platform.Name()
		}
	}

	timeout := 45 * time.Minute
	if platformName == baremetal.Name || platformName == vsphere.Name {
		timeout = 60 * time.Minute
	}
	if platformName == aws.Name && installConfig != nil && installConfig.Config.AWS != nil && installConfig.Config.AWS.UserProvisionedDNS == dns.UserProvisionedDNSEnabled {
		timeout = 60 * time.Minute
		logrus.Infof("Increasing bootstrapping timeout on AWS to %v since UserProvisionedDNS is enabled", timeout)
	}

	untilTime = time.Now().Add(timeout)
	timezone, _ = untilTime.Zone()
	logrus.Infof("Waiting up to %v (until %v %s) for bootstrapping to complete...",
		timeout, untilTime.Format(time.Kitchen), timezone)

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if platformName == baremetal.Name {
		if err := baremetalutils.WaitForBaremetalBootstrapControlPlane(waitCtx, config, directory); err != nil {
			return errors.Wrap(err, "failed waiting for baremetal bootstrap control plane")
		}
		logrus.Infof("  Baremetal control plane finished provisioning.")
	}

	if err := waitForBootstrapConfigMap(waitCtx, client); err != nil {
		return err
	}

	if err := waitForEtcdBootstrapMemberRemoval(ctx, config); err != nil {
		return errors.Wrap(err, "failed waiting for the bootstrap etcd member removal")
	}

	return nil
}

func waitForBootstrapConfigMap(ctx context.Context, client *kubernetes.Clientset) error {
	_, err := clientwatch.UntilWithSync(
		ctx,
		cache.NewListWatchFromClient(client.CoreV1().RESTClient(), "configmaps", "kube-system", fields.OneTermEqualSelector("metadata.name", "bootstrap")),
		&corev1.ConfigMap{},
		nil,
		func(event watch.Event) (bool, error) {
			switch event.Type {
			case watch.Added, watch.Modified:
			default:
				return false, nil
			}
			cm, ok := event.Object.(*corev1.ConfigMap)
			if !ok {
				return false, nil
			}
			status, ok := cm.Data["status"]
			return ok && status == "complete", nil
		},
	)
	if err != nil {
		return errors.Wrap(err, "failed waiting for the bootstrap configmap")
	}
	return nil
}

func waitForEtcdBootstrapMemberRemoval(ctx context.Context, config *rest.Config) error {
	logrus.Info("Waiting for the bootstrap etcd member to be removed...")
	client, err := operatorclient.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating operator client: %w", err)
	}
	if err := wait.PollUntilContextCancel(ctx, 1*time.Second, true, func(ctx context.Context) (bool, error) {
		etcd, err := client.OperatorV1().Etcds().Get(ctx, "cluster", metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		for _, condition := range etcd.Status.Conditions {
			if condition.Type == "EtcdBootstrapMemberRemoved" {
				return condition.Status == operatorv1.ConditionTrue, nil
			}
		}
		return false, nil
	}); err != nil {
		return err
	}
	logrus.Info("Bootstrap etcd member has been removed")
	return nil
}

func handleUnreachableAPIServer(ctx context.Context, config *rest.Config, directory string) error {
	assetStore, err := assetstore.NewStore(directory)
	if err != nil {
		return fmt.Errorf("failed to create asset store: %w", err)
	}

	installConfig := &installconfig.InstallConfig{}
	if err := assetStore.Fetch(ctx, installConfig); err != nil {
		return fmt.Errorf("failed to fetch %s: %w", installConfig.Name(), err)
	}
	switch installConfig.Config.Platform.Name() { //nolint:gocritic
	case aws.Name:
		if installConfig.Config.AWS.UserProvisionedDNS != dns.UserProvisionedDNSEnabled {
			return nil
		}
	case azure.Name:
		if installConfig.Config.Azure.UserProvisionedDNS != dns.UserProvisionedDNSEnabled {
			return nil
		}
	case gcp.Name:
		if installConfig.Config.GCP.UserProvisionedDNS != dns.UserProvisionedDNSEnabled {
			return nil
		}
	default:
		return nil
	}

	lbConfig := &lbconfig.Config{}
	if err := assetStore.Fetch(ctx, lbConfig); err != nil {
		return fmt.Errorf("failed to fetch %s: %w", lbConfig.Name(), err)
	}

	lbType := lbconfig.PublicLoadBalancer
	if !installConfig.Config.PublicAPI() {
		lbType = lbconfig.PrivateLoadBalancer
	}

	_, ipAddrs, err := lbConfig.ParseDNSDataFromConfig(lbType)
	if err != nil {
		return fmt.Errorf("failed to parse lbconfig: %w", err)
	}

	ipAddr := ""
	if len(ipAddrs) > 0 {
		ipAddr = ipAddrs[0].String()
	}
	if ipAddr == "" {
		return fmt.Errorf("no ip address found in lbconfig")
	}

	dialer := &net.Dialer{Timeout: 1 * time.Minute, KeepAlive: 1 * time.Minute}
	config.Dial = kubeconfig.CreateDialContext(dialer, ipAddr)

	if err := asset.DeleteAssetFromDisk(lbConfig, directory); err != nil {
		return fmt.Errorf("failed to delete %s from disk", lbConfig.Name())
	}
	return nil
}
