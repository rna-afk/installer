package main

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestClusterDiagramObserve(t *testing.T) {
	d := newClusterDiagram(1)

	d.observe("Creating infrastructure resources...")
	if d.components[0].State != clusterRunning {
		t.Fatalf("network state = %v, want running", d.components[0].State)
	}

	d.observe("Creating load balancer")
	if d.components[0].State != clusterDone || d.components[1].State != clusterDone || d.components[2].State != clusterRunning {
		t.Fatalf("unexpected states: %+v", d.components[:3])
	}

	d.observe("Waiting for the Kubernetes API")
	if d.components[4].State != clusterRunning {
		t.Fatalf("temporary control plane state = %v, want running", d.components[4].State)
	}

	d.observe("Install complete!")
	for i, component := range d.components {
		if component.State != clusterDone {
			t.Fatalf("component %d state = %v, want done", i, component.State)
		}
	}
}

func TestClusterDiagramSelection(t *testing.T) {
	d := newClusterDiagram(1)
	d.observe("Creating infrastructure resources...")
	d.observe("Creating load balancer")

	d.moveSelection(1)
	if d.selected != 1 {
		t.Fatalf("selected = %d, want 1", d.selected)
	}
	d.moveSelection(100)
	if d.selected != 2 {
		t.Fatalf("selected = %d, want last visible component", d.selected)
	}
}

func TestClusterLogsScreenToggle(t *testing.T) {
	m := newModel(options{})
	m.clusterDiagram = newClusterDiagram(1)
	m.screen = screenCluster

	m.handleClusterKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if m.screen != screenClusterLogs {
		t.Fatalf("screen = %v, want logs", m.screen)
	}
	m.handleClusterLogsKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if m.screen != screenCluster {
		t.Fatalf("screen = %v, want diagram", m.screen)
	}
}

func TestClusterView(t *testing.T) {
	m := newModel(options{})
	m.width, m.height = 120, 40
	m.resize()
	m.screen = screenCluster
	m.clusterDiagram = newClusterDiagram(1)
	m.clusterDiagram.observe("Creating infrastructure resources...")
	m.appendLog("creating network")
	m.flushLogs()

	view := m.View()
	for _, want := range []string{"Install Cluster", "Network", "expand logs"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view is missing %q:\n%s", want, view)
		}
	}
	m.screen = screenClusterLogs
	logs := m.View()
	for _, want := range []string{"Cluster install logs", "creating network"} {
		if !strings.Contains(logs, want) {
			t.Fatalf("logs view is missing %q:\n%s", want, logs)
		}
	}
}

func TestClusterLogsDoNotTruncateLongErrors(t *testing.T) {
	m := newModel(options{})
	m.width, m.height = 80, 40
	m.resize()
	m.screen = screenClusterLogs
	long := "level=error msg=\"failed to do something: " + strings.Repeat("a", 150) + " end-of-error\""
	m.appendLog(long)
	m.flushLogs()
	m.clusterLogViewport.SetXOffset(1000)

	view := m.View()
	if !strings.Contains(view, "end-of-error") {
		t.Fatalf("long log line was not horizontally scrollable:\n%s", view)
	}
}

func TestFullLogViewDoesNotTruncateLongLines(t *testing.T) {
	dir := t.TempDir()
	line := strings.Repeat("b", 150) + " end-of-log"
	writeFile(t, dir, logFileName, line+"\n")

	m := newModel(options{})
	m.dir = dir
	m.width, m.height = 80, 40
	m.resize()
	m.openLogs()
	m.fileLogViewport.SetXOffset(1000)

	view := m.View()
	if !strings.Contains(view, "end-of-log") {
		t.Fatalf("full log line was not horizontally scrollable:\n%s", view)
	}
}

func TestFilterLogLines(t *testing.T) {
	lines := []string{
		"level=debug msg=debug",
		"level=info msg=info",
		"level=warning msg=warn",
		"level=error msg=error",
		"plain line",
	}

	tests := []struct {
		filter logFilter
		want   int
	}{
		{logFilterAll, 5},
		{logFilterDebug, 5},
		{logFilterInfo, 4},
		{logFilterWarn, 3},
		{logFilterError, 2},
	}
	for _, tt := range tests {
		if got := len(filterLogLines(lines, tt.filter)); got != tt.want {
			t.Fatalf("filter %s returned %d lines, want %d", tt.filter, got, tt.want)
		}
	}
}

func TestCycleLogFilter(t *testing.T) {
	m := newModel(options{})
	m.appendLog("level=debug msg=debug")
	m.appendLog("level=error msg=error")
	m.flushLogs()

	if m.logFilter != logFilterAll {
		t.Fatal("expected default filter to be all")
	}
	m.cycleLogFilter()
	if m.logFilter != logFilterDebug {
		t.Fatalf("filter = %s, want debug", m.logFilter)
	}
	m.cycleLogFilter()
	m.cycleLogFilter()
	m.cycleLogFilter()
	if m.logFilter != logFilterError {
		t.Fatalf("filter = %s, want error", m.logFilter)
	}
	if strings.Contains(m.clusterLogViewport.View(), "msg=debug") {
		t.Fatal("debug log should be filtered out at error level")
	}
	if !strings.Contains(m.clusterLogViewport.View(), "msg=error") {
		t.Fatal("error log should remain visible")
	}
}

func TestDestroyPromptsToDeleteDirectory(t *testing.T) {
	dir := t.TempDir()
	m := newModel(options{})
	m.dir = dir

	m.handleCommandDone(commandDoneMsg{
		title:    "Destroy cluster",
		detail:   "Cluster destroyed.",
		returnTo: screenCommand,
	})

	if m.screen != screenDeleteDir {
		t.Fatalf("screen = %v, want delete directory prompt", m.screen)
	}
}

func TestDeleteDirectoryOnYes(t *testing.T) {
	dir := t.TempDir()
	m := newModel(options{})
	m.dir = dir
	m.screen = screenDeleteDir
	m.completionTitle = "Destroy cluster completed"

	m.handleDeleteDirKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be removed, stat err = %v", dir, err)
	}
	if m.dir != "" {
		t.Fatalf("dir = %q, want empty", m.dir)
	}
}

func TestDeleteDirectoryOnNo(t *testing.T) {
	dir := t.TempDir()
	m := newModel(options{})
	m.dir = dir
	m.screen = screenDeleteDir

	m.handleDeleteDirKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("expected %s to remain: %v", dir, err)
	}
	if m.screen != screenCommand {
		t.Fatalf("screen = %v, want command picker", m.screen)
	}
}

func TestClusterTargetsAvailable(t *testing.T) {
	targets, ok := targetsForStage(StageCluster)
	if !ok || len(targets) == 0 {
		t.Fatal("expected cluster targets")
	}
}
