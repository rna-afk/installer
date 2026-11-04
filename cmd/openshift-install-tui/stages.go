package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	stateFileName     = ".openshift_install_state.json"
	logFileName       = ".openshift_install.log"
	installConfigName = "install-config.yaml"
	metadataName      = "metadata.json"
)

// Stage is one of the installer asset-generation phases.
type Stage int

const (
	StageNone Stage = iota
	StageInstallConfig
	StageManifests
	StageIgnitionConfigs
	StageCluster
)

// stages is the canonical execution order.
var stages = []Stage{
	StageInstallConfig,
	StageManifests,
	StageIgnitionConfigs,
	StageCluster,
}

func (s Stage) String() string {
	switch s {
	case StageInstallConfig:
		return "install-config"
	case StageManifests:
		return "manifests"
	case StageIgnitionConfigs:
		return "ignition-configs"
	case StageCluster:
		return "cluster"
	default:
		return "none"
	}
}

// Title is the human-friendly name shown in the UI.
func (s Stage) Title() string {
	switch s {
	case StageInstallConfig:
		return "Install Config"
	case StageManifests:
		return "Manifests"
	case StageIgnitionConfigs:
		return "Ignition Configs"
	case StageCluster:
		return "Install Cluster"
	default:
		return "None"
	}
}

// CreateArgs returns the openshift-install arguments for this stage.
func (s Stage) CreateArgs(dir string) []string {
	return []string{"create", s.String(), "--dir", dir}
}

// stagesFrom returns the stages to run to reach target from current,
// including target. If target is at or before current, only target is
// returned so an explicit rerun does not repeat later stages.
func stagesFrom(current, target Stage) []Stage {
	if target == StageNone {
		return nil
	}
	if current >= target {
		return []Stage{target}
	}
	out := make([]Stage, 0, int(target-current))
	for stage := current + 1; stage <= target; stage++ {
		out = append(out, stage)
	}
	return out
}

// Next returns the stage that normally follows this one.
func (s Stage) Next() Stage {
	switch s {
	case StageInstallConfig:
		return StageManifests
	case StageManifests:
		return StageIgnitionConfigs
	case StageIgnitionConfigs:
		return StageCluster
	default:
		return StageNone
	}
}

// LogError is one error-ish line found in the installer log.
type LogError struct {
	Line int
	Text string
}

// Detection is the result of inspecting an assets directory.
type Detection struct {
	Dir string

	// New is true when the directory contains no installer state at all.
	New bool
	// Last is the most advanced stage with evidence of completion.
	Last Stage
	// LastRun is the stage of the last "Running: openshift-install create ..."
	// entry in the log, if any.
	LastRun Stage
	// LastClusterRun is true when a cluster stage was attempted.
	LastClusterRun bool

	Evidence    []string
	StateKeys   []string
	HasMetadata bool
	LogBundle   string
	Errors      []LogError
}

// Detect inspects dir and reports how far the installer got.
func Detect(dir string) (*Detection, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return &Detection{Dir: dir, New: true}, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	d := &Detection{Dir: dir}

	stateStage, keys := detectState(dir)
	d.StateKeys = keys
	if stateStage != StageNone {
		d.Evidence = append(d.Evidence, fmt.Sprintf("state file records stage %q", stateStage))
	}

	diskStage, diskEvidence := detectDisk(dir)
	d.Evidence = append(d.Evidence, diskEvidence...)

	lastRun, clusterRun, logEvidence := detectLog(dir)
	d.LastRun = lastRun
	d.LastClusterRun = clusterRun
	d.Evidence = append(d.Evidence, logEvidence...)

	// On-disk artifacts are the source of truth for completed stages. A stale
	// state file can contain assets from an earlier run whose files were
	// removed, so state keys must not advance the stage beyond what is present
	// on disk. If there is no on-disk evidence, fall back to the state file,
	// then to the log (stepping back one stage because a log entry only proves
	// an attempt).
	switch {
	case diskStage != StageNone:
		d.Last = diskStage
	case stateStage != StageNone:
		d.Last = stateStage
	case lastRun != StageNone:
		d.Last = prevStage(lastRun)
	}

	d.HasMetadata = fileExists(filepath.Join(dir, metadataName))
	d.LogBundle = findLogBundle(dir)
	d.New = d.Last == StageNone && len(d.Evidence) == 0

	if d.Last == StageCluster || d.LastClusterRun {
		errs, err := scanLogErrors(dir)
		if err != nil {
			return nil, err
		}
		d.Errors = errs
	}

	return d, nil
}

// Next returns the stage the user would normally continue with.
func (d *Detection) Next() Stage {
	if d.Last == StageNone {
		return StageInstallConfig
	}
	return d.Last.Next()
}

func detectState(dir string) (Stage, []string) {
	path := filepath.Join(dir, stateFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return StageNone, nil
	}

	var assets map[string]json.RawMessage
	if err := json.Unmarshal(data, &assets); err != nil {
		return StageNone, nil
	}

	keys := make([]string, 0, len(assets))
	for k := range assets {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	stage := StageNone
	for _, k := range keys {
		if s := stageFromStateKey(k); s > stage {
			stage = s
		}
	}
	return stage, keys
}

func stageFromStateKey(key string) Stage {
	switch {
	// Cluster target assets.
	case strings.Contains(key, "cluster.Cluster"),
		strings.Contains(key, "tfvars.TerraformVariables"):
		return StageCluster

	// Ignition-configs target assets (cluster.Metadata is written here too).
	case strings.Contains(key, "bootstrap.Bootstrap"),
		strings.Contains(key, "machine.Master"),
		strings.Contains(key, "machine.Worker"),
		strings.Contains(key, "machine.Arbiter"),
		strings.Contains(key, "kubeconfig.AdminClient"),
		strings.Contains(key, "password.KubeadminPassword"),
		strings.Contains(key, "cluster.Metadata"):
		return StageIgnitionConfigs

	// Manifests target assets.
	case strings.Contains(key, "manifests.Manifests"),
		strings.Contains(key, "manifests.Openshift"),
		strings.Contains(key, "clusterapi.Cluster"),
		strings.Contains(key, "machines."),
		strings.Contains(key, "tls.SignerKeyParams"):
		return StageManifests

	// The install-config target itself.
	case strings.HasSuffix(key, "installconfig.InstallConfig"):
		return StageInstallConfig
	}
	return StageNone
}

func detectDisk(dir string) (Stage, []string) {
	var stage Stage
	var evidence []string

	add := func(s Stage, reason string) {
		if s > stage {
			stage = s
		}
		evidence = append(evidence, reason)
	}

	// Cluster infrastructure is provisioned from these files.
	for _, name := range []string{
		"terraform.platform.auto.tfvars.json",
		".terraform.platform.tfstate",
	} {
		if fileExists(filepath.Join(dir, name)) {
			add(StageCluster, name+" is present")
			break
		}
	}
	if isDir(filepath.Join(dir, ".terraform")) {
		add(StageCluster, ".terraform/ is present")
	}

	// Ignition configs.
	if fileExists(filepath.Join(dir, metadataName)) {
		add(StageIgnitionConfigs, metadataName+" is present")
	}
	for _, name := range []string{
		"bootstrap.ign",
		"master.ign",
		"worker.ign",
		filepath.Join("auth", "kubeconfig"),
	} {
		if fileExists(filepath.Join(dir, name)) {
			add(StageIgnitionConfigs, name+" is present")
		}
	}

	// Manifests (consumed once ignition configs are generated).
	for _, name := range []string{"manifests", "openshift"} {
		if isDir(filepath.Join(dir, name)) {
			add(StageManifests, name+"/ is present")
		}
	}

	if fileExists(filepath.Join(dir, installConfigName)) {
		add(StageInstallConfig, installConfigName+" is present")
	}

	return stage, evidence
}

func detectLog(dir string) (Stage, bool, []string) {
	path := filepath.Join(dir, logFileName)
	f, err := os.Open(path)
	if err != nil {
		return StageNone, false, nil
	}
	defer f.Close()

	var (
		lastRun    Stage
		clusterRun bool
		evidence   []string
	)

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !isCreateRunLine(line) {
			continue
		}
		stage := stageFromCommandLine(line)
		if stage == StageNone {
			continue
		}
		lastRun = stage
		if stage == StageCluster {
			clusterRun = true
		}
		evidence = append(evidence, "log shows: "+line)
	}

	return lastRun, clusterRun, dedupe(evidence)
}

func prevStage(s Stage) Stage {
	switch s {
	case StageCluster:
		return StageIgnitionConfigs
	case StageIgnitionConfigs:
		return StageManifests
	case StageManifests:
		return StageInstallConfig
	default:
		return StageNone
	}
}

func isCreateRunLine(line string) bool {
	return strings.Contains(line, "Running: openshift-install create ") ||
		strings.Contains(line, "Running: openshift-install-tui create ")
}

func stageFromCommandLine(line string) Stage {
	fields := strings.Fields(line)
	for i, f := range fields {
		if f != "create" || i+1 >= len(fields) {
			continue
		}
		switch fields[i+1] {
		case "install-config":
			return StageInstallConfig
		case "manifests":
			return StageManifests
		case "ignition-configs":
			return StageIgnitionConfigs
		case "cluster":
			return StageCluster
		}
	}
	return StageNone
}

// scanLogErrors returns error lines from the last cluster run onward. When no
// cluster run is recorded, the whole log is scanned.
func scanLogErrors(dir string) ([]LogError, error) {
	path := filepath.Join(dir, logFileName)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	start := 0
	hasClusterRun := false
	for i, line := range lines {
		if isCreateRunLine(line) && stageFromCommandLine(line) == StageCluster {
			start = i
			hasClusterRun = true
		}
	}
	if !hasClusterRun {
		start = 0
	}

	var errs []LogError
	for i := start; i < len(lines); i++ {
		if isErrorLine(lines[i]) {
			errs = append(errs, LogError{Line: i + 1, Text: strings.TrimSpace(lines[i])})
		}
	}

	// Keep the tail so the UI stays readable.
	const maxErrors = 50
	if len(errs) > maxErrors {
		errs = errs[len(errs)-maxErrors:]
	}
	return errs, nil
}

// scanErrorsSinceLastRun returns error lines emitted after the most recent
// "Running: openshift-install create ..." entry.
func scanErrorsSinceLastRun(dir string) ([]LogError, error) {
	path := filepath.Join(dir, logFileName)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	start := 0
	for i, line := range lines {
		if isCreateRunLine(line) {
			start = i
		}
	}

	var errs []LogError
	for i := start; i < len(lines); i++ {
		if isErrorLine(lines[i]) {
			errs = append(errs, LogError{Line: i + 1, Text: strings.TrimSpace(lines[i])})
		}
	}
	return errs, nil
}

func isErrorLine(line string) bool {
	for _, marker := range []string{
		"level=error",
		"level=fatal",
		"ERRO[",
		"FATA[",
	} {
		if strings.Contains(line, marker) {
			return true
		}
	}
	for _, phrase := range []string{
		"failed to create cluster",
		"Bootstrap failed to complete",
		"failed to provision control-plane machines",
		"bootstrap failed to complete",
	} {
		if strings.Contains(line, phrase) {
			return true
		}
	}
	return false
}

func findLogBundle(dir string) string {
	matches, err := filepath.Glob(filepath.Join(dir, "log-bundle-*.tar.gz"))
	if err != nil || len(matches) == 0 {
		return ""
	}
	sort.Slice(matches, func(i, j int) bool {
		fi, errI := os.Stat(matches[i])
		fj, errJ := os.Stat(matches[j])
		if errI != nil || errJ != nil {
			return matches[i] > matches[j]
		}
		return fi.ModTime().After(fj.ModTime())
	})
	return matches[0]
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func dedupe(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
