package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestStageCompletionShowsSuccessScreen(t *testing.T) {
	m := newModel(options{})
	m.dir = t.TempDir()
	writeFile(t, m.dir, installConfigName, "apiVersion: v1\n")
	m.detection = &Detection{Dir: m.dir, Last: StageInstallConfig}

	if _, _ = m.handleRunDone(runDoneMsg{stage: StageInstallConfig}); m.screen != screenStageSuccess {
		t.Fatalf("screen = %v, want success screen", m.screen)
	}
	if !strings.Contains(m.completionTitle, "Install Config") {
		t.Fatalf("success message = %q", m.completionTitle)
	}
}

func TestDirectoryWithOnlyLogShowsCommandMenu(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, logFileName,
		`time="t" level=debug msg="Running: openshift-install-tui create install-config --dir ."`+"\n"+
			`time="t" level=error msg="install-config failed"`+"\n")

	m := newModel(options{})
	m.dirInput.SetValue(dir)
	m.openDirectory()

	if m.screen != screenCommand {
		t.Fatalf("screen = %v, want command menu", m.screen)
	}
}

func TestCommandMenuNavigation(t *testing.T) {
	m := newModel(options{})

	if _, _ = m.handleCommandKey(tea.KeyMsg{Type: tea.KeyDown}); m.commandIndex != 1 {
		t.Fatalf("commandIndex = %d, want 1", m.commandIndex)
	}
	if _, _ = m.handleCommandKey(tea.KeyMsg{Type: tea.KeyUp}); m.commandIndex != 0 {
		t.Fatalf("commandIndex = %d, want 0", m.commandIndex)
	}
}

func TestPendingCommandRunsAfterDirectorySelection(t *testing.T) {
	dir := t.TempDir()
	m := newModel(options{})

	m.runCommand(commandGather)
	if m.screen != screenDir {
		t.Fatalf("screen = %v, want directory prompt", m.screen)
	}

	m.dirInput.SetValue(dir)
	m.openDirectory()

	if m.screen != screenRunning {
		t.Fatalf("screen = %v, want running gather", m.screen)
	}
	if m.runLabel != "Gather bootstrap logs" {
		t.Fatalf("runLabel = %q, want gather", m.runLabel)
	}
}

func TestWaitForAsksStageThenDirectory(t *testing.T) {
	m := newModel(options{})

	m.runCommand(commandWaitFor)
	if m.screen != screenWaitFor {
		t.Fatalf("screen = %v, want wait-for stage picker", m.screen)
	}

	if _, _ = m.handleWaitForKey(tea.KeyMsg{Type: tea.KeyEnter}); m.screen != screenDir {
		t.Fatalf("screen = %v, want directory prompt", m.screen)
	}
	if m.pendingCommand != commandWaitFor {
		t.Fatalf("pendingCommand = %v, want wait-for", m.pendingCommand)
	}
}

func TestDestroyAsksForDirectory(t *testing.T) {
	m := newModel(options{})

	m.runCommand(commandDestroy)

	if m.screen != screenDir {
		t.Fatalf("screen = %v, want directory prompt", m.screen)
	}
	if m.pendingCommand != commandDestroy {
		t.Fatalf("pendingCommand = %v, want destroy", m.pendingCommand)
	}
}

func TestDestroyClearsDirectorySelection(t *testing.T) {
	dir := t.TempDir()
	m := newModel(options{})
	m.dir = dir
	m.detection = &Detection{Dir: dir, Last: StageCluster}
	m.dirInput.SetValue(dir)

	m.runCommand(commandDestroy)

	if m.screen != screenDir {
		t.Fatalf("screen = %v, want directory prompt", m.screen)
	}
	if m.dir != "" {
		t.Fatalf("dir = %q, want empty", m.dir)
	}
	if m.dirInput.Value() != "" {
		t.Fatalf("dir input = %q, want empty", m.dirInput.Value())
	}
	if m.detection != nil {
		t.Fatal("expected detection to be cleared")
	}
}

func TestCommandDoneShowsResult(t *testing.T) {
	m := newModel(options{})
	m.dir = t.TempDir()
	m.running = true
	m.screen = screenRunning

	m.handleCommandDone(commandDoneMsg{
		title:    "Gather bootstrap logs",
		detail:   "Bootstrap gather logs captured here \"bundle.tar.gz\"",
		returnTo: screenCommand,
	})

	if m.screen != screenPlaceholder {
		t.Fatalf("screen = %v, want result placeholder", m.screen)
	}
	if m.placeholderReturn != screenCommand {
		t.Fatalf("placeholderReturn = %v, want command picker", m.placeholderReturn)
	}
	if !m.placeholderOK {
		t.Fatal("result should be marked successful")
	}
}

func TestInstallClearsDirectorySelection(t *testing.T) {
	dir := t.TempDir()
	m := newModel(options{})
	m.dir = dir
	m.detection = &Detection{Dir: dir, Last: StageManifests}
	m.dirInput.SetValue(dir)

	m.runCommand(commandInstall)

	if m.screen != screenDir {
		t.Fatalf("screen = %v, want directory prompt", m.screen)
	}
	if m.dir != "" {
		t.Fatalf("dir = %q, want empty", m.dir)
	}
	if m.dirInput.Value() != "" {
		t.Fatalf("dir input = %q, want empty", m.dirInput.Value())
	}
	if m.detection != nil {
		t.Fatal("expected detection to be cleared")
	}
}

func TestCommandInstallEntersStageFlow(t *testing.T) {
	dir := t.TempDir()
	m := newModel(options{})

	m.runCommand(commandInstall)
	if m.screen != screenDir {
		t.Fatalf("screen = %v, want directory prompt", m.screen)
	}

	m.dirInput.SetValue(dir)
	m.openDirectory()

	if m.screen != screenMenu {
		t.Fatalf("screen = %v, want stage menu", m.screen)
	}
}

func TestCommandNeedingDirectoryPrompts(t *testing.T) {
	m := newModel(options{})

	m.runCommand(commandGather)

	if m.screen != screenDir {
		t.Fatalf("screen = %v, want directory prompt", m.screen)
	}
	if m.pendingCommand != commandGather {
		t.Fatalf("pendingCommand = %v, want gather", m.pendingCommand)
	}
}

func TestCommandPlaceholderReturnsToCommandMenu(t *testing.T) {
	m := newModel(options{})

	m.runCommand(commandCompletion)
	if m.screen != screenPlaceholder {
		t.Fatalf("screen = %v, want placeholder", m.screen)
	}

	if _, _ = m.handlePlaceholderKey(tea.KeyMsg{Type: tea.KeyEscape}); m.screen != screenCommand {
		t.Fatalf("screen = %v, want command menu", m.screen)
	}
}

func TestMenuRunsMissingPrerequisites(t *testing.T) {
	m := newModel(options{})
	m.dir = t.TempDir()
	m.detection = &Detection{Last: StageNone}
	m.menuIndex = stageIndex(StageManifests)

	m.runSelectedStage()

	if m.screen != screenInstallConfig {
		t.Fatalf("screen = %v, want install-config form", m.screen)
	}
	if m.lastStage != StageInstallConfig {
		t.Fatalf("lastStage = %v, want install-config", m.lastStage)
	}
	if len(m.pendingStages) != 1 || m.pendingStages[0] != StageManifests {
		t.Fatalf("pendingStages = %v, want [manifests]", m.pendingStages)
	}
}

func TestRunDonePausesThenAdvancesStageQueue(t *testing.T) {
	m := newModel(options{})
	m.dir = t.TempDir()
	m.runID = 1
	m.pendingStages = []Stage{StageManifests, StageIgnitionConfigs}
	m.running = true

	_, cmd := m.handleRunDone(runDoneMsg{runID: 1, stage: StageInstallConfig})
	if cmd == nil {
		t.Fatal("expected a delayed advance command")
	}
	if m.screen != screenStageSuccess {
		t.Fatalf("screen = %v, want stage success", m.screen)
	}
	if m.advanceRunID != 1 {
		t.Fatalf("advanceRunID = %d, want 1", m.advanceRunID)
	}

	_, _ = m.Update(stageAdvanceMsg{runID: 1})
	if m.screen != screenRunning {
		t.Fatalf("screen = %v, want running", m.screen)
	}
	if m.lastStage != StageManifests {
		t.Fatalf("lastStage = %v, want manifests", m.lastStage)
	}
	if len(m.pendingStages) != 1 || m.pendingStages[0] != StageIgnitionConfigs {
		t.Fatalf("pendingStages = %v, want [ignition-configs]", m.pendingStages)
	}
}

func TestStageSuccessKeyBindings(t *testing.T) {
	m := newModel(options{})
	m.dir = t.TempDir()
	m.detection = &Detection{Last: StageInstallConfig}

	m.screen = screenStageSuccess
	if _, _ = m.handleStageSuccessKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}); m.screen != screenRunning {
		t.Fatalf("m: screen = %v, want running", m.screen)
	}
	if m.lastStage != StageManifests {
		t.Fatalf("m: lastStage = %v, want manifests", m.lastStage)
	}

	m.screen = screenStageSuccess
	m.running = false
	m.detection.Last = StageManifests
	if _, _ = m.handleStageSuccessKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}); m.screen != screenRunning {
		t.Fatalf("i: screen = %v, want running", m.screen)
	}
	if m.lastStage != StageIgnitionConfigs {
		t.Fatalf("i: lastStage = %v, want ignition", m.lastStage)
	}

	m.screen = screenStageSuccess
	m.running = false
	m.detection.Last = StageIgnitionConfigs
	if _, _ = m.handleStageSuccessKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}); m.screen != screenCluster {
		t.Fatalf("c: screen = %v, want cluster diagram", m.screen)
	}
}

func TestStageSuccessView(t *testing.T) {
	m := newModel(options{})
	m.screen = screenStageSuccess
	m.detection = &Detection{Last: StageInstallConfig}
	m.completionTitle = "Install Config"
	m.width = 120
	m.height = 40

	view := m.View()
	for _, want := range []string{"✓ success:", "Install Config", "m  Manifests", "i  Ignition Configs", "c  Install Cluster", "Full installer logs"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view is missing %q:\n%s", want, view)
		}
	}

	installConfig := strings.Index(view, "Install Config")
	manifests := strings.Index(view, "Manifests")
	ignition := strings.Index(view, "Ignition Configs")
	cluster := strings.Index(view, "Install Cluster")
	if !(installConfig < manifests && manifests < ignition && ignition < cluster) {
		t.Fatalf("stage order is wrong:\n%s", view)
	}
}

func TestOpenInstallerLogs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, logFileName, "first line\nsecond line\n")

	m := newModel(options{})
	m.dir = dir
	m.screen = screenStageSuccess

	if _, _ = m.handleStageSuccessKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}); m.screen != screenLogs {
		t.Fatalf("screen = %v, want logs", m.screen)
	}
	if !strings.Contains(m.fileLogViewport.View(), "second line") {
		t.Fatalf("log view does not contain the installer log: %s", m.fileLogViewport.View())
	}

	if _, _ = m.handleLogsKey(tea.KeyMsg{Type: tea.KeyEscape}); m.screen != screenStageSuccess {
		t.Fatalf("screen after esc = %v, want success screen", m.screen)
	}
}
