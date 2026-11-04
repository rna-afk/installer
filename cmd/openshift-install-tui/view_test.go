package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func TestViewFillsTerminalHeight(t *testing.T) {
	m := newModel(options{})
	m.width = 120
	m.height = 40

	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) != m.height {
		t.Fatalf("view has %d lines, want %d", len(lines), m.height)
	}
	found := false
	for _, line := range lines {
		if strings.Contains(line, "enter run") {
			found = true
		}
	}
	if !found {
		t.Fatal("footer is missing from the view")
	}
}

func TestViewCentersInstallerColumn(t *testing.T) {
	m := newModel(options{})
	m.width = 160
	m.height = 50

	lines := strings.Split(m.View(), "\n")
	if len(lines) != m.height {
		t.Fatalf("view has %d lines, want %d", len(lines), m.height)
	}

	titleAt, boxAt := -1, -1
	for i, line := range lines {
		if titleAt < 0 && strings.Contains(line, "What do you want to do?") {
			titleAt = i
		}
		if boxAt < 0 && strings.Contains(line, "┃") {
			boxAt = i
		}
	}
	if titleAt < 0 || boxAt < 0 {
		t.Fatal("did not find the title or the panel rule")
	}
	if titleAt == 0 {
		t.Fatal("content is not nudged down: title is on the first line")
	}
	if boxAt <= titleAt {
		t.Fatal("panel border should be rendered below the title")
	}

	// The column is capped and centered, so the rule has leading padding.
	if !strings.HasPrefix(ansi.Strip(lines[boxAt]), " ") {
		t.Fatalf("panel is not centered horizontally: %q", lines[boxAt])
	}
	if got := lipgloss.Width(lines[boxAt]); got != m.width {
		t.Fatalf("line width = %d, want %d", got, m.width)
	}
}

func TestCommandPromptOutsidePanel(t *testing.T) {
	m := newModel(options{})
	m.width = 120
	m.height = 40

	promptAt, boxAt := -1, -1
	for i, line := range strings.Split(m.View(), "\n") {
		if promptAt < 0 && strings.Contains(line, "What do you want to do?") {
			promptAt = i
		}
		if boxAt < 0 && strings.Contains(line, "┃") {
			boxAt = i
		}
	}
	if promptAt < 0 || boxAt < 0 {
		t.Fatal("did not find the prompt or the panel rule")
	}
	if promptAt >= boxAt {
		t.Fatalf("prompt (line %d) should be above the panel (line %d)", promptAt, boxAt)
	}
}

func panelOrigin(t *testing.T, view string) (int, int) {
	t.Helper()
	for i, line := range strings.Split(view, "\n") {
		if idx := strings.Index(ansi.Strip(line), "┃"); idx >= 0 {
			return i, idx
		}
	}
	t.Fatal("panel rule not found")
	return 0, 0
}

func TestCommandScreenHidesDirectory(t *testing.T) {
	m := newModel(options{})
	m.width = 120
	m.height = 40
	m.dir = "/tmp/assets"

	if strings.Contains(m.View(), "Directory:") {
		t.Fatal("command screen should not print the directory")
	}

	m.screen = screenMenu
	if !strings.Contains(m.View(), "Directory:") {
		t.Fatal("stage menu should print the directory")
	}
}

func TestPanelStaysAnchoredWhenContentChanges(t *testing.T) {
	m := newModel(options{})
	m.width = 120
	m.height = 40

	baseRow, baseCol := panelOrigin(t, m.View())
	for i := range commandChoices {
		m.commandIndex = i
		row, col := panelOrigin(t, m.View())
		if row != baseRow || col != baseCol {
			t.Fatalf("command %d moved the panel to (%d,%d), want (%d,%d)", i, row, col, baseRow, baseCol)
		}
	}

	m.statusMessage = "a status message must not move the panel"
	row, col := panelOrigin(t, m.View())
	if row != baseRow || col != baseCol {
		t.Fatalf("status message moved the panel to (%d,%d), want (%d,%d)", row, col, baseRow, baseCol)
	}

	m.screen = screenDir
	m.resize()
	m.statusMessage = ""
	m.dirChoices = nil
	m.errMessage = ""
	calmRow, calmCol := panelOrigin(t, m.View())

	m.dirChoices = []dirChoice{{Name: strings.Repeat("long-directory-name-", 8), Path: "x"}}
	m.errMessage = "an error line must not move the panel"
	busyRow, busyCol := panelOrigin(t, m.View())
	if busyRow != calmRow || busyCol != calmCol {
		t.Fatalf("busy directory screen moved the panel to (%d,%d), want (%d,%d)", busyRow, busyCol, calmRow, calmCol)
	}
}

func TestCommandSelectionAnimatesLift(t *testing.T) {
	m := newModel(options{})
	m.width = 120
	m.height = 40

	if m.selectionLift != 1 {
		t.Fatalf("initial lift = %v, want 1", m.selectionLift)
	}

	_, cmd := m.handleCommandKey(tea.KeyMsg{Type: tea.KeyDown})
	if cmd == nil {
		t.Fatal("expected an animation command")
	}
	if m.commandIndex != 1 {
		t.Fatalf("commandIndex = %d, want 1", m.commandIndex)
	}
	if m.selectionLift != 0 {
		t.Fatalf("lift = %v at animation start, want 0", m.selectionLift)
	}

	prev := m.selectionLift
	for i := 0; i < selectionAnimFrames+2; i++ {
		_, _ = m.Update(selectionAnimMsg{id: m.selectionAnimID})
		if m.selectionLift < prev {
			t.Fatalf("lift went backwards: %v -> %v", prev, m.selectionLift)
		}
		prev = m.selectionLift
	}
	if m.selectionLift != 1 {
		t.Fatalf("lift = %v after animation, want 1", m.selectionLift)
	}
}

func TestCommandPickerHidesUnselectedOptions(t *testing.T) {
	m := newModel(options{})
	m.width = 120
	m.height = 40

	view := m.View()
	if !strings.Contains(view, "Install") {
		t.Fatal("selected option is missing")
	}
	if strings.Contains(view, "Completion") {
		t.Fatal("unselected option should stay hidden until selected")
	}
}

func TestRenderWheelCentersSelection(t *testing.T) {
	m := newModel(options{})
	view := m.renderWheel([]string{"one", "two", "three", "four", "five", "six", "seven"}, 3)
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")

	if len(lines) != 7 {
		t.Fatalf("wheel has %d lines, want 7:\n%s", len(lines), view)
	}
	if !strings.Contains(lines[3], "four") {
		t.Fatalf("selected entry is not in the middle:\n%s", view)
	}
	if !strings.Contains(lines[2], "three") || !strings.Contains(lines[4], "five") {
		t.Fatalf("neighboring entries are not around the selection:\n%s", view)
	}
}

func TestRenderWheelShowsAllShortLists(t *testing.T) {
	m := newModel(options{})
	view := m.renderWheel([]string{"alpha", "beta"}, 0)
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")

	if len(lines) != 2 {
		t.Fatalf("wheel has %d lines, want 2:\n%s", len(lines), view)
	}
	if !strings.Contains(lines[0], "alpha") || !strings.Contains(lines[1], "beta") {
		t.Fatalf("wheel entries are out of order:\n%s", view)
	}
}

func TestPanelFillsTerminalWidth(t *testing.T) {
	m := newModel(options{})
	m.width = 120
	m.height = 40

	view := m.View()
	for _, line := range strings.Split(view, "\n") {
		if !strings.Contains(line, "┃") {
			continue
		}
		if got := lipgloss.Width(line); got != m.width {
			t.Fatalf("view width = %d, want %d", got, m.width)
		}
		return
	}
	t.Fatal("did not find the panel rule")
}

func TestDirectoryPromptTitleMatchesCommand(t *testing.T) {
	tests := []struct {
		command command
		want    string
	}{
		{commandNone, "Where should the installer run?"},
		{commandGather, "Where should Gather look for the assets?"},
		{commandWaitFor, "Where are the assets to wait on?"},
		{commandDestroy, "Where is the cluster to destroy?"},
	}
	for _, tt := range tests {
		m := newModel(options{})
		m.pendingCommand = tt.command
		m.screen = screenDir
		if got := m.dirPromptTitle(); got != tt.want {
			t.Fatalf("prompt title for %v = %q, want %q", tt.command, got, tt.want)
		}
		if !strings.Contains(m.View(), tt.want) {
			t.Fatalf("directory view is missing %q", tt.want)
		}
	}
}

func TestTitleUsesBannerWhenItFits(t *testing.T) {
	m := newModel(options{})
	m.width, m.height = 120, 40

	banner, ok := renderBanner(appBannerText)
	if !ok {
		t.Fatalf("banner text %q is not covered by the font", appBannerText)
	}
	view := m.View()
	for _, line := range strings.Split(banner, "\n") {
		if !strings.Contains(view, line) {
			t.Fatalf("view is missing banner line %q:\n%s", line, view)
		}
	}
	if got, want := m.titleLines(), bannerHeight; got != want {
		t.Fatalf("titleLines = %d, want %d", got, want)
	}
	// The command screen also reserves a blank line and the prompt.
	if got, want := m.headerLines(), bannerHeight+2; got != want {
		t.Fatalf("headerLines = %d, want %d on the command screen", got, want)
	}
}

func TestTitleFallsBackWhenBannerDoesNotFit(t *testing.T) {
	m := newModel(options{})
	m.width, m.height = bannerWidth(appBannerText), 40

	banner, _ := renderBanner(appBannerText)
	view := m.View()
	if strings.Contains(view, strings.Split(banner, "\n")[0]) {
		t.Fatalf("narrow terminal should not render the banner:\n%s", view)
	}
	if !strings.Contains(view, "OpenShift Installer") {
		t.Fatalf("view is missing the plain title:\n%s", view)
	}
	if got := m.titleLines(); got != 1 {
		t.Fatalf("titleLines = %d, want 1", got)
	}
}

func TestMainScreenHidesDirectoryContext(t *testing.T) {
	m := newModel(options{})
	m.width, m.height = 120, 40
	m.screen = screenCommand
	m.dir = "/tmp/assets"
	m.detection = &Detection{Last: StageCluster}

	view := m.View()
	if strings.Contains(view, "Directory:") {
		t.Fatalf("command picker should not print the directory:\n%s", view)
	}
	for _, badge := range []string{"[last stage:", "[new directory]", "[no completed stage]"} {
		if strings.Contains(view, badge) {
			t.Fatalf("command picker should not print %q:\n%s", badge, view)
		}
	}
	if got, want := m.headerLines(), m.titleLines()+2; got != want {
		t.Fatalf("headerLines = %d, want %d", got, want)
	}
	if got := len(strings.Split(view, "\n")); got != m.height {
		t.Fatalf("view has %d lines, want %d", got, m.height)
	}
}

func TestDirectoryPromptHidesDirectoryContext(t *testing.T) {
	m := newModel(options{})
	m.width, m.height = 120, 40
	m.screen = screenDir
	m.dir = "/tmp/assets"
	m.detection = &Detection{New: true}
	m.dirInput.SetValue("/tmp/assets")

	view := m.View()
	if strings.Contains(view, "Directory:") {
		t.Fatalf("directory prompt should not print a directory header:\n%s", view)
	}
	if strings.Contains(view, "[new directory]") {
		t.Fatalf("directory prompt should not print the stage badge:\n%s", view)
	}
}

func TestStageBadgeTrailsDirectoryOnStageScreens(t *testing.T) {
	m := newModel(options{})
	m.width, m.height = 120, 40
	m.screen = screenMenu
	m.dir = "/tmp/assets"
	m.detection = &Detection{Last: StageCluster}

	lines := strings.Split(m.View(), "\n")
	dirAt, badgeAt := -1, -1
	for i, line := range lines {
		if dirAt < 0 && strings.Contains(line, "Directory: /tmp/assets") {
			dirAt = i
		}
		if badgeAt < 0 && strings.Contains(line, "[last stage: cluster]") {
			badgeAt = i
		}
	}
	if dirAt < 0 || badgeAt < 0 {
		t.Fatalf("directory or badge is missing:\n%s", m.View())
	}
	if dirAt != badgeAt {
		t.Fatalf("badge is on line %d, want it inline with the directory on line %d", badgeAt, dirAt)
	}
}

// The install cluster screens fill the terminal so the diagram and its log box
// get as many rows as possible; every other screen keeps the compact panel.
func TestClusterScreensFillTerminal(t *testing.T) {
	sizes := []struct {
		width, height int
	}{
		{80, 24},
		{120, 40},
		{200, 60},
	}
	for _, size := range sizes {
		for _, scr := range []screen{screenCluster, screenClusterLogs} {
			m := newModel(options{})
			m.width, m.height = size.width, size.height
			m.dir = "/tmp/assets"
			m.detection = &Detection{Last: StageCluster}
			m.screen = scr
			m.runLabel = "Installing cluster"
			m.clusterDiagram = newClusterDiagram(1)
			m.clusterDiagram.observe("Creating infrastructure resources...")
			for i := 0; i < 40; i++ {
				m.appendLog("installer log line")
			}
			m.flushLogs()
			m.resize()

			// The panel plus its left rule spans the terminal.
			if got, want := m.panelWidth()+1, m.width; got != want {
				t.Fatalf("screen %v panel width = %d at %dx%d, want %d", scr, got, size.width, size.height, want)
			}
			lines := strings.Split(m.View(), "\n")
			if len(lines) != m.height {
				t.Fatalf("screen %v rendered %d lines at %dx%d, want %d", scr, len(lines), size.width, size.height, m.height)
			}
			// The title is pinned one row below the top edge.
			if got := ansi.Strip(lines[topMargin]); !strings.Contains(got, "█") {
				t.Fatalf("screen %v title is not on line %d at %dx%d: %q", scr, topMargin, size.width, size.height, got)
			}
			// The log viewport gets the rows the diagram leaves free.
			if got, want := m.clusterLogViewport.Height, 4; got < want {
				t.Fatalf("screen %v log box height = %d, want at least %d", scr, got, want)
			}
		}
	}
}

func TestOtherScreensKeepCompactPanel(t *testing.T) {
	for _, scr := range []screen{screenCommand, screenDir, screenMenu, screenClusterCheck, screenExplain, screenInstallConfig} {
		m := newModel(options{})
		m.width, m.height = 200, 60
		m.dir = "/tmp/assets"
		m.detection = &Detection{Last: StageCluster}
		m.screen = scr
		m.resize()

		if got := m.panelWidth(); got > maxBoxWidth-1 {
			t.Fatalf("screen %v panel width = %d, want at most %d", scr, got, maxBoxWidth-1)
		}
		if got := m.panelHeight(); got > maxBoxHeight-2 {
			t.Fatalf("screen %v panel height = %d, want at most %d", scr, got, maxBoxHeight-2)
		}
		if got := len(strings.Split(m.View(), "\n")); got != m.height {
			t.Fatalf("screen %v rendered %d lines, want %d", scr, got, m.height)
		}
	}
}

func TestRunningViewFillsTerminalHeight(t *testing.T) {
	m := newModel(options{})
	m.width = 120
	m.height = 40
	m.screen = screenRunning
	m.runLabel = "Running Manifests"
	m.resize()

	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) != m.height {
		t.Fatalf("view has %d lines, want %d", len(lines), m.height)
	}
	found := false
	for _, line := range lines {
		if strings.Contains(line, "cancel") {
			found = true
		}
	}
	if !found {
		t.Fatal("footer is missing from the view")
	}
}

// Starting a run resizes the log view for the running screen. A previous resize
// on a compact screen (the command picker is only 8 rows) must not shrink it.
func TestRunningLogIsNotShrunkByCompactScreens(t *testing.T) {
	for _, start := range []screen{screenCommand, screenDir, screenMenu, screenInstallConfig} {
		m := newModel(options{})
		m.width, m.height = 120, 40
		m.dir = "/tmp/assets"
		m.screen = start
		m.resize()

		m.prepareRun("Running Manifests")

		want := m.logPanelHeight() - 4
		if got := m.logViewport.Height; got != want {
			t.Fatalf("starting from %v: log viewport height = %d, want %d", start, got, want)
		}
		if m.logViewport.Height < 15 {
			t.Fatalf("starting from %v: log viewport height = %d, want a large log view", start, m.logViewport.Height)
		}

		m.statusMessage = ""
		lines := strings.Split(m.View(), "\n")
		if len(lines) != m.height {
			t.Fatalf("starting from %v: view has %d lines, want %d", start, len(lines), m.height)
		}
	}
}

func TestFullLogViewUsesLogPanelFromCompactScreen(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, logFileName, strings.Repeat("log line\n", 100))

	m := newModel(options{})
	m.width, m.height = 120, 40
	m.dir = dir
	m.screen = screenMenu
	m.resize()

	m.openLogs()

	want := m.logPanelHeight() - 4
	if got := m.fileLogViewport.Height; got != want {
		t.Fatalf("full log viewport height = %d, want %d", got, want)
	}
	if got := len(strings.Split(m.View(), "\n")); got != m.height {
		t.Fatalf("view has %d lines, want %d", got, m.height)
	}
}

// The stage result box and the full log box share the same geometry, so opening
// the log with l does not move or resize the frame.
func TestStageSuccessAndLogsBoxesMatch(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, logFileName, strings.Repeat("log line\n", 100))

	m := newModel(options{})
	m.width, m.height = 120, 40
	m.dir = dir
	m.detection = &Detection{Last: StageManifests}
	m.screen = screenStageSuccess
	m.completionTitle = "Manifests"
	m.resize()

	successRow, successCol := panelOrigin(t, m.View())
	successWidth, successHeight := m.panelWidth(), m.panelHeight()

	m.openLogs()
	logsRow, logsCol := panelOrigin(t, m.View())

	if logsRow != successRow || logsCol != successCol {
		t.Fatalf("logs box at (%d,%d), want (%d,%d) like the success box", logsRow, logsCol, successRow, successCol)
	}
	if got, want := m.panelWidth(), successWidth; got != want {
		t.Fatalf("logs box width = %d, want %d", got, want)
	}
	if got, want := m.panelHeight(), successHeight; got != want {
		t.Fatalf("logs box height = %d, want %d", got, want)
	}
	if got := len(strings.Split(m.View(), "\n")); got != m.height {
		t.Fatalf("logs view has %d lines, want %d", got, m.height)
	}
	if !strings.Contains(m.View(), "Installer logs") {
		t.Fatal("logs view is missing its title")
	}
}

// Every log view renders inside the same red-ruled panel as the other screens.
func TestLogViewsShareThePanelBorder(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, logFileName, strings.Repeat("log line\n", 50))

	for _, scr := range []screen{screenRunning, screenLogs, screenStageSuccess, screenClusterLogs} {
		m := newModel(options{})
		m.width, m.height = 120, 40
		m.dir = dir
		m.detection = &Detection{Last: StageCluster}
		m.screen = scr
		m.runLabel = "Running Manifests"
		m.clusterDiagram = newClusterDiagram(1)
		m.clusterDiagram.observe("Creating infrastructure resources...")
		m.completionTitle = "Manifests"
		m.resize()

		lines := strings.Split(m.View(), "\n")
		ruleAt := -1
		for i, line := range lines {
			if strings.Contains(ansi.Strip(line), "┃") {
				ruleAt = i
				break
			}
		}
		if ruleAt < 0 {
			t.Fatalf("screen %v has no panel rule", scr)
		}
		if got, want := len(lines), m.height; got != want {
			t.Fatalf("screen %v rendered %d lines, want %d", scr, got, want)
		}
	}
}

// Panels never push the footer off small terminals.
func TestPanelsFitSmallTerminals(t *testing.T) {
	for _, size := range []struct{ width, height int }{{80, 24}, {100, 30}} {
		for _, scr := range []screen{screenCommand, screenMenu, screenRunning, screenStageSuccess, screenLogs} {
			m := newModel(options{})
			m.width, m.height = size.width, size.height
			m.dir = "/tmp/assets"
			m.detection = &Detection{Last: StageCluster}
			m.screen = scr
			m.runLabel = "Running Manifests"
			m.completionTitle = "Manifests"
			m.appendLog("log line")
			m.flushLogs()
			m.resize()

			lines := strings.Split(m.View(), "\n")
			if len(lines) != m.height {
				t.Fatalf("screen %v at %dx%d rendered %d lines, want %d", scr, size.width, size.height, len(lines), m.height)
			}
			if view := ansi.Strip(strings.Join(lines, "\n")); !strings.Contains(view, "•") {
				t.Fatalf("screen %v at %dx%d lost its footer", scr, size.width, size.height)
			}
		}
	}
}

// The title padding must be painted with the app background; otherwise the
// terminal's own background shows through around the wordmark.
func TestHeaderPaddingUsesAppBackground(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	for _, size := range []struct{ width, height int }{{80, 24}, {120, 40}, {200, 60}} {
		m := newModel(options{})
		m.width, m.height = size.width, size.height
		m.dir = "/tmp/assets"
		m.detection = &Detection{Last: StageManifests}
		m.resize()

		for i, line := range strings.Split(m.View(), "\n") {
			if strings.Contains(line, "\x1b[0m ") || strings.HasPrefix(line, " ") {
				t.Fatalf("%dx%d line %d has unpainted padding: %q", size.width, size.height, i, line)
			}
		}
	}
}
