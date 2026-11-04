package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	awsconfig "github.com/openshift/installer/pkg/asset/installconfig/aws"
	azureconfig "github.com/openshift/installer/pkg/asset/installconfig/azure"
	"github.com/openshift/installer/pkg/types"
	awstypes "github.com/openshift/installer/pkg/types/aws"
	azuretypes "github.com/openshift/installer/pkg/types/azure"
	gcptypes "github.com/openshift/installer/pkg/types/gcp"
	nonetypes "github.com/openshift/installer/pkg/types/none"
)

func TestFormBuildConfig(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, nonetypes.Name)
	f.clusterName.SetValue("test-cluster")
	f.baseDomain.SetValue("example.com")
	f.pullSecret.SetValue(`{"auths":{}}`)
	f.sshKeyPath.SetValue("")

	ic, err := f.buildConfig()
	if err != nil {
		t.Fatal(err)
	}
	if ic.ObjectMeta.Name != "test-cluster" {
		t.Fatalf("name = %q", ic.ObjectMeta.Name)
	}
	if ic.BaseDomain != "example.com" {
		t.Fatalf("baseDomain = %q", ic.BaseDomain)
	}
	if ic.Platform.None == nil {
		t.Fatal("expected platform none")
	}
	if ic.Platform.AWS != nil {
		t.Fatal("expected no AWS platform")
	}
	if ic.ControlPlane == nil || ic.ControlPlane.Replicas == nil || *ic.ControlPlane.Replicas != 3 {
		t.Fatalf("defaults were not applied: %+v", ic.ControlPlane)
	}
}

func TestFormBuildConfigRequiresAWSRegion(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, awstypes.Name)
	f.clusterName.SetValue("test-cluster")
	f.baseDomain.SetValue("example.com")
	f.pullSecret.SetValue(`{"auths":{}}`)
	f.sshKeyPath.SetValue("")
	f.region.SetValue("")

	if _, err := f.buildConfig(); err == nil {
		t.Fatal("expected an error for a missing AWS region")
	}
}

func TestFormBuildConfigRejectsInvalidPullSecret(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, nonetypes.Name)
	f.clusterName.SetValue("test-cluster")
	f.baseDomain.SetValue("example.com")
	f.pullSecret.SetValue("not-json")
	f.sshKeyPath.SetValue("")

	if _, err := f.buildConfig(); err == nil {
		t.Fatal("expected an error for an invalid pull secret")
	}
}

func TestFormBuildConfigPreservesAWSPlatform(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, awstypes.Name)
	f.clusterName.SetValue("test-cluster")
	f.baseDomain.SetValue("example.com")
	f.pullSecret.SetValue(`{"auths":{}}`)
	f.sshKeyPath.SetValue("")
	f.region.SetValue("us-west-2")
	f.existing = &types.InstallConfig{
		Platform: types.Platform{
			AWS: &awstypes.Platform{
				Region: "us-east-1",
				VPC: awstypes.VPC{
					Subnets: []awstypes.Subnet{{ID: "subnet-0123456789abcdef0"}},
				},
			},
		},
	}

	ic, err := f.buildConfig()
	if err != nil {
		t.Fatal(err)
	}
	if ic.Platform.AWS == nil {
		t.Fatal("expected AWS platform")
	}
	if ic.Platform.AWS.Region != "us-west-2" {
		t.Fatalf("region = %q", ic.Platform.AWS.Region)
	}
	if len(ic.Platform.AWS.VPC.Subnets) != 1 {
		t.Fatalf("expected VPC subnets to be preserved, got %+v", ic.Platform.AWS.VPC)
	}
}

func TestFormBuildConfigPreservesExistingSSHKey(t *testing.T) {
	const key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIexample"
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, nonetypes.Name)
	f.clusterName.SetValue("test-cluster")
	f.baseDomain.SetValue("example.com")
	f.pullSecret.SetValue(`{"auths":{}}`)
	f.existingSSHKey = key
	f.sshKeyPath.SetValue("")

	ic, err := f.buildConfig()
	if err != nil {
		t.Fatal(err)
	}
	if ic.SSHKey != key {
		t.Fatalf("sshKey = %q", ic.SSHKey)
	}
}

func TestFormLoadsExistingInstallConfig(t *testing.T) {
	dir := t.TempDir()
	data := []byte(`apiVersion: v1
baseDomain: example.com
metadata:
  name: existing
platform:
  none: {}
publish: Internal
pullSecret: '{"auths":{}}'
`)
	if err := os.WriteFile(filepath.Join(dir, installConfigName), data, 0o600); err != nil {
		t.Fatal(err)
	}

	f := newInstallConfigForm(dir)
	if f.existing == nil {
		t.Fatal("expected the existing install-config to load")
	}
	if f.clusterName.Value() != "existing" {
		t.Fatalf("cluster name = %q", f.clusterName.Value())
	}
	if f.baseDomain.Value() != "example.com" {
		t.Fatalf("base domain = %q", f.baseDomain.Value())
	}
	if f.currentPlatform() != nonetypes.Name {
		t.Fatalf("platform = %q", f.currentPlatform())
	}
	if f.currentPublish() != string(types.InternalPublishingStrategy) {
		t.Fatalf("publish = %q", f.currentPublish())
	}
}

func TestPlatformChoicesSorted(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	if !sort.StringsAreSorted(f.platforms) {
		t.Fatalf("platform choices are not sorted: %v", f.platforms)
	}
}

func TestDefaultPlatform(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	if f.currentPlatform() == "" {
		t.Fatal("expected a default platform selection")
	}
	if err := f.validateCurrentStep(); err != nil {
		t.Fatalf("default platform should be valid: %v", err)
	}
}

func TestFormStepsForAzure(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.azureFileCreds = nil
	f.platformIdx = indexOf(f.platforms, azuretypes.Name)

	want := []installFormStep{
		stepPlatform,
		stepAzureCredentials,
		stepRegion,
		stepBaseDomain,
		stepClusterName,
		stepPublish,
		stepSSHKey,
		stepPullSecret,
	}
	assertSteps(t, f.steps(), want)
}

func TestAzureCredentialsSkippedWhenFileCredentialsExist(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.azureFileCreds = &azureconfig.Credentials{
		SubscriptionID: "sub",
		TenantID:       "tenant",
		ClientID:       "client",
		ClientSecret:   "secret",
	}
	f.platformIdx = indexOf(f.platforms, azuretypes.Name)
	f.step = stepPlatform

	f.nextStep()

	if f.step != stepRegion {
		t.Fatalf("step = %v, want region", f.step)
	}
}

func TestAzureCredentialFailureReturnsToCredentialsStep(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.azureFileCreds = &azureconfig.Credentials{
		SubscriptionID: "sub",
		TenantID:       "tenant",
		ClientID:       "client",
		ClientSecret:   "secret",
	}
	f.platformIdx = indexOf(f.platforms, azuretypes.Name)
	f.step = stepRegion
	m := &Model{form: f, screen: screenInstallConfig}

	_, _ = m.handleAzureConnected(azureConnectedMsg{err: errors.New("bad credentials")})

	if f.step != stepAzureCredentials {
		t.Fatalf("step = %v, want azure credentials", f.step)
	}
	if !f.forceAzureCredentials {
		t.Fatal("expected credentials step to be forced")
	}
}

const testSSHKey = "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEAklOUpkDHrfHY17SbrmTIpNLTGK9Tjom/BWDSUGPl+nafzlHDTYW7hdI4yZ5ew18JH4JW9jbhUFrviQzM7xlELEVf4h9lFX5QVkbPppSwg0cda3Pbv7kOdJ/MTyBlWXFCR+HAo3FXRitBqxiX1nKhXpHAZsMciLq8V6RjsNAQwdsdMFvSlVK/7XAt3FaoJoAsncM1Q9x5+3V0Ww68/eIFmb1zuUFljQJKprrX88XypNDvjYNby6vw/Pb0rwert/EnmZ+AW4OZPnTPI89ZPmVMLuayrD2cE86Z/il8b+gw3r3+1nKatmIkjn2so1d01QraTlMqVSsbxNrRFi9wrf+M7Q=="

func TestLoadSSHKeys(t *testing.T) {
	home := t.TempDir()
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(sshDir, "id_test.pub")
	if err := os.WriteFile(keyPath, []byte(testSSHKey), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "bad.pub"), []byte("not-a-key"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	f := newInstallConfigForm(t.TempDir())
	f.loadSSHKeys()

	if len(f.sshKeyChoices) != 2 {
		t.Fatalf("choices = %+v, want one key and <none>", f.sshKeyChoices)
	}
	if f.sshKeyChoices[0].Path != keyPath {
		t.Fatalf("key path = %q, want %q", f.sshKeyChoices[0].Path, keyPath)
	}
	if f.sshKeyChoices[0].Content != testSSHKey {
		t.Fatal("key content was not loaded")
	}
	if f.sshKeyChoices[1].Label != noSSHKey {
		t.Fatalf("last choice = %q, want %q", f.sshKeyChoices[1].Label, noSSHKey)
	}
	if f.sshKeyIdx != 1 {
		t.Fatalf("default index = %d, want <none>", f.sshKeyIdx)
	}
}

func TestLoadSSHKeysPreservesExisting(t *testing.T) {
	home := t.TempDir()
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(sshDir, "id_test.pub")
	if err := os.WriteFile(keyPath, []byte(testSSHKey), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	f := newInstallConfigForm(t.TempDir())
	f.existingSSHKey = testSSHKey
	f.loadSSHKeys()

	if f.sshKeyChoices[0].Path != keyPath {
		t.Fatalf("selected path = %q, want %q", f.sshKeyChoices[0].Path, keyPath)
	}
	if f.sshKeyIdx != 0 {
		t.Fatalf("index = %d, want the matching key", f.sshKeyIdx)
	}

	f = newInstallConfigForm(t.TempDir())
	f.existingSSHKey = "ssh-rsa AAAAdifferent-key"
	f.loadSSHKeys()
	if !f.sshKeyChoices[0].Keep {
		t.Fatalf("first choice = %+v, want keep-existing", f.sshKeyChoices[0])
	}
	if f.sshKeyIdx != 0 {
		t.Fatalf("index = %d, want keep-existing", f.sshKeyIdx)
	}
}

func TestApplySSHKeyChoice(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.sshKeyChoices = []sshKeyChoice{
		{Label: "/tmp/id.pub", Path: "/tmp/id.pub", Content: testSSHKey},
		{Label: noSSHKey},
	}

	f.sshKeyIdx = 0
	f.applySSHKeyChoice()
	if f.sshKeyPath.Value() != "/tmp/id.pub" || f.sshKeyNone {
		t.Fatalf("path = %q, none = %v", f.sshKeyPath.Value(), f.sshKeyNone)
	}

	f.sshKeyIdx = 1
	f.applySSHKeyChoice()
	if f.sshKeyPath.Value() != "" || !f.sshKeyNone {
		t.Fatalf("path = %q, none = %v", f.sshKeyPath.Value(), f.sshKeyNone)
	}
}

func TestBuildConfigSSHKeyNone(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, nonetypes.Name)
	f.clusterName.SetValue("test-cluster")
	f.baseDomain.SetValue("example.com")
	f.pullSecret.SetValue(`{"auths":{}}`)
	f.existingSSHKey = testSSHKey
	f.sshKeyNone = true

	ic, err := f.buildConfig()
	if err != nil {
		t.Fatal(err)
	}
	if ic.SSHKey != "" {
		t.Fatalf("sshKey = %q, want empty", ic.SSHKey)
	}
}

func TestBuildConfigSSHKeyKeepExisting(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, nonetypes.Name)
	f.clusterName.SetValue("test-cluster")
	f.baseDomain.SetValue("example.com")
	f.pullSecret.SetValue(`{"auths":{}}`)
	f.existingSSHKey = testSSHKey
	f.sshKeyKeepExisting = true

	ic, err := f.buildConfig()
	if err != nil {
		t.Fatal(err)
	}
	if ic.SSHKey != testSSHKey {
		t.Fatalf("sshKey = %q", ic.SSHKey)
	}
}

func TestAzureRegionSearchBackspace(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, azuretypes.Name)
	f.step = stepRegion
	f.platformRegions = []string{"eastus", "westus"}
	f.focusCurrent()
	m := &Model{form: f, screen: screenInstallConfig}

	for _, r := range "east" {
		_, _ = m.handleInstallFormKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if got := f.regionSearch.Value(); got != "east" {
		t.Fatalf("search = %q, want east", got)
	}
	_, _ = m.handleInstallFormKey(tea.KeyMsg{Type: tea.KeyBackspace})
	if got := f.regionSearch.Value(); got != "eas" {
		t.Fatalf("search after backspace = %q, want eas", got)
	}
	_, _ = m.handleInstallFormKey(tea.KeyMsg{Type: tea.KeyCtrlH})
	if got := f.regionSearch.Value(); got != "ea" {
		t.Fatalf("search after ctrl+h = %q, want ea", got)
	}
}

func TestFilteredAzureRegions(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformRegions = []string{"eastus", "westus", "eastus2", "centralus"}
	f.regionSearch.SetValue("EAST")

	assertArgs(t, f.filteredRegions(), []string{"eastus", "eastus2"})
}

func TestApplyRegionFilterSelectsFirstMatch(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformRegions = []string{"eastus", "westus", "eastus2"}
	f.platformRegionIdx = 3
	f.regionSearch.SetValue("west")

	f.applyRegionFilter()
	if f.platformRegionIdx != 0 {
		t.Fatalf("region index = %d, want 0", f.platformRegionIdx)
	}
	if f.region.Value() != "westus" {
		t.Fatalf("region = %q, want westus", f.region.Value())
	}
}

func TestAzureRegionStepShowsSearch(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, azuretypes.Name)
	f.step = stepRegion
	f.platformRegions = []string{"eastus", "westus"}
	f.regionSearch.SetValue("east")
	f.applyRegionFilter()

	m := newModel(options{})
	m.form = f
	m.screen = screenInstallConfig
	m.width, m.height = 120, 40

	view := m.View()
	if !strings.Contains(view, "Search:") {
		t.Fatal("region search input is missing")
	}
	if !strings.Contains(view, "eastus") {
		t.Fatal("matching region is missing")
	}
	if strings.Contains(view, "westus") {
		t.Fatal("non-matching region should be filtered out")
	}
}

func TestAzureConnectedPopulatesSelections(t *testing.T) {
	const zoneID = "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Network/dnszones/example.com"
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, azuretypes.Name)
	f.step = stepAzureCredentials
	m := &Model{form: f, screen: screenInstallConfig}

	_, _ = m.handleAzureConnected(azureConnectedMsg{
		regions: []string{"eastus", "westus"},
		zones:   []azureconfig.Zone{{Name: "example.com", ID: zoneID}},
	})

	if f.step != stepRegion {
		t.Fatalf("step = %v, want region", f.step)
	}
	if !f.regionSearch.Focused() {
		t.Fatal("region search input is not focused")
	}
	if f.region.Value() != "eastus" {
		t.Fatalf("region = %q", f.region.Value())
	}
	if f.baseDomain.Value() != "example.com" {
		t.Fatalf("base domain = %q", f.baseDomain.Value())
	}
	if f.baseDomainZoneID != zoneID {
		t.Fatalf("base domain zone ID = %q", f.baseDomainZoneID)
	}
}

func TestBuildConfigAzure(t *testing.T) {
	const zoneID = "/subscriptions/sub/resourceGroups/myrg/providers/Microsoft.Network/dnszones/example.com"
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, azuretypes.Name)
	f.region.SetValue("eastus")
	f.clusterName.SetValue("test-cluster")
	f.baseDomain.SetValue("example.com")
	f.baseDomainZoneID = zoneID
	f.pullSecret.SetValue(`{"auths":{}}`)
	f.sshKeyPath.SetValue("")

	ic, err := f.buildConfig()
	if err != nil {
		t.Fatal(err)
	}
	if ic.Platform.Azure == nil {
		t.Fatal("expected Azure platform")
	}
	if ic.Platform.Azure.Region != "eastus" {
		t.Fatalf("region = %q", ic.Platform.Azure.Region)
	}
	if ic.Platform.Azure.CloudName != azuretypes.PublicCloud {
		t.Fatalf("cloud = %q", ic.Platform.Azure.CloudName)
	}
	if ic.Platform.Azure.BaseDomainResourceGroupName != "myrg" {
		t.Fatalf("base domain resource group = %q", ic.Platform.Azure.BaseDomainResourceGroupName)
	}
}

func TestFormStepsSkipRegionForNonAWS(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, nonetypes.Name)

	want := []installFormStep{
		stepPlatform,
		stepBaseDomain,
		stepClusterName,
		stepPublish,
		stepSSHKey,
		stepPullSecret,
	}
	assertSteps(t, f.steps(), want)

	f.step = stepBaseDomain
	f.prevStep()
	if f.step != stepPlatform {
		t.Fatalf("prev step = %v, want platform", f.step)
	}
}

func TestFormStepsIncludeRegionForAWS(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, awstypes.Name)

	want := []installFormStep{
		stepPlatform,
		stepRegion,
		stepBaseDomain,
		stepClusterName,
		stepPublish,
		stepSSHKey,
		stepPullSecret,
	}
	assertSteps(t, f.steps(), want)

	f.step = stepBaseDomain
	f.prevStep()
	if f.step != stepRegion {
		t.Fatalf("prev step = %v, want region", f.step)
	}
}

func TestEditingStepMarksItDirty(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.step = stepClusterName
	f.clusterName.Focus()

	f.updateFocused(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("abc")})

	if !f.dirty[stepClusterName] {
		t.Fatal("expected cluster name step to be marked dirty")
	}
}

func TestNextStepKeepsDownstreamValuesWhenStepUnchanged(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, awstypes.Name)
	f.step = stepPlatform
	setDownstreamValues(f)

	f.nextStep()

	if f.step != stepRegion {
		t.Fatalf("step = %v, want region", f.step)
	}
	assertDownstreamValues(t, f)
}

func TestNextStepResetsDownstreamWhenPlatformChanges(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, awstypes.Name)
	f.step = stepPlatform
	setDownstreamValues(f)
	f.markDirty(stepPlatform)

	f.nextStep()

	if f.step != stepRegion {
		t.Fatalf("step = %v, want region", f.step)
	}
	if f.region.Value() != "" || f.baseDomain.Value() != "" || f.clusterName.Value() != "" || f.pullSecret.Value() != "" {
		t.Fatalf("expected downstream values to be cleared: %+v", f)
	}
	if f.publishIdx != 0 || f.sshKeyPath.Value() != "" || len(f.sshKeyChoices) != 0 {
		t.Fatal("expected publish, SSH key, and pull secret to be reset")
	}
}

func TestNextStepPreservesAzureCredentialFields(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, awstypes.Name)
	f.step = stepPlatform
	f.azureSubscriptionID.SetValue("sub")
	f.azureTenantID.SetValue("tenant")
	f.azureClientID.SetValue("client")
	f.azureClientSecret.SetValue("secret")
	f.markDirty(stepPlatform)

	f.nextStep()

	if f.azureSubscriptionID.Value() != "sub" || f.azureTenantID.Value() != "tenant" ||
		f.azureClientID.Value() != "client" || f.azureClientSecret.Value() != "secret" {
		t.Fatal("expected Azure credential fields to be preserved")
	}
}

func TestNextStepResetsOnlyLaterSteps(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, awstypes.Name)
	f.region.SetValue("us-east-1")
	f.baseDomain.SetValue("example.com")
	f.step = stepClusterName
	f.clusterName.SetValue("cluster")
	f.publishIdx = 1
	f.sshKeyPath.SetValue("/tmp/key.pub")
	f.pullSecret.SetValue("secret")
	f.markDirty(stepClusterName)

	f.nextStep()

	if f.step != stepPublish {
		t.Fatalf("step = %v, want publish", f.step)
	}
	if f.region.Value() != "us-east-1" || f.baseDomain.Value() != "example.com" || f.clusterName.Value() != "cluster" {
		t.Fatal("stored or current step values should be preserved")
	}
	if f.publishIdx != 0 || f.sshKeyPath.Value() != "" || f.pullSecret.Value() != "" {
		t.Fatal("expected later step values to be reset")
	}
	if len(f.dirty) != 0 {
		t.Fatal("expected dirty state to be cleared")
	}
}

func setDownstreamValues(f *installConfigForm) {
	f.region.SetValue("us-east-1")
	f.baseDomain.SetValue("example.com")
	f.baseDomainOptions = []string{"example.com"}
	f.clusterName.SetValue("cluster")
	f.publishIdx = 1
	f.sshKeyPath.SetValue("/tmp/key.pub")
	f.sshKeyChoices = []sshKeyChoice{{Label: "key", Path: "/tmp/key.pub"}}
	f.pullSecret.SetValue("secret")
}

func assertDownstreamValues(t *testing.T, f *installConfigForm) {
	t.Helper()
	if f.region.Value() != "us-east-1" || f.baseDomain.Value() != "example.com" || f.clusterName.Value() != "cluster" {
		t.Fatal("expected values to be preserved")
	}
	if f.publishIdx != 1 || f.sshKeyPath.Value() != "/tmp/key.pub" || f.pullSecret.Value() != "secret" {
		t.Fatal("expected later values to be preserved")
	}
}

func TestHandleBaseDomainsSelectsExistingValue(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, awstypes.Name)
	f.region.SetValue("us-east-1")
	f.baseDomain.SetValue("example.com")
	m := &Model{form: f, screen: screenInstallConfig}

	_, _ = m.handleBaseDomains(baseDomainsMsg{
		key:     "us-east-1",
		domains: []string{"example.com", "other.com"},
	})

	if f.baseDomainErr != "" {
		t.Fatalf("unexpected error: %s", f.baseDomainErr)
	}
	if f.baseDomainIdx != 0 {
		t.Fatalf("base domain index = %d, want 0", f.baseDomainIdx)
	}
	if f.baseDomain.Value() != "example.com" {
		t.Fatalf("base domain = %q", f.baseDomain.Value())
	}
}

func TestHandleBaseDomainsKeepsExistingValueWhenMissing(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, awstypes.Name)
	f.region.SetValue("us-east-1")
	f.baseDomain.SetValue("example.com")
	m := &Model{form: f, screen: screenInstallConfig}

	_, _ = m.handleBaseDomains(baseDomainsMsg{
		key:     "us-east-1",
		domains: []string{"other.com"},
	})

	if len(f.baseDomainOptions) != 2 || f.baseDomainOptions[0] != "example.com" {
		t.Fatalf("options = %v", f.baseDomainOptions)
	}
	if f.baseDomainIdx != 0 {
		t.Fatalf("base domain index = %d, want 0", f.baseDomainIdx)
	}
}

func TestHandleBaseDomainsAPIErrorShowsCredentials(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, awstypes.Name)
	f.region.SetValue("us-east-1")
	m := &Model{form: f, screen: screenInstallConfig}

	_, _ = m.handleBaseDomains(baseDomainsMsg{
		key: "us-east-1",
		err: errors.New("access denied"),
	})

	if !f.needsAWSCreds {
		t.Fatal("expected the credentials step to be enabled")
	}
	if f.baseDomainErr != "access denied" {
		t.Fatalf("base domain error = %q", f.baseDomainErr)
	}
}

func TestHandleBaseDomainsEmptyList(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, awstypes.Name)
	f.region.SetValue("us-east-1")
	m := &Model{form: f, screen: screenInstallConfig}

	_, _ = m.handleBaseDomains(baseDomainsMsg{key: "us-east-1"})
	if f.baseDomainErr == "" {
		t.Fatal("expected an error when no domains are returned")
	}
}

func assertSteps(t *testing.T, got, want []installFormStep) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("steps = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("steps = %v, want %v", got, want)
		}
	}
}

func TestHandleBaseDomainsNoCredentialsShowsCredsStep(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, awstypes.Name)
	f.region.SetValue("us-east-1")
	f.step = stepBaseDomain
	m := &Model{form: f, screen: screenInstallConfig}

	_, _ = m.handleBaseDomains(baseDomainsMsg{
		key: "us-east-1",
		err: fmt.Errorf("%w: missing", awsconfig.ErrNoCredentials),
	})

	if !f.needsAWSCreds {
		t.Fatal("expected the credentials step to be enabled")
	}
	if f.baseDomainIsSelect() || f.baseDomainIsManual() {
		t.Fatal("base domain input should be hidden while credentials are required")
	}
	if f.baseDomainErr != "AWS credentials are required to list Route 53 hosted zones" {
		t.Fatalf("base domain error = %q", f.baseDomainErr)
	}
}

func TestLoadBaseDomainsRequiresCredentialFields(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, awstypes.Name)
	f.region.SetValue("us-east-1")
	f.step = stepBaseDomain
	f.needsAWSCreds = true
	m := &Model{form: f, screen: screenInstallConfig}

	if cmd := m.loadBaseDomains(); cmd != nil {
		t.Fatal("expected no command without credentials")
	}
	if f.baseDomainErr == "" {
		t.Fatal("expected a validation error")
	}
	if f.credsFocus != 0 {
		t.Fatalf("creds focus = %d, want access key", f.credsFocus)
	}
}

func TestBaseDomainStepShowsCredentialsForm(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, awstypes.Name)
	f.region.SetValue("us-east-1")
	f.step = stepBaseDomain
	f.needsAWSCreds = true

	m := newModel(options{})
	m.form = f
	m.screen = screenInstallConfig
	m.width, m.height = 120, 40

	view := m.View()
	if !strings.Contains(view, "Unable to list Route 53 hosted zones") {
		t.Fatal("credentials prompt is missing")
	}
	if !strings.Contains(view, "Access key ID") || !strings.Contains(view, "Secret access key") {
		t.Fatalf("credential fields are missing:\n%s", view)
	}
}

func TestBaseDomainStepShowsInlineSpinnerWhileLoadingCredentials(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, awstypes.Name)
	f.region.SetValue("us-east-1")
	f.step = stepBaseDomain
	f.needsAWSCreds = true
	f.baseDomainLoading = true

	m := newModel(options{})
	m.form = f
	m.screen = screenInstallConfig
	m.width, m.height = 120, 40

	view := m.View()
	if !strings.Contains(view, "Looking up Route 53 hosted zones") {
		t.Fatalf("loading spinner is missing while fetching base domains:\n%s", view)
	}
	if !strings.Contains(view, "Access key ID") {
		t.Fatalf("credential fields should stay visible while loading:\n%s", view)
	}
}

func TestBaseDomainStepShowsSelection(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, awstypes.Name)
	f.region.SetValue("us-east-1")
	f.step = stepBaseDomain
	f.baseDomainOptions = []string{"example.com", "other.com"}
	f.baseDomainIdx = 1
	f.baseDomain.SetValue("other.com")

	m := newModel(options{})
	m.form = f
	m.screen = screenInstallConfig
	m.width, m.height = 120, 40

	view := m.View()
	if !strings.Contains(view, "Base domain") {
		t.Fatal("base domain step is not shown")
	}
	if !strings.Contains(view, "example.com") || !strings.Contains(view, "other.com") {
		t.Fatalf("base domain options are missing from the view:\n%s", view)
	}
}

func TestSetFormPlatformClearsOtherPlatforms(t *testing.T) {
	ic := &types.InstallConfig{
		Platform: types.Platform{
			AWS: &awstypes.Platform{Region: "us-east-1"},
		},
	}

	if err := setFormPlatform(ic, gcptypes.Name, ""); err != nil {
		t.Fatal(err)
	}
	if ic.Platform.AWS != nil {
		t.Fatal("expected AWS platform to be cleared")
	}
	if ic.Platform.GCP == nil {
		t.Fatal("expected GCP platform to be set")
	}
}

func TestGCPFormSteps(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, gcptypes.Name)

	steps := f.steps()
	for _, want := range []installFormStep{stepPlatform, stepGCPCredentials, stepGCPProject, stepRegion, stepBaseDomain} {
		if indexOfStep(steps, want) < 0 {
			t.Fatalf("steps %v are missing %v", steps, want)
		}
	}
	if indexOfStep(steps, stepGCPCredentials) > indexOfStep(steps, stepGCPProject) {
		t.Fatal("credentials should be collected before the project")
	}
	if indexOfStep(steps, stepGCPProject) > indexOfStep(steps, stepRegion) {
		t.Fatal("the project is needed before regions can be listed")
	}
}

func TestGCPFormBuildConfig(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.json")
	if err := os.WriteFile(keyPath, []byte(`{"type":"service_account","project_id":"p"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")

	f := newInstallConfigForm(dir)
	f.platformIdx = indexOf(f.platforms, gcptypes.Name)
	f.gcpProject.SetValue("my-project")
	f.region.SetValue("us-central1")
	f.gcpCredsPath.SetValue(keyPath)
	f.clusterName.SetValue("test-cluster")
	f.baseDomain.SetValue("example.com")
	f.pullSecret.SetValue(`{"auths":{}}`)
	f.sshKeyPath.SetValue("")

	ic, err := f.buildConfig()
	if err != nil {
		t.Fatal(err)
	}
	if ic.Platform.GCP == nil {
		t.Fatal("expected a GCP platform")
	}
	if ic.Platform.GCP.ProjectID != "my-project" {
		t.Fatalf("project = %q, want my-project", ic.Platform.GCP.ProjectID)
	}
	if ic.Platform.GCP.Region != "us-central1" {
		t.Fatalf("region = %q, want us-central1", ic.Platform.GCP.Region)
	}
	if got := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); got != keyPath {
		t.Fatalf("GOOGLE_APPLICATION_CREDENTIALS = %q, want %q", got, keyPath)
	}
}

func TestGCPFormRejectsNonKeyFile(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "not-a-key.json")
	if err := os.WriteFile(keyPath, []byte(`{"not":"a key"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	f := newInstallConfigForm(dir)
	f.platformIdx = indexOf(f.platforms, gcptypes.Name)
	f.gcpProject.SetValue("my-project")
	f.region.SetValue("us-central1")
	f.gcpCredsPath.SetValue(keyPath)
	f.clusterName.SetValue("test-cluster")
	f.baseDomain.SetValue("example.com")
	f.pullSecret.SetValue(`{"auths":{}}`)
	f.sshKeyPath.SetValue("")

	if _, err := f.buildConfig(); err == nil {
		t.Fatal("expected an error for a non service-account key file")
	}
}

func TestGCPFormRequiresProject(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, gcptypes.Name)
	f.step = stepGCPProject

	if err := f.validateCurrentStep(); err == nil {
		t.Fatal("expected an error for a missing GCP project")
	}

	f.gcpProject.SetValue("my-project")
	if err := f.validateCurrentStep(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleRegionsSelectsExisting(t *testing.T) {
	for _, platform := range []string{awstypes.Name, gcptypes.Name} {
		f := newInstallConfigForm(t.TempDir())
		f.platformIdx = indexOf(f.platforms, platform)
		f.region.SetValue("us-east-2")
		m := &Model{form: f, screen: screenInstallConfig}

		_, _ = m.handleRegions(regionsMsg{
			platform: platform,
			key:      "all",
			regions:  []string{"us-east-1", "us-east-2", "us-west-2"},
		})

		if f.regionsErr != "" {
			t.Fatalf("%s: unexpected error: %s", platform, f.regionsErr)
		}
		if len(f.platformRegions) != 3 {
			t.Fatalf("%s: regions = %v", platform, f.platformRegions)
		}
		if !f.regionIsSelect() {
			t.Fatalf("%s: expected a selectable region list", platform)
		}
		if f.platformRegionIdx != 1 || f.region.Value() != "us-east-2" {
			t.Fatalf("%s: selected %q at %d, want us-east-2 at 1", platform, f.region.Value(), f.platformRegionIdx)
		}
	}
}

func TestHandleRegionsErrorFallsBackToManualEntry(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, gcptypes.Name)
	m := &Model{form: f, screen: screenInstallConfig}

	_, _ = m.handleRegions(regionsMsg{platform: gcptypes.Name, key: "p", err: errors.New("permission denied")})

	if f.regionsErr == "" {
		t.Fatal("expected the lookup error to be recorded")
	}
	if f.regionIsSelect() {
		t.Fatal("expected a manual region field when the lookup fails")
	}
}

func TestRegionFilterUsesPlatformRegions(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, gcptypes.Name)
	f.platformRegions = []string{"us-central1", "europe-west1", "us-east1"}

	f.regionSearch.SetValue("US-")
	if got := f.filteredRegions(); len(got) != 2 || got[0] != "us-central1" || got[1] != "us-east1" {
		t.Fatalf("filtered regions = %v", got)
	}
}

func TestGCPBaseDomainErrorKeepsManualEntry(t *testing.T) {
	f := newInstallConfigForm(t.TempDir())
	f.platformIdx = indexOf(f.platforms, gcptypes.Name)
	f.gcpProject.SetValue("my-project")
	m := &Model{form: f, screen: screenInstallConfig}

	_, _ = m.handleBaseDomains(baseDomainsMsg{key: "my-project", err: errors.New("permission denied")})

	if f.needsAWSCreds {
		t.Fatal("GCP errors should not ask for AWS credentials")
	}
	if f.baseDomainErr == "" {
		t.Fatal("expected the lookup error to be recorded")
	}
	if !f.baseDomainIsManual() {
		t.Fatal("expected manual base-domain entry to stay available")
	}
}

func TestGCPCredentialError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	for _, env := range []string{"GOOGLE_APPLICATION_CREDENTIALS", "GOOGLE_CREDENTIALS", "GOOGLE_CLOUD_KEYFILE_JSON", "GCLOUD_KEYFILE_JSON"} {
		t.Setenv(env, "")
	}

	if err := gcpCredentialError(); err == nil {
		t.Fatal("expected an error when no GCP credentials are available")
	}

	envKey := filepath.Join(home, "key.json")
	if err := os.WriteFile(envKey, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", envKey)
	if err := gcpCredentialError(); err != nil {
		t.Fatalf("unexpected error with GOOGLE_APPLICATION_CREDENTIALS: %v", err)
	}

	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")
	adcPath := filepath.Join(home, "config", "gcloud", "application_default_credentials.json")
	if err := os.MkdirAll(filepath.Dir(adcPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adcPath, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := gcpCredentialError(); err != nil {
		t.Fatalf("unexpected error with gcloud application default credentials: %v", err)
	}
}
