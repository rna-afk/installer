// Package azure generates Machine objects for azure.
package azure

import (
	"fmt"
	"sort"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/manifests/capiutils"
	"github.com/openshift/installer/pkg/types"
	"github.com/openshift/installer/pkg/types/azure"
)

// GenerateMachines returns manifests and runtime objects to provision the control plane (including bootstrap, if applicable) nodes using CAPI.
func GenerateMachines(platform *azure.Platform, pool *types.MachinePool, userDataSecret string, clusterID string, role string, capabilities map[string]string, useImageGallery bool, userTags map[string]string, hyperVGen string, subnet string, resourceGroup string) ([]*asset.RuntimeFile, error) {
	if poolPlatform := pool.Platform.Name(); poolPlatform != azure.Name {
		return nil, fmt.Errorf("non-Azure machine-pool: %q", poolPlatform)
	}
	mpool := pool.Platform.Azure

	total := int64(1)
	if pool.Replicas != nil {
		total = *pool.Replicas
	}

	if len(mpool.Zones) == 0 {
		// if no azs are given we set to []string{""} for convenience over later operations.
		// It means no-zoned for the machine API
		mpool.Zones = []string{""}
	}
	tags, err := CapzTagsFromUserTags(clusterID, userTags)
	if err != nil {
		return nil, fmt.Errorf("failed to create machineapi.TagSpecifications from UserTags: %w", err)
	}

	var image *capz.Image
	osImage := mpool.OSImage
	if osImage.Publisher != "" {
		image = &capz.Image{
			Marketplace: &capz.AzureMarketplaceImage{
				ImagePlan: capz.ImagePlan{
					Publisher: osImage.Publisher,
					Offer:     osImage.Offer,
					SKU:       osImage.SKU,
				},
				Version: osImage.Version,
			},
		}
	} else if useImageGallery {
		// image gallery names cannot have dashes
		galleryName := strings.Replace(clusterID, "-", "_", -1)
		id := clusterID
		if hyperVGen == "V2" {
			id += "-gen2"
		}
		imageID := fmt.Sprintf("/resourceGroups/%s/providers/Microsoft.Compute/galleries/gallery_%s/images/%s/versions/latest", resourceGroup, galleryName, id)
		image.ID = &imageID
	} else {
		imageID := fmt.Sprintf("/resourceGroups/%s/providers/Microsoft.Compute/images/%s", resourceGroup, clusterID)
		if hyperVGen == "V2" && platform.CloudName != azure.StackCloud {
			imageID += "-gen2"
		}
		image.ID = &imageID
	}

	osDisk := capz.OSDisk{
		OSType:     "Linux",
		DiskSizeGB: &mpool.DiskSizeGB,
		ManagedDisk: &capz.ManagedDiskParameters{
			StorageAccountType: mpool.DiskType,
		},
	}
	ultrassd := mpool.UltraSSDCapability == "Enabled"
	additionalCapabilities := &capz.AdditionalCapabilities{
		UltraSSDEnabled: &ultrassd,
	}

	machineProfile := generateSecurityProfile(mpool)
	securityProfile := &capz.SecurityProfile{
		EncryptionAtHost: machineProfile.EncryptionAtHost,
		SecurityType:     capz.SecurityTypes(machineProfile.Settings.SecurityType),
	}
	if machineProfile.Settings.ConfidentialVM != nil {
		securityProfile.UefiSettings = &capz.UefiSettings{
			VTpmEnabled:       ptr.To(machineProfile.Settings.ConfidentialVM.UEFISettings.VirtualizedTrustedPlatformModule == "Enabled"),
			SecureBootEnabled: ptr.To(machineProfile.Settings.ConfidentialVM.UEFISettings.SecureBoot == "Enabled"),
		}
	} else if machineProfile.Settings.TrustedLaunch != nil {
		securityProfile.UefiSettings = &capz.UefiSettings{
			VTpmEnabled:       ptr.To(machineProfile.Settings.TrustedLaunch.UEFISettings.VirtualizedTrustedPlatformModule == "Enabled"),
			SecureBootEnabled: ptr.To(machineProfile.Settings.TrustedLaunch.UEFISettings.SecureBoot == "Enabled"),
		}
	}

	var result []*asset.RuntimeFile
	for idx := int64(0); idx < total; idx++ {
		zone := mpool.Zones[int(idx)%len(mpool.Zones)]
		azureMachine := &capz.AzureMachine{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				Kind:       "AzureMachine",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s-%s-%d", clusterID, pool.Name, idx),
				Labels: map[string]string{
					"cluster.x-k8s.io/control-plane": "",
					"cluster.x-k8s.io/cluster-name":  clusterID,
				},
			},
			Spec: capz.AzureMachineSpec{
				VMSize:                 mpool.InstanceType,
				FailureDomain:          ptr.To(zone),
				Image:                  image,
				OSDisk:                 osDisk,
				AdditionalTags:         tags,
				AdditionalCapabilities: additionalCapabilities,
				AllocatePublicIP:       false,
				SecurityProfile:        securityProfile,
			},
		}
		result = append(result, &asset.RuntimeFile{
			File:   asset.File{Filename: fmt.Sprintf("10_inframachine_%s.yaml", azureMachine.Name)},
			Object: azureMachine,
		})

		controlPlaneMachine := &capi.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Name: azureMachine.Name,
				Labels: map[string]string{
					"cluster.x-k8s.io/control-plane": "",
				},
			},
			Spec: capi.MachineSpec{
				ClusterName: clusterID,
				Bootstrap: capi.Bootstrap{
					DataSecretName: ptr.To(fmt.Sprintf("%s-%s", clusterID, role)),
				},
				InfrastructureRef: v1.ObjectReference{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1beta2",
					Kind:       "AzureMachine",
					Name:       azureMachine.Name,
				},
			},
		}

		result = append(result, &asset.RuntimeFile{
			File:   asset.File{Filename: fmt.Sprintf("10_machine_%s.yaml", azureMachine.Name)},
			Object: controlPlaneMachine,
		})
	}

	bootstrapAzureMachine := &capz.AzureMachine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			Kind:       "AzureMachine",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: capiutils.GenerateBoostrapMachineName(clusterID),
			Labels: map[string]string{
				"cluster.x-k8s.io/control-plane": "",
				"install.openshift.io/bootstrap": "",
				"cluster.x-k8s.io/cluster-name":  clusterID,
			},
		},
		Spec: capz.AzureMachineSpec{
			VMSize:                 mpool.InstanceType,
			Image:                  image,
			FailureDomain:          ptr.To(mpool.Zones[0]),
			OSDisk:                 osDisk,
			AdditionalTags:         tags,
			AllocatePublicIP:       true,
			AdditionalCapabilities: additionalCapabilities,
			SecurityProfile:        securityProfile,
		},
	}

	result = append(result, &asset.RuntimeFile{
		File:   asset.File{Filename: fmt.Sprintf("10_inframachine_%s.yaml", bootstrapAzureMachine.Name)},
		Object: bootstrapAzureMachine,
	})

	bootstrapMachine := &capi.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name: bootstrapAzureMachine.Name,
			Labels: map[string]string{
				"cluster.x-k8s.io/control-plane": "",
			},
		},
		Spec: capi.MachineSpec{
			ClusterName: clusterID,
			Bootstrap: capi.Bootstrap{
				DataSecretName: ptr.To(fmt.Sprintf("%s-%s", clusterID, "bootstrap")),
			},
			InfrastructureRef: v1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta2",
				Kind:       "AzureMachine",
				Name:       bootstrapAzureMachine.Name,
			},
		},
	}

	result = append(result, &asset.RuntimeFile{
		File:   asset.File{Filename: fmt.Sprintf("10_machine_%s.yaml", bootstrapMachine.Name)},
		Object: bootstrapMachine,
	})

	return result, nil
}

// CapzTagsFromUserTags converts a map of user tags to a map of capz.Tags.
func CapzTagsFromUserTags(clusterID string, usertags map[string]string) (capz.Tags, error) {
	tags := capz.Tags{}
	tags[fmt.Sprintf("kubernetes.io_cluster.%s", clusterID)] = "owned"

	forbiddenTags := sets.NewString()
	for key := range tags {
		forbiddenTags.Insert(key)
	}

	userTagKeys := make([]string, 0, len(usertags))
	for key := range usertags {
		userTagKeys = append(userTagKeys, key)
	}
	sort.Strings(userTagKeys)

	for _, k := range userTagKeys {
		if forbiddenTags.Has(k) {
			return nil, fmt.Errorf("user tags may not clobber %s", k)
		}
		tags[k] = usertags[k]
	}
	return tags, nil
}
