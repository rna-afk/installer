package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"

	installercommand "github.com/openshift/installer/cmd/openshift-install/command"
	"github.com/openshift/installer/pkg/asset"
	clusterasset "github.com/openshift/installer/pkg/asset/cluster"
	"github.com/openshift/installer/pkg/asset/installconfig"
	assetstore "github.com/openshift/installer/pkg/asset/store"
	targetassets "github.com/openshift/installer/pkg/asset/targets"
	"github.com/openshift/installer/pkg/clusterapi"
	"github.com/openshift/installer/pkg/gather/service"
	"github.com/openshift/installer/pkg/types"
)

type logLineMsg struct {
	runID int
	line  string
}

type runDoneMsg struct {
	runID     int
	stage     Stage
	err       error
	cancelled bool
}

type stageAdvanceMsg struct {
	runID int
}

type commandDoneMsg struct {
	runID     int
	title     string
	detail    string
	returnTo  screen
	err       error
	cancelled bool
}

func targetsForStage(stage Stage) ([]asset.WritableAsset, bool) {
	switch stage {
	case StageInstallConfig:
		return targetassets.InstallConfig, true
	case StageManifests:
		return targetassets.Manifests, true
	case StageIgnitionConfigs:
		return targetassets.IgnitionConfigs, true
	case StageCluster:
		return targetassets.Cluster, true
	default:
		return nil, false
	}
}

// startStage opens the install-config form for that stage, and runs the other
// stages through the installer's asset store.
func (m *Model) startStage(stage Stage) tea.Cmd {
	if stage == StageInstallConfig {
		m.openInstallConfigForm()
		return nil
	}

	assets, ok := targetsForStage(stage)
	if !ok {
		return nil
	}
	return m.runTargets(stage, assets)
}

// runTargets runs the given asset targets in-process and streams their logs to
// the TUI.
func (m *Model) runTargets(stage Stage, assets []asset.WritableAsset) tea.Cmd {
	id := m.nextRunID()
	m.prepareRun("Running " + stage.Title())
	if stage == StageCluster {
		installercommand.RootOpts.Dir = m.dir
		installercommand.SkipPasswordPrintFlag = m.opts.skipPasswordPrint
		clusterasset.InstallDir = m.dir
		m.clusterDiagram = newClusterDiagram(id)
		m.screen = screenCluster
		m.resize()
	} else {
		m.clusterDiagram = nil
	}
	if m.uiLog != nil {
		m.uiLog.SetRunID(id)
	}

	run := func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		m.setCancel(cancel)
		defer cancel()

		logRun(stage, m.dir)
		if stage == StageCluster {
			defer func() {
				clusterapi.System().Teardown()
				clusterapi.System().CleanEtcd()
			}()
		}

		err := assetstore.NewAssetsFetcher(m.dir).FetchAndPersist(ctx, assets)
		if err == nil && stage == StageCluster {
			err = runClusterPost(ctx, m.dir, m.opts.skipPasswordPrint)
		}

		if m.uiLog != nil {
			m.uiLog.SetRunID(0)
		}
		return runDoneMsg{
			runID:     id,
			stage:     stage,
			err:       err,
			cancelled: ctx.Err() != nil,
		}
	}
	return tea.Batch(run, tickLogs())
}

func logRun(stage Stage, dir string) {
	logrus.Debugf("Running: openshift-install-tui create %s --dir %s", stage.String(), dir)
}

func (m *Model) nextRunID() int {
	m.runID++
	return m.runID
}

func (m *Model) prepareRun(label string) {
	m.screen = screenRunning
	m.running = true
	m.advanceRunID = 0
	m.runLabel = label
	m.logLines = nil
	m.logDirty = false
	m.logViewport.SetContent("")
	m.logViewport.GotoTop()
	m.clusterLogViewport.SetContent("")
	m.clusterLogViewport.GotoTop()
	m.statusMessage = ""
}

// runTask runs an installer operation in-process and streams its logrus output
// to the TUI.
func (m *Model) runTask(title string, returnTo screen, fn func(context.Context) (string, error)) tea.Cmd {
	return m.runTaskInternal(title, returnTo, true, fn)
}

// runTaskUncancellable is for operations that cannot be interrupted once
// started, such as cluster destruction.
func (m *Model) runTaskUncancellable(title string, returnTo screen, fn func(context.Context) (string, error)) tea.Cmd {
	return m.runTaskInternal(title, returnTo, false, fn)
}

func (m *Model) runTaskInternal(title string, returnTo screen, cancellable bool, fn func(context.Context) (string, error)) tea.Cmd {
	id := m.nextRunID()
	m.prepareRun(title)
	if m.uiLog != nil {
		m.uiLog.SetRunID(id)
	}

	run := func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		if cancellable {
			m.setCancel(cancel)
		}
		defer cancel()

		detail, err := fn(ctx)

		if m.uiLog != nil {
			m.uiLog.SetRunID(0)
		}
		return commandDoneMsg{
			runID:     id,
			title:     title,
			detail:    detail,
			returnTo:  returnTo,
			err:       err,
			cancelled: cancellable && ctx.Err() != nil,
		}
	}
	return tea.Batch(run, tickLogs())
}

func (m *Model) startGather(returnTo screen) tea.Cmd {
	return m.runTask("Gather bootstrap logs", returnTo, func(ctx context.Context) (string, error) {
		bundlePath, err := runGatherBootstrap(ctx, m.dir, gatherOptions{
			Bootstrap: strings.TrimSpace(m.opts.bootstrap),
			Masters:   append([]string{}, m.opts.masters...),
			SSHKeys:   append([]string{}, m.opts.keys...),
		})
		if err != nil {
			return "", err
		}
		if !m.opts.skipAnalysis {
			if err := service.AnalyzeGatherBundle(bundlePath); err != nil {
				return "", err
			}
		}
		logrus.Infof("Bootstrap gather logs captured here %q", bundlePath)
		return bundlePath, nil
	})
}

func (m *Model) startWaitFor() tea.Cmd {
	if m.dir == "" {
		m.pendingCommand = commandWaitFor
		m.statusMessage = ""
		m.screen = screenDir
		m.dirInput.Focus()
		return nil
	}
	return m.executeWaitFor()
}

func (m *Model) executeWaitFor() tea.Cmd {
	stage := waitForStages[m.waitForIndex]
	return m.runTask("Wait for "+stage.Title(), screenCommand, func(ctx context.Context) (string, error) {
		config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(m.dir, "auth", "kubeconfig"))
		if err != nil {
			return "", fmt.Errorf("loading kubeconfig: %w", err)
		}

		installercommand.RootOpts.Dir = m.dir
		installercommand.SkipPasswordPrintFlag = m.opts.skipPasswordPrint

		switch stage {
		case waitForBootstrapComplete:
			if err := runWaitForBootstrapComplete(ctx, config, m.dir); err != nil {
				return "", err
			}
			return "Bootstrap complete.", nil
		case waitForInstallComplete:
			store, err := assetstore.NewStore(m.dir)
			if err != nil {
				return "", err
			}
			var ic *types.InstallConfig
			if a, err := store.Load(&installconfig.InstallConfig{}); err == nil && a != nil {
				ic = a.(*installconfig.InstallConfig).Config
			}
			if err := installercommand.WaitForInstallComplete(ctx, config, waitOptionsForInstallConfig(ic)); err != nil {
				return "", err
			}
			return "Install complete.", nil
		default:
			return "", fmt.Errorf("unknown wait-for stage")
		}
	})
}

func (m *Model) handleCommandDone(msg commandDoneMsg) (tea.Model, tea.Cmd) {
	m.running = false
	m.setCancel(nil)
	if m.uiLog != nil {
		m.uiLog.SetRunID(0)
	}
	m.flushLogs()
	m.lastStage = StageNone
	if detection, err := Detect(m.dir); err == nil {
		m.detection = detection
	}

	if msg.cancelled {
		logrus.Warnf("%s cancelled", msg.title)
		m.screen = screenFailed
		m.errMessage = "command cancelled"
		m.details = nil
		return m, nil
	}
	if msg.err != nil {
		logrus.Errorf("%s failed: %v", msg.title, msg.err)
		m.screen = screenFailed
		m.errMessage = msg.err.Error()
		m.details = tail(m.logLines, 10)
		return m, nil
	}

	logrus.Infof("%s completed", msg.title)
	if msg.title == "Destroy cluster" && m.dir != "" {
		m.completionTitle = "Destroy cluster completed"
		m.completionBody = []string{
			msg.detail,
			"",
			"Delete " + m.dir + "? [y]es / [n]o",
		}
		m.screen = screenDeleteDir
		return m, nil
	}

	body := []string{}
	if msg.detail != "" {
		body = append(body, msg.detail)
	}
	if len(body) == 0 {
		body = append(body, "Command completed.")
	}
	m.showSuccessFrom(msg.title, body, msg.returnTo)
	return m, nil
}

func (m *Model) startAnalyze(returnTo screen) tea.Cmd {
	logrus.Info("done")
	m.showPlaceholderFrom("Analyze", []string{"done"}, returnTo)
	return nil
}
