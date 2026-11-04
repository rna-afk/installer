package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sirupsen/logrus"
)

type screen int

const (
	screenDir screen = iota
	screenCommand
	screenWaitFor
	screenExplain
	screenExplainDetail
	screenMenu
	screenResume
	screenClusterCheck
	screenRunning
	screenCluster
	screenClusterLogs
	screenDeleteDir
	screenComplete
	screenFailed
	screenPlaceholder
	screenInstallConfig
	screenStageSuccess
	screenLogs
)

// Model is the Bubble Tea application model.
type Model struct {
	program *tea.Program
	opts    options

	width  int
	height int
	screen screen

	dir            string
	detection      *Detection
	form           *installConfigForm
	explain        *explainState
	clusterDiagram *clusterDiagram

	commandIndex int
	waitForIndex int

	// selectionLift drives the depth animation of the highlighted option:
	// 0 is flat against the panel, 1 is fully lifted.
	selectionLift   float64
	selectionFrames int
	selectionAnimID int
	pendingCommand  command
	pendingStages   []Stage

	uiLog   *uiLogHook
	fileLog *fileLogHook

	dirInput       textinput.Model
	dirChoices     []dirChoice
	dirChoiceIndex int
	dirListBase    string
	menuIndex      int

	spinner            spinner.Model
	inputSpinner       spinner.Model
	logViewport        viewport.Model
	clusterLogViewport viewport.Model
	fileLogViewport    viewport.Model
	logLines           []string
	fileLogLines       []string
	logDirty           bool
	logFilter          logFilter
	logsReturn         screen

	running      bool
	runID        int
	advanceRunID int
	runLabel     string
	lastStage    Stage

	cancelRun context.CancelFunc
	mu        sync.Mutex

	statusMessage     string
	errMessage        string
	details           []string
	placeholderReturn screen
	placeholderOK     bool

	completionTitle string
	completionBody  []string
}

func newModel(opts options) *Model {
	if opts.logLevel == "" {
		opts.logLevel = "info"
	}

	dirInput := textinput.New()
	dirInput.Placeholder = ". (current directory)"
	dirInput.Prompt = "> "
	dirInput.CharLimit = 1024
	dirInput.Width = 60
	dirInput.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = spinnerStyle

	inputSp := spinner.New()
	inputSp.Spinner = spinner.MiniDot
	inputSp.Style = spinnerStyle

	vp := viewport.New(80, 20)
	vp.SetContent("")
	vp.SetHorizontalStep(4)
	clusterVP := viewport.New(80, 20)
	clusterVP.SetContent("")
	clusterVP.SetHorizontalStep(4)
	fileVP := viewport.New(80, 20)
	fileVP.SetContent("")
	fileVP.SetHorizontalStep(4)

	initialScreen := screenCommand
	if opts.stage != StageNone {
		initialScreen = screenDir
	}

	m := &Model{
		opts:               opts,
		screen:             initialScreen,
		dirInput:           dirInput,
		spinner:            sp,
		inputSpinner:       inputSp,
		logViewport:        vp,
		clusterLogViewport: clusterVP,
		fileLogViewport:    fileVP,
		placeholderReturn:  screenMenu,
		selectionLift:      1,
	}

	if opts.dirSet {
		m.dirInput.SetValue(opts.dir)
	}

	m.refreshDirChoices()

	return m
}

func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink, m.spinner.Tick, m.inputSpinner.Tick}
	switch {
	case m.opts.dirSet:
		// --dir was explicit, so behave like the installer and open it
		// immediately instead of prompting.
		cmds = append(cmds, m.openDirectory())
	case m.screen == screenDir:
		cmds = append(cmds, m.dirInput.Focus())
	}
	return tea.Batch(cmds...)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resize()
		return m, nil

	case selectionAnimMsg:
		return m, m.stepSelectionAnim(msg)

	case tea.KeyMsg:
		// Global keys.
		switch msg.String() {
		case "ctrl+c":
			if m.running {
				m.cancelRunning()
				m.statusMessage = "cancelling..."
				return m, nil
			}
			return m, tea.Quit
		}

		switch m.screen {
		case screenDir:
			switch msg.String() {
			case "enter":
				return m, m.openDirectory()
			case "esc":
				if m.opts.stage == StageNone {
					m.pendingCommand = commandNone
					m.screen = screenCommand
				}
				return m, nil
			case "up":
				m.moveDirChoice(-1)
				return m, m.startSelectionAnim()
			case "down":
				m.moveDirChoice(1)
				return m, m.startSelectionAnim()
			}
		case screenCommand:
			return m.handleCommandKey(msg)
		case screenWaitFor:
			return m.handleWaitForKey(msg)
		case screenExplain:
			return m.handleExplainKey(msg)
		case screenExplainDetail:
			return m.handleExplainDetailKey(msg)
		case screenMenu:
			return m.handleMenuKey(msg)
		case screenResume:
			return m.handleResumeKey(msg)
		case screenClusterCheck:
			return m.handleClusterCheckKey(msg)
		case screenRunning:
			return m.handleRunningKey(msg)
		case screenCluster:
			return m.handleClusterKey(msg)
		case screenClusterLogs:
			return m.handleClusterLogsKey(msg)
		case screenDeleteDir:
			return m.handleDeleteDirKey(msg)
		case screenComplete:
			return m.handleCompleteKey(msg)
		case screenFailed:
			return m.handleFailedKey(msg)
		case screenPlaceholder:
			return m.handlePlaceholderKey(msg)
		case screenInstallConfig:
			return m.handleInstallFormKey(msg)
		case screenStageSuccess:
			return m.handleStageSuccessKey(msg)
		case screenLogs:
			return m.handleLogsKey(msg)
		}

	case logLineMsg:
		if msg.runID != m.runID {
			return m, nil
		}
		m.appendLog(msg.line)
		if m.clusterDiagram != nil && m.clusterDiagram.runID == msg.runID {
			m.clusterDiagram.observe(msg.line)
		}
		return m, nil

	case flushLogsMsg:
		m.flushLogs()
		if m.running {
			return m, tickLogs()
		}
		return m, nil

	case runDoneMsg:
		if msg.runID != m.runID {
			return m, nil
		}
		return m.handleRunDone(msg)

	case commandDoneMsg:
		if msg.runID != m.runID {
			return m, nil
		}
		return m.handleCommandDone(msg)

	case stageAdvanceMsg:
		if m.advanceRunID != msg.runID || m.screen != screenStageSuccess {
			return m, nil
		}
		m.advanceRunID = 0
		return m, m.advanceStageQueue()

	case baseDomainsMsg:
		return m.handleBaseDomains(msg)

	case azureConnectedMsg:
		return m.handleAzureConnected(msg)

	case spinner.TickMsg:
		var spinCmd, inputCmd tea.Cmd
		m.spinner, spinCmd = m.spinner.Update(msg)
		m.inputSpinner, inputCmd = m.inputSpinner.Update(msg)
		return m, tea.Batch(spinCmd, inputCmd)
	}

	return m, m.updateFocused(msg)
}

func (m *Model) updateFocused(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch m.screen {
	case screenDir:
		m.dirInput, cmd = m.dirInput.Update(msg)
		if _, ok := msg.(tea.KeyMsg); ok {
			m.errMessage = ""
			m.refreshDirChoices()
		}
	case screenRunning:
		m.logViewport, cmd = m.logViewport.Update(msg)
	case screenInstallConfig:
		if m.form != nil {
			cmd = m.form.updateFocused(msg)
		}
	case screenExplain:
		if m.explain != nil {
			switch {
			case m.explain.globalSearching:
				m.explain.globalSearch, cmd = m.explain.globalSearch.Update(msg)
				m.explain.applyGlobalSearch(m.explain.globalSearch.Value())
			case m.explain.searching:
				m.explain.search, cmd = m.explain.search.Update(msg)
				m.explain.applySearch(m.explain.search.Value())
			}
		}
	}

	return cmd
}

func (m *Model) resize() {
	pw := m.panelWidth()
	ph := m.panelHeight()

	// The log panes size their viewports from the log panel budget so a resize
	// that happened on a compact screen cannot shrink them.
	logWidth := m.logPanelWidth()
	logHeight := m.logPanelHeight()

	// The running screen shows a heading line and a blank line above the log.
	m.logViewport.Width = clamp(logWidth-4, 20, 200)
	m.logViewport.Height = clamp(logHeight-4, 5, 200)

	// The full log screen is a panel with a heading line and a blank line.
	m.fileLogViewport.Width = clamp(logWidth-4, 20, 200)
	m.fileLogViewport.Height = clamp(logHeight-4, 5, 200)

	m.clusterLogViewport.Width = clamp(pw-4, 20, 200)
	switch {
	case m.screen == screenClusterLogs && m.fullBleed():
		// The expanded log screen fills the panel below its heading line.
		m.clusterLogViewport.Height = clamp(ph-4, 5, 200)
	case m.screen == screenClusterLogs:
		m.clusterLogViewport.Height = clamp(ph-8, 5, 200)
	default:
		// The inline log box is resized to the leftover rows in clusterView.
		m.clusterLogViewport.Height = clamp(ph-20, 4, 200)
	}
	if m.explain != nil {
		m.explain.search.Width = clamp(pw-16, 20, 80)
		m.explain.globalSearch.Width = clamp(pw-16, 20, 100)
		if m.width > 0 && m.height > 0 {
			m.explain.detail.Width = clamp(pw-6, 20, 200)
			m.explain.detail.Height = clamp(ph-6, 3, 200)
		}
	}

	m.dirInput.Width = clamp(pw-12, 20, 120)
	if m.form != nil {
		m.form.setWidth(pw - 26)
	}
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func (m *Model) openDirectory() tea.Cmd {
	raw := strings.TrimSpace(m.dirInput.Value())
	if raw == "" {
		raw = m.defaultDir()
	}

	dir, err := expandPath(raw)
	if err != nil {
		m.errMessage = err.Error()
		m.dirInput.Focus()
		return nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		m.errMessage = fmt.Sprintf("cannot create %s: %v", dir, err)
		m.dirInput.Focus()
		return nil
	}

	detection, err := Detect(dir)
	if err != nil {
		m.errMessage = fmt.Sprintf("cannot inspect %s: %v", dir, err)
		m.dirInput.Focus()
		return nil
	}

	if m.fileLog != nil {
		if err := m.openLogFile(dir); err != nil {
			m.errMessage = fmt.Sprintf("cannot open installer log in %s: %v", dir, err)
			m.dirInput.Focus()
			return nil
		}
	}

	m.dir = dir
	m.detection = detection
	m.errMessage = ""
	m.dirInput.Blur()

	if m.opts.stage != StageNone {
		return m.runStage(m.opts.stage)
	}

	if m.pendingCommand != commandNone {
		c := m.pendingCommand
		m.pendingCommand = commandNone
		if c == commandWaitFor {
			return m.executeWaitFor()
		}
		return m.executeCommand(c)
	}

	m.screen = screenCommand
	return nil
}

// clearDirectorySelection resets the directory prompt to an empty input.
func (m *Model) clearDirectorySelection() {
	m.dir = ""
	m.detection = nil
	m.errMessage = ""
	m.dirInput.SetValue("")
	m.refreshDirChoices()
}

// prefillDirectory prepares the directory prompt with the directory already in
// use, so a command that always asks for one can reuse the previous selection
// with a single keypress while still allowing a different path to be typed.
func (m *Model) prefillDirectory() {
	m.errMessage = ""
	if m.dir != "" {
		m.dirInput.SetValue(m.dir)
		m.dirInput.CursorEnd()
	}
	m.refreshDirChoices()
	m.dirInput.Focus()
}

// defaultDir returns the directory to open when the prompt is submitted
// without input: the first directory in the list, falling back to the current
// directory when no subdirectories are listed.
func (m *Model) defaultDir() string {
	if len(m.dirChoices) > 0 {
		return m.dirChoices[0].Path
	}
	return "."
}

type dirChoice struct {
	Name string
	Path string
}

// refreshDirChoices lists subdirectories of the directory implied by the
// current input. If the input is a prefix rather than a directory, the parent
// is listed and the entries are filtered by that prefix.
func (m *Model) refreshDirChoices() {
	m.dirChoices = nil
	m.dirChoiceIndex = 0
	m.dirInput.Placeholder = ". (current directory)"

	base, prefix := dirListBase(m.dirInput.Value())
	m.dirListBase = base

	entries, err := os.ReadDir(base)
	if err != nil {
		return
	}

	const maxChoices = 200
	choices := make([]dirChoice, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if prefix != "" && !strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			continue
		}
		choices = append(choices, dirChoice{
			Name: name,
			Path: filepath.Join(base, name),
		})
	}

	sort.Slice(choices, func(i, j int) bool {
		return strings.ToLower(choices[i].Name) < strings.ToLower(choices[j].Name)
	})
	if len(choices) > maxChoices {
		choices = choices[:maxChoices]
	}
	m.dirChoices = choices
	if len(choices) > 0 {
		m.dirInput.Placeholder = choices[0].Name + " (first directory in the list)"
	}
}

func (m *Model) moveDirChoice(delta int) {
	if len(m.dirChoices) == 0 {
		return
	}
	index := m.dirChoiceIndex + delta
	if index < 0 {
		index = 0
	}
	if index >= len(m.dirChoices) {
		index = len(m.dirChoices) - 1
	}
	m.dirChoiceIndex = index
	m.dirInput.SetValue(m.dirChoices[index].Path)
	m.errMessage = ""
}

// dirListBase returns the directory to list and an optional name prefix.
func dirListBase(raw string) (string, string) {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return cwd, ""
	}

	trailingSeparator := strings.HasSuffix(raw, "/") || strings.HasSuffix(raw, string(os.PathSeparator))
	expanded, err := expandPath(raw)
	if err != nil {
		return cwd, ""
	}

	if info, err := os.Stat(expanded); err == nil && info.IsDir() {
		return expanded, ""
	}
	if trailingSeparator {
		return expanded, ""
	}
	return filepath.Dir(expanded), filepath.Base(expanded)
}

func expandPath(raw string) (string, error) {
	if raw == "~" || strings.HasPrefix(raw, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		raw = filepath.Join(home, strings.TrimPrefix(raw, "~"))
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

const (
	selectionAnimStep   = 18 * time.Millisecond
	selectionAnimFrames = 10
)

// selectionAnimMsg advances the depth animation of the highlighted option.
type selectionAnimMsg struct{ id int }

// startSelectionAnim restarts the lift animation from the flat state.
func (m *Model) startSelectionAnim() tea.Cmd {
	m.selectionAnimID++
	m.selectionFrames = selectionAnimFrames
	m.selectionLift = 0
	return selectionTickCmd(m.selectionAnimID)
}

func selectionTickCmd(id int) tea.Cmd {
	return tea.Tick(selectionAnimStep, func(time.Time) tea.Msg {
		return selectionAnimMsg{id: id}
	})
}

// stepSelectionAnim advances one frame and schedules the next one.
func (m *Model) stepSelectionAnim(msg selectionAnimMsg) tea.Cmd {
	if msg.id != m.selectionAnimID {
		return nil
	}
	if m.selectionFrames > 0 {
		m.selectionFrames--
	}
	progress := 1 - float64(m.selectionFrames)/float64(selectionAnimFrames)
	m.selectionLift = easeOutCubic(progress)
	if m.selectionFrames == 0 {
		m.selectionLift = 1
		return nil
	}
	return selectionTickCmd(msg.id)
}

func easeOutCubic(t float64) float64 {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return 1 - math.Pow(1-t, 3)
}

func (m *Model) handleCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.commandIndex > 0 {
			m.commandIndex--
			return m, m.startSelectionAnim()
		}
	case "down", "j":
		if m.commandIndex < len(commandChoices)-1 {
			m.commandIndex++
			return m, m.startSelectionAnim()
		}
	case "enter":
		return m, m.runCommand(commandChoices[m.commandIndex])
	default:
		if len(msg.String()) == 1 {
			if digit := msg.String()[0]; digit >= '1' && digit <= '9' {
				index := int(digit - '1')
				if index < len(commandChoices) {
					m.commandIndex = index
					return m, m.runCommand(commandChoices[index])
				}
			}
		}
	}
	return m, nil
}

func (m *Model) handleWaitForKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		m.screen = screenCommand
	case "up", "k":
		if m.waitForIndex > 0 {
			m.waitForIndex--
			return m, m.startSelectionAnim()
		}
	case "down", "j":
		if m.waitForIndex < len(waitForStages)-1 {
			m.waitForIndex++
			return m, m.startSelectionAnim()
		}
	case "enter":
		return m, m.startWaitFor()
	default:
		if len(msg.String()) == 1 {
			if digit := msg.String()[0]; digit >= '1' && digit <= '9' {
				index := int(digit - '1')
				if index < len(waitForStages) {
					m.waitForIndex = index
					return m, m.startWaitFor()
				}
			}
		}
	}
	return m, nil
}

func (m *Model) handleMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		m.opts.stage = StageNone
		m.screen = screenCommand
	case "up", "k":
		if m.menuIndex > 0 {
			m.menuIndex--
			return m, m.startSelectionAnim()
		}
	case "down", "j":
		if m.menuIndex < len(stages)-1 {
			m.menuIndex++
			return m, m.startSelectionAnim()
		}
	case "1", "2", "3", "4":
		m.menuIndex = int(msg.String()[0] - '1')
		return m, m.runSelectedStage()
	case "enter":
		return m, m.runSelectedStage()
	}
	return m, nil
}

func (m *Model) runSelectedStage() tea.Cmd {
	return m.runStage(stages[m.menuIndex])
}

func (m *Model) handleResumeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "n", "m":
		m.screen = screenMenu
		m.menuIndex = 0
		return m, nil
	case "y", "enter":
		next := m.detection.Next()
		if next == StageNone {
			m.screen = screenClusterCheck
			return m, nil
		}
		return m, m.runStage(next)
	}
	return m, nil
}

func (m *Model) handleClusterCheckKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "g":
		return m, m.startGather(screenClusterCheck)
	case "a":
		return m, m.startAnalyze(screenClusterCheck)
	case "l":
		m.openLogs()
		return m, nil
	case "c", "y", "enter":
		return m, m.runStage(StageCluster)
	case "m", "n":
		m.screen = screenMenu
		m.menuIndex = 0
	}
	return m, nil
}

func (m *Model) handleRunningKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		if !m.canCancel() {
			m.statusMessage = "this operation cannot be cancelled"
			return m, nil
		}
		m.cancelRunning()
		m.statusMessage = "cancelling..."
		return m, nil
	case "f":
		m.cycleLogFilter()
		return m, nil
	case "left", "h":
		m.logViewport.ScrollLeft(m.logViewport.Width / 2)
		return m, nil
	case "right":
		m.logViewport.ScrollRight(m.logViewport.Width / 2)
		return m, nil
	}
	var cmd tea.Cmd
	m.logViewport, cmd = m.logViewport.Update(msg)
	return m, cmd
}

func (m *Model) handleClusterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.cancelRunning()
		m.statusMessage = "cancelling..."
	case "l":
		m.screen = screenClusterLogs
		m.resize()
		m.clusterLogViewport.GotoBottom()
	case "up", "k":
		if m.clusterDiagram != nil {
			m.clusterDiagram.moveSelection(-1)
		}
	case "down", "j":
		if m.clusterDiagram != nil {
			m.clusterDiagram.moveSelection(1)
		}
	case "pgup":
		m.clusterLogViewport.PageUp()
	case "pgdown":
		m.clusterLogViewport.PageDown()
	case "f":
		m.cycleLogFilter()
	case "left", "h":
		m.clusterLogViewport.ScrollLeft(m.clusterLogViewport.Width / 2)
	case "right":
		m.clusterLogViewport.ScrollRight(m.clusterLogViewport.Width / 2)
	}
	return m, nil
}

func (m *Model) handleClusterLogsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.cancelRunning()
		m.statusMessage = "cancelling..."
	case "l":
		m.screen = screenCluster
		m.resize()
	case "up", "k":
		m.clusterLogViewport.LineUp(1)
	case "down", "j":
		m.clusterLogViewport.LineDown(1)
	case "pgup":
		m.clusterLogViewport.PageUp()
	case "pgdown":
		m.clusterLogViewport.PageDown()
	case "f":
		m.cycleLogFilter()
	case "left", "h":
		m.clusterLogViewport.ScrollLeft(m.clusterLogViewport.Width / 2)
	case "right":
		m.clusterLogViewport.ScrollRight(m.clusterLogViewport.Width / 2)
	}
	return m, nil
}

func (m *Model) handleDeleteDirKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "y":
		dir := m.dir
		if dir == "" || dir == "/" {
			m.screen = screenCommand
			return m, nil
		}
		if m.fileLog != nil {
			m.fileLog.SetFile(nil)
		}
		if err := os.RemoveAll(dir); err != nil {
			m.lastStage = StageNone
			m.screen = screenFailed
			m.errMessage = "cannot delete " + dir + ": " + err.Error()
			m.details = nil
			return m, nil
		}
		m.clearDirectorySelection()
		m.showSuccessFrom("Directory deleted", []string{dir}, screenCommand)
	case "n", "esc", "enter":
		m.screen = screenCommand
	}
	return m, nil
}

func (m *Model) handleCompleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "enter":
		return m, tea.Quit
	case "m":
		if m.detection != nil {
			m.screen = screenMenu
			m.menuIndex = 0
		}
	}
	return m, nil
}

func (m *Model) openLogs() {
	m.logsReturn = m.screen

	path := filepath.Join(m.dir, logFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		m.fileLogLines = []string{"cannot read " + path + ": " + err.Error()}
	} else {
		lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
		const maxLines = 5000
		if len(lines) > maxLines {
			lines = lines[len(lines)-maxLines:]
		}
		m.fileLogLines = lines
	}

	m.fileLogViewport.SetContent(strings.Join(filterLogLines(m.fileLogLines, m.logFilter), "\n"))
	m.fileLogViewport.GotoBottom()
	m.screen = screenLogs
	m.resize()
}

func (m *Model) handleLogsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = m.logsReturn
		return m, nil
	case "q":
		return m, tea.Quit
	case "f":
		m.cycleLogFilter()
		return m, nil
	case "left", "h":
		m.fileLogViewport.ScrollLeft(m.fileLogViewport.Width / 2)
		return m, nil
	case "right":
		m.fileLogViewport.ScrollRight(m.fileLogViewport.Width / 2)
		return m, nil
	}

	var cmd tea.Cmd
	m.fileLogViewport, cmd = m.fileLogViewport.Update(msg)
	return m, cmd
}

func (m *Model) handleStageSuccessKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.advanceRunID = 0
	switch msg.String() {
	case "l":
		m.openLogs()
		return m, nil
	case "m":
		return m, m.runStage(StageManifests)
	case "i":
		return m, m.runStage(StageIgnitionConfigs)
	case "c":
		return m, m.runStage(StageCluster)
	case "esc", "n":
		m.screen = screenMenu
		m.menuIndex = 0
		return m, nil
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) handleFailedKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "g":
		return m, m.startGather(screenFailed)
	case "a":
		return m, m.startAnalyze(screenFailed)
	case "l":
		m.openLogs()
		return m, nil
	case "r":
		return m, m.runStage(m.lastStage)
	case "e":
		if m.lastStage == StageInstallConfig {
			m.openInstallConfigForm()
		}
		return m, nil
	case "m":
		m.screen = screenMenu
		m.menuIndex = 0
	}
	return m, nil
}

func (m *Model) handlePlaceholderKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc", "m", "enter":
		m.screen = m.placeholderReturn
		if m.screen == screenMenu {
			m.menuIndex = 0
		}
	}
	return m, nil
}

func (m *Model) showPlaceholder(title string, body []string) {
	m.showPlaceholderFrom(title, body, screenMenu)
}

func (m *Model) showPlaceholderFrom(title string, body []string, returnTo screen) {
	m.screen = screenPlaceholder
	m.placeholderReturn = returnTo
	m.placeholderOK = false
	m.completionTitle = title
	m.completionBody = body
	m.running = false
	if m.uiLog != nil {
		m.uiLog.SetRunID(0)
	}
}

func (m *Model) showSuccessFrom(title string, body []string, returnTo screen) {
	m.showPlaceholderFrom(title, body, returnTo)
	m.placeholderOK = true
}

func (m *Model) setCancel(cancel context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancelRun = cancel
}

func (m *Model) canCancel() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cancelRun != nil
}

func (m *Model) cancelRunning() {
	m.mu.Lock()
	cancel := m.cancelRun
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (m *Model) runStage(stage Stage) tea.Cmd {
	if stage == StageNone {
		return nil
	}

	m.menuIndex = stageIndex(stage)
	m.pendingStages = m.stagesToRun(stage)
	return m.advanceStageQueue()
}

// stagesToRun returns the stages required to reach target from the directory's
// current stage. Rerunning an already-completed stage only runs that stage.
func (m *Model) stagesToRun(target Stage) []Stage {
	current := StageNone
	if m.detection != nil {
		current = m.detection.Last
	} else if m.dir != "" {
		if detection, err := Detect(m.dir); err == nil {
			m.detection = detection
			current = detection.Last
		}
	}
	return stagesFrom(current, target)
}

func (m *Model) advanceStageQueue() tea.Cmd {
	if len(m.pendingStages) == 0 {
		return nil
	}
	stage := m.pendingStages[0]
	m.pendingStages = m.pendingStages[1:]
	m.lastStage = stage
	return m.startStage(stage)
}

func (m *Model) clearStageQueue() {
	m.pendingStages = nil
}

func (m *Model) handleRunDone(msg runDoneMsg) (tea.Model, tea.Cmd) {
	m.running = false
	m.setCancel(nil)
	if m.uiLog != nil {
		m.uiLog.SetRunID(0)
	}
	m.flushLogs()

	detection, err := Detect(m.dir)
	if err == nil {
		m.detection = detection
	}
	m.lastStage = msg.stage

	if msg.cancelled {
		logrus.Warnf("Run cancelled: %s", msg.stage.Title())
		m.advanceRunID = 0
		m.clearStageQueue()
		m.screen = screenFailed
		m.errMessage = "command cancelled"
		m.details = nil
		return m, nil
	}

	if msg.err != nil {
		logrus.Errorf("Run failed: %s: %v", msg.stage.Title(), msg.err)
		m.advanceRunID = 0
		m.clearStageQueue()
		m.screen = screenFailed
		m.errMessage = msg.err.Error()

		errs, scanErr := scanErrorsSinceLastRun(m.dir)
		switch {
		case scanErr == nil:
			m.details = logErrorDetails(errs)
		case m.detection != nil:
			m.details = errorDetails(m.detection)
		}
		return m, nil
	}

	logrus.Infof("Run completed: %s", msg.stage.Title())
	m.errMessage = ""
	m.details = nil
	if msg.stage == StageCluster {
		m.clearStageQueue()
		m.screen = screenComplete
		m.completionTitle = "Cluster installation finished"
		m.completionBody = completionBody(m.dir)
	} else if len(m.pendingStages) > 0 {
		m.screen = screenStageSuccess
		m.completionTitle = msg.stage.Title()
		m.completionBody = nil
		m.advanceRunID = msg.runID
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return stageAdvanceMsg{runID: msg.runID}
		})
	} else {
		m.screen = screenStageSuccess
		m.completionTitle = msg.stage.Title()
		m.completionBody = nil
	}
	return m, nil
}

type logFilter int

const (
	logFilterAll logFilter = iota
	logFilterDebug
	logFilterInfo
	logFilterWarn
	logFilterError
)

func (f logFilter) String() string {
	switch f {
	case logFilterDebug:
		return "debug"
	case logFilterInfo:
		return "info"
	case logFilterWarn:
		return "warn"
	case logFilterError:
		return "error"
	default:
		return "all"
	}
}

func (f logFilter) next() logFilter {
	switch f {
	case logFilterAll:
		return logFilterDebug
	case logFilterDebug:
		return logFilterInfo
	case logFilterInfo:
		return logFilterWarn
	case logFilterWarn:
		return logFilterError
	default:
		return logFilterAll
	}
}

func logLevelValue(line string) int {
	lower := strings.ToLower(line)
	for _, level := range []struct {
		name  string
		value int
	}{
		{"panic", 6},
		{"fatal", 5},
		{"error", 4},
		{"warn", 3},
		{"warning", 3},
		{"info", 2},
		{"debug", 1},
		{"trace", 0},
	} {
		if strings.Contains(lower, "level="+level.name) {
			return level.value
		}
	}
	return -1
}

func filterLogLines(lines []string, filter logFilter) []string {
	if filter == logFilterAll {
		return lines
	}

	threshold := 1
	switch filter {
	case logFilterInfo:
		threshold = 2
	case logFilterWarn:
		threshold = 3
	case logFilterError:
		threshold = 4
	}

	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		level := logLevelValue(line)
		if level < 0 || level >= threshold {
			filtered = append(filtered, line)
		}
	}
	return filtered
}

func (m *Model) cycleLogFilter() {
	m.logFilter = m.logFilter.next()
	m.logDirty = true
	m.flushLogs()

	m.fileLogViewport.SetContent(strings.Join(filterLogLines(m.fileLogLines, m.logFilter), "\n"))
	m.fileLogViewport.GotoBottom()
}

func (m *Model) appendLog(line string) {
	for _, part := range strings.Split(strings.TrimRight(line, "\r\n"), "\n") {
		m.logLines = append(m.logLines, part)
	}
	const maxLines = 2000
	if len(m.logLines) > maxLines {
		m.logLines = m.logLines[len(m.logLines)-maxLines:]
	}
	m.logDirty = true
}

func (m *Model) flushLogs() {
	if !m.logDirty {
		return
	}
	m.logDirty = false
	content := strings.Join(filterLogLines(m.logLines, m.logFilter), "\n")

	atBottom := m.logViewport.AtBottom()
	m.logViewport.SetContent(content)
	if atBottom {
		m.logViewport.GotoBottom()
	}

	clusterAtBottom := m.clusterLogViewport.AtBottom()
	m.clusterLogViewport.SetContent(content)
	if m.screen == screenCluster || clusterAtBottom {
		m.clusterLogViewport.GotoBottom()
	}
}

type flushLogsMsg struct{}

func tickLogs() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		return flushLogsMsg{}
	})
}

func stageIndex(s Stage) int {
	for i, stage := range stages {
		if stage == s {
			return i
		}
	}
	return 0
}

func errorDetails(d *Detection) []string {
	if d == nil {
		return nil
	}
	return logErrorDetails(d.Errors)
}

func logErrorDetails(errs []LogError) []string {
	out := make([]string, 0, len(errs))
	for _, e := range errs {
		out = append(out, fmt.Sprintf("line %d: %s", e.Line, e.Text))
	}
	return out
}

func completionBody(dir string) []string {
	body := []string{
		"Assets directory: " + dir,
	}
	kubeconfig := filepath.Join(dir, "auth", "kubeconfig")
	if fileExists(kubeconfig) {
		body = append(body, "Kubeconfig: "+kubeconfig)
	}
	metadata := filepath.Join(dir, metadataName)
	if fileExists(metadata) {
		body = append(body, "Metadata: "+metadata)
	}
	return body
}
