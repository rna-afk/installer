package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	awsconfig "github.com/openshift/installer/pkg/asset/installconfig/aws"
	azureconfig "github.com/openshift/installer/pkg/asset/installconfig/azure"
	targetassets "github.com/openshift/installer/pkg/asset/targets"
	"github.com/openshift/installer/pkg/types"
	awstypes "github.com/openshift/installer/pkg/types/aws"
	azuretypes "github.com/openshift/installer/pkg/types/azure"
	baremetaltypes "github.com/openshift/installer/pkg/types/baremetal"
	"github.com/openshift/installer/pkg/types/defaults"
	gcptypes "github.com/openshift/installer/pkg/types/gcp"
	ibmcloudtypes "github.com/openshift/installer/pkg/types/ibmcloud"
	nonetypes "github.com/openshift/installer/pkg/types/none"
	nutanixtypes "github.com/openshift/installer/pkg/types/nutanix"
	openstacktypes "github.com/openshift/installer/pkg/types/openstack"
	powervctypes "github.com/openshift/installer/pkg/types/powervc"
	powervstypes "github.com/openshift/installer/pkg/types/powervs"
	vspheretypes "github.com/openshift/installer/pkg/types/vsphere"
	"github.com/openshift/installer/pkg/validate"
)

const noSSHKey = "<none>"

type installFormStep int

const (
	stepPlatform installFormStep = iota
	stepAzureCredentials
	stepRegion
	stepBaseDomain
	stepClusterName
	stepPublish
	stepSSHKey
	stepPullSecret
)

// installConfigForm walks the user through install-config inputs one at a
// time. Some inputs depend on earlier ones (for example, AWS base domains are
// looked up using the selected region).
type installConfigForm struct {
	step installFormStep

	platforms   []string
	platformIdx int

	region textinput.Model

	baseDomain          textinput.Model
	baseDomainOptions   []string
	baseDomainIdx       int
	baseDomainLoading   bool
	baseDomainLoadedFor string
	baseDomainErr       string
	baseDomainRequest   int
	needsAWSCreds       bool
	credsFocus          int

	awsAccessKey  textinput.Model
	awsSecretKey  textinput.Model
	awsSessionKey textinput.Model

	azureSubscriptionID   textinput.Model
	azureTenantID         textinput.Model
	azureClientID         textinput.Model
	azureClientSecret     textinput.Model
	azureCredsFocus       int
	azureConnecting       bool
	forceAzureCredentials bool
	azureSession          *azureconfig.Session
	azureFileCreds        *azureconfig.Credentials
	azureRegions          []string
	azureRegionIdx        int
	regionSearch          textinput.Model
	azureZones            []azureconfig.Zone
	baseDomainIDs         map[string]string
	baseDomainZoneID      string

	clusterName textinput.Model
	publishes   []string
	publishIdx  int
	sshKeyPath  textinput.Model

	sshKeyChoices      []sshKeyChoice
	sshKeyIdx          int
	sshKeyNone         bool
	sshKeyKeepExisting bool

	pullSecret textinput.Model

	err   string
	dirty map[installFormStep]bool

	existing       *types.InstallConfig
	existingSSHKey string
}

var allInstallFormSteps = []installFormStep{
	stepPlatform,
	stepAzureCredentials,
	stepRegion,
	stepBaseDomain,
	stepClusterName,
	stepPublish,
	stepSSHKey,
	stepPullSecret,
}

type sshKeyChoice struct {
	Label   string
	Path    string
	Content string
	Keep    bool
}

type baseDomainsMsg struct {
	request int
	region  string
	domains []string
	err     error
}

func newInstallConfigForm(dir string) *installConfigForm {
	f := &installConfigForm{
		clusterName:         newFormInput("my-cluster"),
		region:              newFormInput("eastus"),
		baseDomain:          newFormInput("example.com"),
		sshKeyPath:          newFormInput("~/.ssh/id_ed25519.pub"),
		pullSecret:          newFormInput("pull secret JSON (hidden)"),
		awsAccessKey:        newFormInput("AKIA..."),
		awsSecretKey:        newFormInput("secret access key"),
		awsSessionKey:       newFormInput("session token (optional)"),
		azureSubscriptionID: newFormInput("subscription ID"),
		azureTenantID:       newFormInput("tenant ID"),
		azureClientID:       newFormInput("client ID"),
		azureClientSecret:   newFormInput("client secret"),
		regionSearch:        newFormInput("type to filter regions"),
		platforms:           formPlatforms(),
		platformIdx:         0,
		publishes:           []string{string(types.ExternalPublishingStrategy), string(types.InternalPublishingStrategy)},
		baseDomainIDs:       map[string]string{},
		dirty:               map[installFormStep]bool{},
	}
	f.pullSecret.EchoMode = textinput.EchoPassword
	f.pullSecret.EchoCharacter = '•'
	f.awsSecretKey.EchoMode = textinput.EchoPassword
	f.awsSecretKey.EchoCharacter = '•'
	f.awsSessionKey.EchoMode = textinput.EchoPassword
	f.awsSessionKey.EchoCharacter = '•'
	f.azureClientSecret.EchoMode = textinput.EchoPassword
	f.azureClientSecret.EchoCharacter = '•'

	if creds, err := azureconfig.LoadCredentialsFromFile(); err == nil {
		f.azureFileCreds = creds
		f.azureSubscriptionID.SetValue(creds.SubscriptionID)
		f.azureTenantID.SetValue(creds.TenantID)
		f.azureClientID.SetValue(creds.ClientID)
		f.azureClientSecret.SetValue(creds.ClientSecret)
	}

	if ic, err := loadInstallConfig(dir); err != nil {
		f.err = err.Error()
	} else if ic != nil {
		f.existing = ic
		f.existingSSHKey = ic.SSHKey
		f.clusterName.SetValue(ic.ObjectMeta.Name)
		f.baseDomain.SetValue(ic.BaseDomain)
		f.pullSecret.SetValue(ic.PullSecret)
		if ic.Platform.AWS != nil {
			f.region.SetValue(ic.Platform.AWS.Region)
		}
		if ic.Platform.Azure != nil {
			f.region.SetValue(ic.Platform.Azure.Region)
		}
		if idx := indexOf(f.platforms, ic.Platform.Name()); idx >= 0 {
			f.platformIdx = idx
		}
		if ic.Publish != "" {
			if idx := indexOf(f.publishes, string(ic.Publish)); idx >= 0 {
				f.publishIdx = idx
			}
		}
	}

	f.step = stepPlatform
	f.focusCurrent()
	return f
}

func newFormInput(placeholder string) textinput.Model {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = placeholder
	input.CharLimit = 0
	input.Width = 46
	return input
}

func formPlatforms() []string {
	platforms := append([]string{}, types.PlatformNames...)
	platforms = append(platforms, nonetypes.Name)
	sort.Strings(platforms)
	return platforms
}

func loadInstallConfig(dir string) (*types.InstallConfig, error) {
	path := filepath.Join(dir, installConfigName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	ic := &types.InstallConfig{}
	if err := yaml.Unmarshal(data, ic); err != nil {
		return nil, fmt.Errorf("cannot parse %s: %w", installConfigName, err)
	}
	return ic, nil
}

func (f *installConfigForm) currentPlatform() string {
	if f.platformIdx < 0 || f.platformIdx >= len(f.platforms) {
		return ""
	}
	return f.platforms[f.platformIdx]
}

func (f *installConfigForm) currentPublish() string {
	if len(f.publishes) == 0 {
		return string(types.ExternalPublishingStrategy)
	}
	return f.publishes[f.publishIdx]
}

func (f *installConfigForm) steps() []installFormStep {
	if f.currentPlatform() == "" {
		return []installFormStep{stepPlatform}
	}

	steps := []installFormStep{stepPlatform}
	switch f.currentPlatform() {
	case awstypes.Name:
		steps = append(steps, stepRegion)
	case azuretypes.Name:
		if !f.skipAzureCredentials() {
			steps = append(steps, stepAzureCredentials)
		}
		steps = append(steps, stepRegion)
	}
	return append(steps, stepBaseDomain, stepClusterName, stepPublish, stepSSHKey, stepPullSecret)
}

func (f *installConfigForm) skipAzureCredentials() bool {
	return f.azureFileCreds != nil && !f.forceAzureCredentials
}

func (f *installConfigForm) stepPosition() (int, int) {
	steps := f.steps()
	for i, step := range steps {
		if step == f.step {
			return i + 1, len(steps)
		}
	}
	return 1, len(steps)
}

func (f *installConfigForm) stepTitle() string {
	switch f.step {
	case stepPlatform:
		return "Platform"
	case stepAzureCredentials:
		return "Azure credentials"
	case stepRegion:
		if f.currentPlatform() == azuretypes.Name {
			return "Azure region"
		}
		return "AWS region"
	case stepBaseDomain:
		if f.needsAWSCreds {
			return "AWS credentials"
		}
		return "Base domain"
	case stepClusterName:
		return "Cluster name"
	case stepPublish:
		return "Publish mode"
	case stepSSHKey:
		return "SSH public key"
	case stepPullSecret:
		return "Pull secret"
	default:
		return ""
	}
}

func (f *installConfigForm) nextStep() {
	if f.dirty[f.step] {
		f.resetDownstream(f.step)
		f.clearDirty()
	}

	steps := f.steps()
	index := indexOfStep(steps, f.step)
	if index < 0 {
		f.step = steps[0]
		return
	}
	if index < len(steps)-1 {
		f.step = steps[index+1]
	}
}

func (f *installConfigForm) markDirty(step installFormStep) {
	f.dirty[step] = true
}

func (f *installConfigForm) clearDirty() {
	for step := range f.dirty {
		delete(f.dirty, step)
	}
}

func (f *installConfigForm) resetDownstream(from installFormStep) {
	index := indexOfStep(allInstallFormSteps, from)
	if index < 0 {
		return
	}
	for _, step := range allInstallFormSteps[index+1:] {
		f.resetStep(step)
	}
}

func (f *installConfigForm) resetStep(step installFormStep) {
	switch step {
	case stepAzureCredentials:
		// Keep the credential fields: they come from the ambient Azure file or
		// the user's input and remain valid when downstream steps are reset.
		f.azureCredsFocus = 0
		f.azureConnecting = false
		f.azureSession = nil
		f.azureRegions = nil
		f.azureRegionIdx = 0
		f.azureZones = nil
		f.baseDomainOptions = nil
		f.baseDomainIDs = map[string]string{}
		f.baseDomainZoneID = ""
	case stepRegion:
		f.region.SetValue("")
		f.regionSearch.SetValue("")
		f.azureRegions = nil
		f.azureRegionIdx = 0
		f.azureZones = nil
	case stepBaseDomain:
		f.baseDomainRequest++
		f.baseDomain.SetValue("")
		f.baseDomainIdx = 0
		f.baseDomainLoading = false
		f.baseDomainErr = ""
		f.baseDomainLoadedFor = ""
		f.baseDomainZoneID = ""
		if f.currentPlatform() == azuretypes.Name {
			if len(f.baseDomainOptions) > 0 {
				f.baseDomainIdx = 0
				f.baseDomain.SetValue(f.baseDomainOptions[0])
				f.baseDomainZoneID = f.baseDomainIDs[f.baseDomainOptions[0]]
			}
		} else {
			f.baseDomainOptions = nil
			f.baseDomainIDs = map[string]string{}
		}
	case stepClusterName:
		f.clusterName.SetValue("")
	case stepPublish:
		f.publishIdx = 0
	case stepSSHKey:
		f.sshKeyPath.SetValue("")
		f.sshKeyChoices = nil
		f.sshKeyIdx = 0
		f.sshKeyNone = false
		f.sshKeyKeepExisting = false
		f.existingSSHKey = ""
	case stepPullSecret:
		f.pullSecret.SetValue("")
	}
}

func (f *installConfigForm) currentStepValue() string {
	switch f.step {
	case stepAzureCredentials:
		return strings.Join([]string{
			f.azureSubscriptionID.Value(),
			f.azureTenantID.Value(),
			f.azureClientID.Value(),
			f.azureClientSecret.Value(),
		}, "\x00")
	case stepRegion:
		return f.region.Value()
	case stepBaseDomain:
		if f.needsAWSCreds {
			return strings.Join([]string{
				f.awsAccessKey.Value(),
				f.awsSecretKey.Value(),
				f.awsSessionKey.Value(),
			}, "\x00")
		}
		return f.baseDomain.Value()
	case stepClusterName:
		return f.clusterName.Value()
	case stepPullSecret:
		return f.pullSecret.Value()
	}
	return ""
}

func (f *installConfigForm) prevStep() {
	steps := f.steps()
	index := indexOfStep(steps, f.step)
	if index > 0 {
		f.step = steps[index-1]
	}
}

func (f *installConfigForm) resetBaseDomainLookup() {
	f.baseDomainRequest++
	f.baseDomainLoading = false
	f.baseDomainOptions = nil
	f.baseDomainIdx = 0
	f.baseDomainErr = ""
	f.baseDomainLoadedFor = ""
	f.needsAWSCreds = false
	f.credsFocus = 0
	f.awsAccessKey.SetValue("")
	f.awsSecretKey.SetValue("")
	f.awsSessionKey.SetValue("")
	f.azureSession = nil
	f.azureRegions = nil
	f.azureRegionIdx = 0
	f.azureZones = nil
	f.azureConnecting = false
	f.forceAzureCredentials = false
	f.azureCredsFocus = 0
	f.baseDomainIDs = map[string]string{}
	f.baseDomainZoneID = ""
}

func (f *installConfigForm) loadSSHKeys() {
	f.sshKeyChoices = nil
	f.sshKeyIdx = 0

	home, err := os.UserHomeDir()
	if err == nil {
		paths, _ := filepath.Glob(filepath.Join(home, ".ssh", "*.pub"))
		sort.Strings(paths)
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			content := strings.TrimSpace(string(data))
			if err := validate.SSHPublicKey(content); err != nil {
				continue
			}
			f.sshKeyChoices = append(f.sshKeyChoices, sshKeyChoice{
				Label:   path,
				Path:    path,
				Content: content,
			})
		}
	}

	existing := strings.TrimSpace(f.existingSSHKey)
	if existing != "" {
		matched := -1
		for i, choice := range f.sshKeyChoices {
			if choice.Content == existing {
				matched = i
				break
			}
		}
		if matched >= 0 {
			f.sshKeyIdx = matched
		} else {
			f.sshKeyChoices = append([]sshKeyChoice{{Label: "<keep existing key>", Keep: true}}, f.sshKeyChoices...)
			f.sshKeyIdx = 0
		}
		f.sshKeyChoices = append(f.sshKeyChoices, sshKeyChoice{Label: noSSHKey})
	} else {
		f.sshKeyChoices = append(f.sshKeyChoices, sshKeyChoice{Label: noSSHKey})
		f.sshKeyIdx = len(f.sshKeyChoices) - 1
	}

	// Preserve a choice made earlier in the same TUI session.
	switch {
	case f.sshKeyNone:
		for i, choice := range f.sshKeyChoices {
			if choice.Label == noSSHKey {
				f.sshKeyIdx = i
			}
		}
	case f.sshKeyKeepExisting:
		for i, choice := range f.sshKeyChoices {
			if choice.Keep {
				f.sshKeyIdx = i
			}
		}
	default:
		path := f.sshKeyPath.Value()
		if path != "" {
			for i, choice := range f.sshKeyChoices {
				if choice.Path == path {
					f.sshKeyIdx = i
				}
			}
		}
	}
}

func (f *installConfigForm) sshKeyChoiceLabels() []string {
	labels := make([]string, len(f.sshKeyChoices))
	for i, choice := range f.sshKeyChoices {
		labels[i] = choice.Label
	}
	return labels
}

func (f *installConfigForm) applySSHKeyChoice() {
	if len(f.sshKeyChoices) == 0 {
		return
	}
	if f.sshKeyIdx < 0 || f.sshKeyIdx >= len(f.sshKeyChoices) {
		f.sshKeyIdx = 0
	}

	choice := f.sshKeyChoices[f.sshKeyIdx]
	switch {
	case choice.Keep:
		f.sshKeyNone = false
		f.sshKeyKeepExisting = true
		f.sshKeyPath.SetValue("")
	case choice.Label == noSSHKey:
		f.sshKeyNone = true
		f.sshKeyKeepExisting = false
		f.sshKeyPath.SetValue("")
	default:
		f.sshKeyNone = false
		f.sshKeyKeepExisting = false
		f.sshKeyPath.SetValue(choice.Path)
	}
}

func (f *installConfigForm) filteredAzureRegions() []string {
	query := strings.ToLower(strings.TrimSpace(f.regionSearch.Value()))
	if query == "" {
		return f.azureRegions
	}

	out := make([]string, 0, len(f.azureRegions))
	for _, region := range f.azureRegions {
		if strings.Contains(strings.ToLower(region), query) {
			out = append(out, region)
		}
	}
	return out
}

func (f *installConfigForm) applyRegionFilter() {
	filtered := f.filteredAzureRegions()
	f.azureRegionIdx = 0
	if len(filtered) == 0 {
		f.region.SetValue("")
		return
	}
	f.region.SetValue(filtered[0])
}

func (f *installConfigForm) baseDomainIsSelect() bool {
	if f.needsAWSCreds {
		return false
	}
	switch f.currentPlatform() {
	case awstypes.Name, azuretypes.Name:
		return !f.baseDomainLoading && f.baseDomainErr == "" && len(f.baseDomainOptions) > 0
	default:
		return false
	}
}

func (f *installConfigForm) baseDomainIsManual() bool {
	if f.needsAWSCreds {
		return false
	}
	switch f.currentPlatform() {
	case awstypes.Name, azuretypes.Name:
		return !f.baseDomainLoading && !f.baseDomainIsSelect()
	default:
		return true
	}
}

func (f *installConfigForm) blurAll() {
	f.region.Blur()
	f.regionSearch.Blur()
	f.baseDomain.Blur()
	f.awsAccessKey.Blur()
	f.awsSecretKey.Blur()
	f.awsSessionKey.Blur()
	f.azureSubscriptionID.Blur()
	f.azureTenantID.Blur()
	f.azureClientID.Blur()
	f.azureClientSecret.Blur()
	f.clusterName.Blur()
	f.sshKeyPath.Blur()
	f.pullSecret.Blur()
}

func (f *installConfigForm) focusCurrent() {
	f.blurAll()
	switch f.step {
	case stepAzureCredentials:
		switch f.azureCredsFocus {
		case 0:
			f.azureSubscriptionID.Focus()
		case 1:
			f.azureTenantID.Focus()
		case 2:
			f.azureClientID.Focus()
		case 3:
			f.azureClientSecret.Focus()
		}
		return
	case stepRegion:
		if f.currentPlatform() == azuretypes.Name && len(f.azureRegions) > 0 {
			f.regionSearch.Focus()
			return
		}
		f.region.Focus()
	case stepBaseDomain:
		if f.needsAWSCreds {
			switch f.credsFocus {
			case 0:
				f.awsAccessKey.Focus()
			case 1:
				f.awsSecretKey.Focus()
			case 2:
				f.awsSessionKey.Focus()
			}
			return
		}
		if f.baseDomainIsManual() {
			f.baseDomain.Focus()
		}
	case stepClusterName:
		f.clusterName.Focus()
	case stepPullSecret:
		f.pullSecret.Focus()
	}
}

func (f *installConfigForm) updateFocused(msg tea.Msg) tea.Cmd {
	before := f.currentStepValue()
	var cmd tea.Cmd
	switch f.step {
	case stepAzureCredentials:
		switch f.azureCredsFocus {
		case 0:
			f.azureSubscriptionID, cmd = f.azureSubscriptionID.Update(msg)
		case 1:
			f.azureTenantID, cmd = f.azureTenantID.Update(msg)
		case 2:
			f.azureClientID, cmd = f.azureClientID.Update(msg)
		case 3:
			f.azureClientSecret, cmd = f.azureClientSecret.Update(msg)
		}
		f.markDirtyIfChanged(before)
		return cmd
	case stepRegion:
		if f.currentPlatform() == azuretypes.Name && len(f.azureRegions) > 0 {
			f.regionSearch, cmd = f.regionSearch.Update(msg)
			if _, ok := msg.(tea.KeyMsg); ok {
				f.applyRegionFilter()
			}
			f.markDirtyIfChanged(before)
			return cmd
		}
		f.region, cmd = f.region.Update(msg)
	case stepBaseDomain:
		if f.needsAWSCreds {
			switch f.credsFocus {
			case 0:
				f.awsAccessKey, cmd = f.awsAccessKey.Update(msg)
			case 1:
				f.awsSecretKey, cmd = f.awsSecretKey.Update(msg)
			case 2:
				f.awsSessionKey, cmd = f.awsSessionKey.Update(msg)
			}
			f.markDirtyIfChanged(before)
			return cmd
		}
		if f.baseDomainIsManual() {
			f.baseDomain, cmd = f.baseDomain.Update(msg)
		}
	case stepClusterName:
		f.clusterName, cmd = f.clusterName.Update(msg)
	case stepPullSecret:
		f.pullSecret, cmd = f.pullSecret.Update(msg)
	}
	f.markDirtyIfChanged(before)
	return cmd
}

func (f *installConfigForm) markDirtyIfChanged(before string) {
	if f.currentStepValue() != before {
		f.markDirty(f.step)
	}
}

func (m *Model) loadBaseDomains() tea.Cmd {
	f := m.form
	if f == nil || f.currentPlatform() != awstypes.Name {
		return nil
	}

	region := strings.TrimSpace(f.region.Value())
	if region == "" {
		f.baseDomainLoading = false
		f.baseDomainErr = "AWS region is required before looking up base domains"
		f.baseDomainLoadedFor = ""
		return nil
	}
	if f.baseDomainLoadedFor == region && f.baseDomainErr == "" && len(f.baseDomainOptions) > 0 {
		return nil
	}

	var endpoints []awstypes.ServiceEndpoint
	if f.existing != nil && f.existing.Platform.AWS != nil {
		endpoints = f.existing.Platform.AWS.ServiceEndpoints
	}

	var staticCreds *awsconfig.StaticCredentials
	if f.needsAWSCreds {
		accessKey := strings.TrimSpace(f.awsAccessKey.Value())
		secretKey := strings.TrimSpace(f.awsSecretKey.Value())
		sessionToken := strings.TrimSpace(f.awsSessionKey.Value())
		if accessKey == "" || secretKey == "" {
			f.baseDomainLoading = false
			f.baseDomainErr = "AWS access key ID and secret access key are required"
			if accessKey == "" {
				f.credsFocus = 0
			} else {
				f.credsFocus = 1
			}
			f.focusCurrent()
			return nil
		}
		staticCreds = &awsconfig.StaticCredentials{
			AccessKeyID:     accessKey,
			SecretAccessKey: secretKey,
			SessionToken:    sessionToken,
		}
	}

	f.baseDomainRequest++
	request := f.baseDomainRequest
	f.baseDomainLoading = true
	f.baseDomainErr = ""
	f.baseDomainLoadedFor = region

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		domains, err := awsconfig.GetBaseDomainOptions(ctx, awsconfig.EndpointOptions{
			Region:    region,
			Endpoints: endpoints,
		}, staticCreds)
		if err == nil && staticCreds != nil {
			// Make the credentials available to the in-process asset
			// generation that runs later in this session.
			_ = os.Setenv("AWS_ACCESS_KEY_ID", staticCreds.AccessKeyID)
			_ = os.Setenv("AWS_SECRET_ACCESS_KEY", staticCreds.SecretAccessKey)
			if staticCreds.SessionToken != "" {
				_ = os.Setenv("AWS_SESSION_TOKEN", staticCreds.SessionToken)
			} else {
				_ = os.Unsetenv("AWS_SESSION_TOKEN")
			}
		}
		return baseDomainsMsg{request: request, region: region, domains: domains, err: err}
	}
}

func (m *Model) handleBaseDomains(msg baseDomainsMsg) (tea.Model, tea.Cmd) {
	f := m.form
	if f == nil {
		return m, nil
	}
	if msg.request != f.baseDomainRequest {
		return m, nil
	}

	f.baseDomainLoading = false
	f.baseDomainLoadedFor = msg.region
	f.baseDomainOptions = nil
	f.baseDomainIdx = 0
	f.baseDomainErr = ""

	if msg.err != nil {
		f.baseDomainLoadedFor = ""
		f.needsAWSCreds = true
		f.credsFocus = 0
		if errors.Is(msg.err, awsconfig.ErrNoCredentials) {
			f.baseDomainErr = "AWS credentials are required to list Route 53 hosted zones"
		} else {
			f.baseDomainErr = msg.err.Error()
		}
		if m.screen == screenInstallConfig && f.step == stepBaseDomain {
			f.focusCurrent()
		}
		return m, nil
	}

	f.needsAWSCreds = false
	f.awsAccessKey.SetValue("")
	f.awsSecretKey.SetValue("")
	f.awsSessionKey.SetValue("")
	if len(msg.domains) == 0 {
		f.baseDomainErr = "no public Route 53 hosted zones found"
		if m.screen == screenInstallConfig && f.step == stepBaseDomain {
			f.focusCurrent()
		}
		return m, nil
	}

	f.baseDomainOptions = msg.domains
	existing := strings.TrimSpace(f.baseDomain.Value())
	if existing != "" {
		if idx := indexOf(f.baseDomainOptions, existing); idx >= 0 {
			f.baseDomainIdx = idx
		} else {
			// Keep an existing value even if the account cannot list it.
			f.baseDomainOptions = append([]string{existing}, f.baseDomainOptions...)
			f.baseDomainIdx = 0
		}
	}
	f.baseDomain.SetValue(f.baseDomainOptions[f.baseDomainIdx])

	if m.screen == screenInstallConfig && f.step == stepBaseDomain {
		f.focusCurrent()
	}
	return m, nil
}

type azureConnectedMsg struct {
	session *azureconfig.Session
	regions []string
	zones   []azureconfig.Zone
	err     error
}

func (m *Model) connectAzure() tea.Cmd {
	f := m.form
	if f == nil {
		return nil
	}

	creds := azureconfig.Credentials{}
	if f.azureFileCreds != nil {
		creds = *f.azureFileCreds
	}
	creds.SubscriptionID = strings.TrimSpace(f.azureSubscriptionID.Value())
	creds.TenantID = strings.TrimSpace(f.azureTenantID.Value())
	creds.ClientID = strings.TrimSpace(f.azureClientID.Value())
	creds.ClientSecret = strings.TrimSpace(f.azureClientSecret.Value())
	if creds.ClientSecret != "" {
		// A client secret takes precedence over certificate authentication.
		creds.ClientCertificatePath = ""
		creds.ClientCertificatePassword = ""
	}

	if creds.SubscriptionID == "" || creds.TenantID == "" {
		f.err = "Azure subscription ID and tenant ID are required"
		if creds.SubscriptionID == "" {
			f.azureCredsFocus = 0
		} else {
			f.azureCredsFocus = 1
		}
		f.focusCurrent()
		return nil
	}
	if (creds.ClientSecret != "" || creds.ClientCertificatePath != "") && creds.ClientID == "" {
		f.err = "Azure client ID is required with a client secret or certificate"
		f.azureCredsFocus = 2
		f.focusCurrent()
		return nil
	}

	f.azureConnecting = true
	f.err = ""

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		session, err := azureconfig.GetSessionWithCredentials(azuretypes.PublicCloud, "", &creds)
		if err != nil {
			return azureConnectedMsg{err: err}
		}
		regions, err := azureconfig.GetRegionOptions(ctx, session)
		if err != nil {
			return azureConnectedMsg{err: err}
		}
		zones, err := azureconfig.NewDNSConfig(session).GetBaseDomainOptions()
		if err != nil {
			return azureConnectedMsg{err: err}
		}
		if err := azureconfig.SaveCredentials(creds); err != nil {
			return azureConnectedMsg{err: fmt.Errorf("connected to Azure, but failed to save credentials: %w", err)}
		}
		return azureConnectedMsg{session: session, regions: regions, zones: zones}
	}
}

func (m *Model) handleAzureConnected(msg azureConnectedMsg) (tea.Model, tea.Cmd) {
	f := m.form
	if f == nil {
		return m, nil
	}

	f.azureConnecting = false
	if f.currentPlatform() != azuretypes.Name {
		return m, nil
	}
	if msg.err != nil {
		f.err = msg.err.Error()
		if f.skipAzureCredentials() {
			f.forceAzureCredentials = true
			f.step = stepAzureCredentials
		}
		if m.screen == screenInstallConfig && f.step == stepAzureCredentials {
			f.focusCurrent()
		}
		return m, nil
	}

	f.forceAzureCredentials = false
	f.resetDownstream(stepAzureCredentials)
	f.clearDirty()

	f.azureSession = msg.session
	f.azureRegions = msg.regions
	f.azureZones = msg.zones
	f.regionSearch.SetValue("")
	f.err = ""

	if len(msg.regions) > 0 {
		f.azureRegionIdx = 0
		if existing := strings.TrimSpace(f.region.Value()); existing != "" {
			if idx := indexOf(msg.regions, existing); idx >= 0 {
				f.azureRegionIdx = idx
			}
		}
		f.region.SetValue(msg.regions[f.azureRegionIdx])
	}

	f.baseDomainOptions = make([]string, 0, len(msg.zones))
	f.baseDomainIDs = make(map[string]string, len(msg.zones))
	for _, zone := range msg.zones {
		f.baseDomainOptions = append(f.baseDomainOptions, zone.Name)
		f.baseDomainIDs[zone.Name] = zone.ID
	}
	f.baseDomainIdx = 0
	if existing := strings.TrimSpace(f.baseDomain.Value()); existing != "" {
		if idx := indexOf(f.baseDomainOptions, existing); idx >= 0 {
			f.baseDomainIdx = idx
		}
	}
	if len(f.baseDomainOptions) > 0 {
		f.baseDomain.SetValue(f.baseDomainOptions[f.baseDomainIdx])
		f.baseDomainZoneID = f.baseDomainIDs[f.baseDomainOptions[f.baseDomainIdx]]
	}
	f.baseDomainErr = ""
	f.baseDomainLoadedFor = ""

	if m.screen == screenInstallConfig && f.step == stepAzureCredentials {
		f.step = stepRegion
		return m, m.afterStepChange()
	}
	return m, nil
}

func (m *Model) afterStepChange() tea.Cmd {
	f := m.form
	if f == nil {
		return nil
	}
	f.focusCurrent()
	if f.step == stepRegion && f.currentPlatform() == azuretypes.Name && f.azureSession == nil && !f.azureConnecting {
		cmd := m.connectAzure()
		if f.err != "" {
			f.forceAzureCredentials = true
			f.step = stepAzureCredentials
			f.focusCurrent()
			return nil
		}
		return cmd
	}
	if f.step == stepSSHKey {
		f.loadSSHKeys()
	}
	if f.step == stepBaseDomain && f.currentPlatform() == awstypes.Name {
		region := strings.TrimSpace(f.region.Value())
		if !f.baseDomainLoading && f.baseDomainLoadedFor != region {
			return m.loadBaseDomains()
		}
	}
	return nil
}

func (m *Model) openInstallConfigForm() {
	m.form = newInstallConfigForm(m.dir)
	m.form.setWidth(m.width - 30)
	m.screen = screenInstallConfig
}

func (m *Model) handleInstallFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	f := m.form
	if f == nil {
		m.clearStageQueue()
		m.screen = screenMenu
		return m, nil
	}

	switch msg.String() {
	case "esc":
		if f.step == stepPlatform {
			m.clearStageQueue()
			m.screen = screenMenu
			m.menuIndex = stageIndex(StageInstallConfig)
			return m, nil
		}
		f.prevStep()
		return m, m.afterStepChange()
	case "tab":
		if f.step == stepAzureCredentials {
			if f.azureConnecting {
				return m, nil
			}
			f.azureCredsFocus = (f.azureCredsFocus + 1) % 4
			f.focusCurrent()
			return m, nil
		}
		if f.step == stepBaseDomain && f.needsAWSCreds {
			f.credsFocus = (f.credsFocus + 1) % 3
			f.focusCurrent()
			return m, nil
		}
		f.nextStep()
		return m, m.afterStepChange()
	case "shift+tab":
		if f.step == stepAzureCredentials {
			if f.azureConnecting {
				return m, nil
			}
			f.azureCredsFocus = (f.azureCredsFocus + 3) % 4
			f.focusCurrent()
			return m, nil
		}
		if f.step == stepBaseDomain && f.needsAWSCreds {
			f.credsFocus = (f.credsFocus + 2) % 3
			f.focusCurrent()
			return m, nil
		}
		f.prevStep()
		return m, m.afterStepChange()
	case "ctrl+s":
		return m, m.submitInstallConfigForm()
	case "ctrl+r":
		if f.step == stepBaseDomain && f.currentPlatform() == awstypes.Name {
			f.baseDomainLoadedFor = ""
			return m, m.loadBaseDomains()
		}
		return m, nil
	case "ctrl+g":
		if f.step == stepBaseDomain && f.currentPlatform() == awstypes.Name {
			f.needsAWSCreds = false
			f.focusCurrent()
		}
		return m, nil
	case "backspace", "ctrl+h":
		if f.step == stepRegion && f.currentPlatform() == azuretypes.Name && len(f.azureRegions) > 0 {
			var cmd tea.Cmd
			f.regionSearch, cmd = f.regionSearch.Update(msg)
			f.applyRegionFilter()
			return m, cmd
		}
	case "enter":
		if f.step == stepAzureCredentials {
			if f.azureConnecting {
				return m, nil
			}
			if f.azureCredsFocus < 3 {
				f.azureCredsFocus++
				f.focusCurrent()
				return m, nil
			}
			return m, m.connectAzure()
		}
		if f.step == stepSSHKey {
			f.applySSHKeyChoice()
			f.nextStep()
			return m, m.afterStepChange()
		}
		if f.step == stepPullSecret {
			return m, m.submitInstallConfigForm()
		}
		if f.step == stepBaseDomain && f.needsAWSCreds {
			if f.credsFocus < 2 {
				f.credsFocus++
				f.focusCurrent()
				return m, nil
			}
			return m, m.loadBaseDomains()
		}
		if f.step == stepBaseDomain && f.baseDomainLoading {
			return m, nil
		}
		if err := f.validateCurrentStep(); err != nil {
			f.err = err.Error()
			return m, nil
		}
		if f.step == stepBaseDomain && f.currentPlatform() == awstypes.Name && f.baseDomainErr != "" {
			// Manual fallback in the error case still requires a value.
			if strings.TrimSpace(f.baseDomain.Value()) == "" {
				f.err = "base domain is required"
				return m, nil
			}
		}
		f.err = ""
		f.nextStep()
		return m, m.afterStepChange()
	}

	switch f.step {
	case stepPlatform:
		switch msg.String() {
		case "left", "up":
			if f.platformIdx < 0 {
				f.platformIdx = 0
			} else {
				f.platformIdx = (f.platformIdx - 1 + len(f.platforms)) % len(f.platforms)
			}
			f.err = ""
			f.resetBaseDomainLookup()
			f.markDirty(stepPlatform)
			return m, nil
		case "right", "down":
			if f.platformIdx < 0 {
				f.platformIdx = 0
			} else {
				f.platformIdx = (f.platformIdx + 1) % len(f.platforms)
			}
			f.err = ""
			f.resetBaseDomainLookup()
			f.markDirty(stepPlatform)
			return m, nil
		}
	case stepPublish:
		switch msg.String() {
		case "left", "up", "right", "down":
			f.publishIdx = (f.publishIdx + 1) % len(f.publishes)
			f.markDirty(stepPublish)
			return m, nil
		}
	case stepSSHKey:
		if len(f.sshKeyChoices) > 0 {
			switch msg.String() {
			case "left", "up":
				f.sshKeyIdx = (f.sshKeyIdx - 1 + len(f.sshKeyChoices)) % len(f.sshKeyChoices)
				f.markDirty(stepSSHKey)
				return m, nil
			case "right", "down":
				f.sshKeyIdx = (f.sshKeyIdx + 1) % len(f.sshKeyChoices)
				f.markDirty(stepSSHKey)
				return m, nil
			}
		}
	case stepRegion:
		if f.currentPlatform() == azuretypes.Name && len(f.azureRegions) > 0 {
			filtered := f.filteredAzureRegions()
			if len(filtered) == 0 {
				return m, nil
			}
			switch msg.String() {
			case "left", "up":
				f.azureRegionIdx = (f.azureRegionIdx - 1 + len(filtered)) % len(filtered)
				f.region.SetValue(filtered[f.azureRegionIdx])
				f.markDirty(stepRegion)
				return m, nil
			case "right", "down":
				f.azureRegionIdx = (f.azureRegionIdx + 1) % len(filtered)
				f.region.SetValue(filtered[f.azureRegionIdx])
				f.markDirty(stepRegion)
				return m, nil
			}
		}
	case stepBaseDomain:
		if f.baseDomainIsSelect() {
			switch msg.String() {
			case "left", "up":
				f.baseDomainIdx = (f.baseDomainIdx - 1 + len(f.baseDomainOptions)) % len(f.baseDomainOptions)
				f.baseDomain.SetValue(f.baseDomainOptions[f.baseDomainIdx])
				if f.currentPlatform() == azuretypes.Name {
					f.baseDomainZoneID = f.baseDomainIDs[f.baseDomainOptions[f.baseDomainIdx]]
				}
				f.markDirty(stepBaseDomain)
				return m, nil
			case "right", "down":
				f.baseDomainIdx = (f.baseDomainIdx + 1) % len(f.baseDomainOptions)
				f.baseDomain.SetValue(f.baseDomainOptions[f.baseDomainIdx])
				if f.currentPlatform() == azuretypes.Name {
					f.baseDomainZoneID = f.baseDomainIDs[f.baseDomainOptions[f.baseDomainIdx]]
				}
				f.markDirty(stepBaseDomain)
				return m, nil
			}
		}
		if f.baseDomainLoading {
			return m, nil
		}
	}

	return m, f.updateFocused(msg)
}

func (f *installConfigForm) validateCurrentStep() error {
	switch f.step {
	case stepPlatform:
		if f.currentPlatform() == "" {
			return errors.New("select a platform")
		}
	case stepRegion:
		if strings.TrimSpace(f.region.Value()) == "" {
			if f.currentPlatform() == azuretypes.Name {
				return errors.New("Azure region is required")
			}
			return errors.New("AWS region is required")
		}
	case stepBaseDomain:
		if f.needsAWSCreds {
			return nil
		}
		if strings.TrimSpace(f.baseDomain.Value()) == "" {
			return errors.New("base domain is required")
		}
	case stepClusterName:
		if strings.TrimSpace(f.clusterName.Value()) == "" {
			return errors.New("cluster name is required")
		}
	}
	return nil
}

func (f *installConfigForm) buildConfig() (*types.InstallConfig, error) {
	clusterName := strings.TrimSpace(f.clusterName.Value())
	if clusterName == "" {
		return nil, errors.New("cluster name is required")
	}

	baseDomain := strings.TrimSpace(f.baseDomain.Value())
	if baseDomain == "" {
		return nil, errors.New("base domain is required")
	}

	pullSecret := strings.TrimSpace(f.pullSecret.Value())
	if pullSecret == "" {
		return nil, errors.New("pull secret is required")
	}
	if !json.Valid([]byte(pullSecret)) {
		return nil, errors.New("pull secret must be valid JSON")
	}

	sshKey := f.existingSSHKey
	switch {
	case f.sshKeyNone:
		sshKey = ""
	case f.sshKeyKeepExisting:
		sshKey = f.existingSSHKey
	default:
		if path := strings.TrimSpace(f.sshKeyPath.Value()); path != "" {
			expanded, err := expandPath(path)
			if err != nil {
				return nil, fmt.Errorf("cannot resolve SSH public key path %q: %w", path, err)
			}
			data, err := os.ReadFile(expanded)
			if err != nil {
				return nil, fmt.Errorf("cannot read SSH public key %q: %w", path, err)
			}
			sshKey = strings.TrimSpace(string(data))
		}
	}

	platform := f.currentPlatform()
	region := strings.TrimSpace(f.region.Value())
	if (platform == awstypes.Name || platform == azuretypes.Name) && region == "" {
		return nil, fmt.Errorf("region is required for platform %s", platform)
	}

	ic := f.existing
	if ic == nil {
		ic = &types.InstallConfig{}
	}
	ic.TypeMeta = metav1.TypeMeta{APIVersion: types.InstallConfigVersion}
	ic.ObjectMeta.Name = clusterName
	ic.BaseDomain = baseDomain
	ic.PullSecret = pullSecret
	ic.SSHKey = sshKey
	ic.Publish = types.PublishingStrategy(f.currentPublish())

	if err := setFormPlatform(ic, platform, region); err != nil {
		return nil, err
	}
	if platform == azuretypes.Name {
		if ic.Platform.Azure == nil {
			ic.Platform.Azure = &azuretypes.Platform{}
		}
		ic.Platform.Azure.Region = region
		if ic.Platform.Azure.CloudName == "" {
			ic.Platform.Azure.CloudName = azuretypes.PublicCloud
		}
		if f.baseDomainZoneID != "" {
			if err := ic.Platform.Azure.SetBaseDomain(f.baseDomainZoneID); err != nil {
				return nil, err
			}
		}
	}
	defaults.SetInstallConfigDefaults(ic)
	return ic, nil
}

func setFormPlatform(ic *types.InstallConfig, platform, region string) error {
	previous := ic.Platform
	ic.Platform = types.Platform{}

	switch platform {
	case awstypes.Name:
		ic.Platform.AWS = previous.AWS
		if ic.Platform.AWS == nil {
			ic.Platform.AWS = &awstypes.Platform{}
		}
		ic.Platform.AWS.Region = region
	case azuretypes.Name:
		ic.Platform.Azure = previous.Azure
		if ic.Platform.Azure == nil {
			ic.Platform.Azure = &azuretypes.Platform{}
		}
	case baremetaltypes.Name:
		ic.Platform.BareMetal = previous.BareMetal
		if ic.Platform.BareMetal == nil {
			ic.Platform.BareMetal = &baremetaltypes.Platform{}
		}
	case gcptypes.Name:
		ic.Platform.GCP = previous.GCP
		if ic.Platform.GCP == nil {
			ic.Platform.GCP = &gcptypes.Platform{}
		}
	case ibmcloudtypes.Name:
		ic.Platform.IBMCloud = previous.IBMCloud
		if ic.Platform.IBMCloud == nil {
			ic.Platform.IBMCloud = &ibmcloudtypes.Platform{}
		}
	case nonetypes.Name:
		ic.Platform.None = &nonetypes.Platform{}
	case nutanixtypes.Name:
		ic.Platform.Nutanix = previous.Nutanix
		if ic.Platform.Nutanix == nil {
			ic.Platform.Nutanix = &nutanixtypes.Platform{}
		}
	case openstacktypes.Name:
		ic.Platform.OpenStack = previous.OpenStack
		if ic.Platform.OpenStack == nil {
			ic.Platform.OpenStack = &openstacktypes.Platform{}
		}
	case powervctypes.Name:
		ic.Platform.PowerVC = previous.PowerVC
		if ic.Platform.PowerVC == nil {
			ic.Platform.PowerVC = &powervctypes.Platform{}
		}
	case powervstypes.Name:
		ic.Platform.PowerVS = previous.PowerVS
		if ic.Platform.PowerVS == nil {
			ic.Platform.PowerVS = &powervstypes.Platform{}
		}
	case vspheretypes.Name:
		ic.Platform.VSphere = previous.VSphere
		if ic.Platform.VSphere == nil {
			ic.Platform.VSphere = &vspheretypes.Platform{}
		}
	default:
		return fmt.Errorf("platform %q is not supported by the TUI form", platform)
	}
	return nil
}

func (m *Model) submitInstallConfigForm() tea.Cmd {
	f := m.form
	if f == nil {
		return nil
	}

	ic, err := f.buildConfig()
	if err != nil {
		f.err = err.Error()
		return nil
	}

	data, err := yaml.Marshal(ic)
	if err != nil {
		f.err = fmt.Sprintf("cannot marshal install-config: %v", err)
		return nil
	}
	if err := os.WriteFile(filepath.Join(m.dir, installConfigName), data, 0o640); err != nil {
		f.err = fmt.Sprintf("cannot write %s: %v", installConfigName, err)
		return nil
	}

	f.existing = ic
	f.existingSSHKey = ic.SSHKey
	f.err = ""
	return m.runTargets(StageInstallConfig, targetassets.InstallConfig)
}

func (f *installConfigForm) setWidth(width int) {
	width = clamp(width, 20, 100)
	f.clusterName.Width = width
	f.baseDomain.Width = width
	f.region.Width = width
	f.regionSearch.Width = width
	f.awsAccessKey.Width = width
	f.awsSecretKey.Width = width
	f.awsSessionKey.Width = width
	f.azureSubscriptionID.Width = width
	f.azureTenantID.Width = width
	f.azureClientID.Width = width
	f.azureClientSecret.Width = width
	f.sshKeyPath.Width = width
	f.pullSecret.Width = width
}

func indexOf(values []string, want string) int {
	for i, value := range values {
		if value == want {
			return i
		}
	}
	return -1
}

func indexOfStep(values []installFormStep, want installFormStep) int {
	for i, value := range values {
		if value == want {
			return i
		}
	}
	return -1
}
