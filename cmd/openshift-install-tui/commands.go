package main

import tea "github.com/charmbracelet/bubbletea"

// command is one of the top-level openshift-install commands exposed by the
// TUI. Commands that are not wired up yet still appear in the picker so the
// TUI mirrors the installer's command surface.
type command int

const (
	commandNone command = iota
	commandInstall
	commandGather
	commandWaitFor
	commandExplain
	commandAnalyze
	commandDestroy
	commandCompletion
)

// waitForStage is one of the openshift-install wait-for subcommands.
type waitForStage int

const (
	waitForBootstrapComplete waitForStage = iota
	waitForInstallComplete
)

var waitForStages = []waitForStage{
	waitForBootstrapComplete,
	waitForInstallComplete,
}

func (s waitForStage) String() string {
	switch s {
	case waitForBootstrapComplete:
		return "bootstrap-complete"
	case waitForInstallComplete:
		return "install-complete"
	default:
		return ""
	}
}

func (s waitForStage) Title() string {
	switch s {
	case waitForBootstrapComplete:
		return "Bootstrap complete"
	case waitForInstallComplete:
		return "Install complete"
	default:
		return ""
	}
}

func (s waitForStage) Description() string {
	switch s {
	case waitForBootstrapComplete:
		return "Wait until cluster bootstrapping has completed"
	case waitForInstallComplete:
		return "Wait until the cluster is ready"
	default:
		return ""
	}
}

// commandChoices is the order shown on the command screen.
var commandChoices = []command{
	commandInstall,
	commandGather,
	commandWaitFor,
	commandExplain,
	commandAnalyze,
	commandDestroy,
	commandCompletion,
}

// Title is the human-friendly name shown in the UI.
func (c command) Title() string {
	switch c {
	case commandInstall:
		return "Install"
	case commandGather:
		return "Gather"
	case commandWaitFor:
		return "Wait for"
	case commandExplain:
		return "Explain"
	case commandAnalyze:
		return "Analyze"
	case commandDestroy:
		return "Destroy"
	case commandCompletion:
		return "Completion"
	default:
		return "None"
	}
}

// Description is the one-line hint shown for the selected command.
func (c command) Description() string {
	switch c {
	case commandInstall:
		return "Create install-config, manifests, ignition configs, and the cluster"
	case commandGather:
		return "Gather bootstrap logs from a failed cluster"
	case commandWaitFor:
		return "Wait for bootstrap or install completion"
	case commandExplain:
		return "Show install-config fields and documentation"
	case commandAnalyze:
		return "Analyze a gathered log bundle"
	case commandDestroy:
		return "Destroy the cluster resources"
	case commandCompletion:
		return "Generate shell completion scripts"
	default:
		return ""
	}
}

// NeedsDir reports whether the command operates on an assets directory.
func (c command) NeedsDir() bool {
	switch c {
	case commandWaitFor, commandExplain, commandCompletion:
		return false
	default:
		return true
	}
}

// usage lists the equivalent CLI invocations for a command that is not wired
// into the TUI yet.
func (c command) usage() []string {
	switch c {
	case commandWaitFor:
		return []string{
			"Subcommands:",
			"  openshift-install wait-for bootstrap-complete --dir <dir>",
			"  openshift-install wait-for install-complete --dir <dir>",
		}
	case commandExplain:
		return []string{
			"Examples:",
			"  openshift-install explain installconfig",
			"  openshift-install explain installconfig.platform.aws",
		}
	case commandDestroy:
		return []string{
			"Subcommands:",
			"  openshift-install destroy cluster --dir <dir>",
			"  openshift-install destroy bootstrap --dir <dir>",
		}
	case commandCompletion:
		return []string{
			"Subcommands:",
			"  openshift-install completion bash",
			"  openshift-install completion zsh",
		}
	default:
		return nil
	}
}

// runCommand prompts for a directory when the command needs one, otherwise it
// runs the command immediately.
func (m *Model) runCommand(c command) tea.Cmd {
	if c == commandInstall || c == commandDestroy {
		m.clearDirectorySelection()
	}
	if c.NeedsDir() && m.dir == "" {
		m.pendingCommand = c
		m.statusMessage = ""
		m.screen = screenDir
		m.dirInput.Focus()
		return nil
	}
	return m.executeCommand(c)
}

func (m *Model) executeCommand(c command) tea.Cmd {
	switch c {
	case commandInstall:
		return m.enterInstallFlow()
	case commandGather:
		return m.startGather(screenCommand)
	case commandAnalyze:
		return m.startAnalyze(screenCommand)
	case commandWaitFor:
		m.screen = screenWaitFor
		m.waitForIndex = 0
		return nil
	case commandExplain:
		return m.openExplain()
	case commandDestroy:
		return m.startDestroy(screenCommand)
	case commandCompletion:
		m.showCommandPlaceholder(c)
		return nil
	default:
		return nil
	}
}

func (m *Model) showCommandPlaceholder(c command) {
	body := []string{"Not wired up in the TUI yet.", ""}
	body = append(body, c.usage()...)
	m.showPlaceholderFrom(c.Title(), body, screenCommand)
}

// enterInstallFlow continues with the installer's stage flow based on the
// current directory detection.
func (m *Model) enterInstallFlow() tea.Cmd {
	d := m.detection
	if d == nil {
		detection, err := Detect(m.dir)
		if err != nil {
			m.screen = screenFailed
			m.errMessage = err.Error()
			m.details = nil
			return nil
		}
		m.detection = detection
		d = detection
	}

	switch {
	case d.Last == StageCluster || d.LastClusterRun:
		m.screen = screenClusterCheck
	case d.Last == StageNone:
		m.screen = screenMenu
		m.menuIndex = 0
	default:
		m.screen = screenResume
	}
	return nil
}
