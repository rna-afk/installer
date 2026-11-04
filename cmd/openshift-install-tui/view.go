package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	awstypes "github.com/openshift/installer/pkg/types/aws"
	azuretypes "github.com/openshift/installer/pkg/types/azure"
)

func (m *Model) View() string {
	var header strings.Builder
	header.WriteString(titleStyle.Render("OpenShift Installer TUI"))
	header.WriteString("\n")
	if m.dir != "" {
		header.WriteString(subtitleStyle.Render("Directory: " + m.dir))
	} else {
		header.WriteString(subtitleStyle.Render("No directory selected"))
	}
	if m.detection != nil {
		header.WriteString("  ")
		header.WriteString(m.stageBadge())
	}

	var body strings.Builder
	switch m.screen {
	case screenDir:
		body.WriteString(m.dirView())
	case screenCommand:
		body.WriteString(m.commandView())
	case screenWaitFor:
		body.WriteString(m.waitForView())
	case screenExplain:
		body.WriteString(m.explainView())
	case screenExplainDetail:
		body.WriteString(m.explainDetailView())
	case screenMenu:
		body.WriteString(m.menuView())
	case screenResume:
		body.WriteString(m.resumeView())
	case screenClusterCheck:
		body.WriteString(m.clusterCheckView())
	case screenRunning:
		body.WriteString(m.runningView())
	case screenCluster:
		body.WriteString(m.clusterView())
	case screenClusterLogs:
		body.WriteString(m.clusterLogsView())
	case screenDeleteDir:
		body.WriteString(m.placeholderView())
	case screenComplete:
		body.WriteString(m.completeView())
	case screenFailed:
		body.WriteString(m.failedView())
	case screenPlaceholder:
		body.WriteString(m.placeholderView())
	case screenInstallConfig:
		body.WriteString(m.installConfigView())
	case screenStageSuccess:
		body.WriteString(m.stageSuccessView())
	case screenLogs:
		body.WriteString(m.fileLogViewport.View())
	}

	footer := dimStyle.Render(m.footer())
	content := header.String() + "\n\n" + body.String()

	// Fill the terminal height so the footer is pinned to the bottom of the
	// alternate screen instead of floating under the content.
	if m.height > 0 {
		lines := strings.Count(content, "\n") + 1
		if padding := m.height - lines - 1; padding > 0 {
			content += strings.Repeat("\n", padding)
		}
	}

	return content + "\n" + footer
}

// renderBox renders content in the main bordered panel, expanding it to the
// available terminal area.
func (m *Model) renderBox(content string) string {
	style := boxStyle
	if m.width > 0 {
		width := m.width - 4
		if width < 20 {
			width = 20
		}
		style = style.Width(width)
	}
	if m.height > 0 {
		height := m.height - 6
		if height < 5 {
			height = 5
		}
		style = style.Height(height)
	}
	return style.Render(content)
}

func (m *Model) stageBadge() string {
	d := m.detection
	if d == nil || d.New {
		return warnStyle.Render("[new directory]")
	}
	if d.Last == StageNone {
		return warnStyle.Render("[no completed stage]")
	}
	return okStyle.Render("[last stage: " + d.Last.String() + "]")
}

func (m *Model) dirView() string {
	cwd, _ := os.Getwd()
	var b strings.Builder
	b.WriteString(titleStyle.Render("Where should the installer run?"))
	b.WriteString("\n\n")
	b.WriteString("Enter an existing assets directory or a new directory name.\n")
	b.WriteString(dimStyle.Render("Relative paths are resolved from " + cwd))
	b.WriteString("\n\n")
	b.WriteString(m.dirInput.View())
	b.WriteString("\n\n")

	if m.dirListBase != "" {
		b.WriteString(dimStyle.Render("Directories in " + m.dirListBase + ":"))
		b.WriteString("\n")
	}
	if len(m.dirChoices) == 0 {
		b.WriteString(dimStyle.Render("  (no subdirectories found; type a path)"))
		b.WriteString("\n")
	} else {
		names := make([]string, len(m.dirChoices))
		for i := range m.dirChoices {
			names[i] = m.dirChoices[i].Name
		}
		b.WriteString(m.renderWheel(names, m.dirChoiceIndex))
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("If the directory does not exist it will be created."))

	if m.errMessage != "" {
		b.WriteString("\n\n")
		b.WriteString(errStyle.Render(m.errMessage))
	}

	return m.renderBox(b.String())
}

// inputLoading renders a compact spinner for an in-flight form operation.
func (m *Model) inputLoading(text string) string {
	if len(m.inputSpinner.Spinner.Frames) == 0 {
		return dimStyle.Render(text)
	}
	return m.inputSpinner.View() + " " + dimStyle.Render(text)
}

func (m *Model) commandView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("What do you want to do?"))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Choose a command to run:"))
	b.WriteString("\n\n")

	titles := make([]string, len(commandChoices))
	for i, c := range commandChoices {
		titles[i] = c.Title()
	}
	b.WriteString(m.renderWheel(titles, m.commandIndex))

	if selected := commandChoices[m.commandIndex]; selected.Description() != "" {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(selected.Description()))
	}

	if m.statusMessage != "" {
		b.WriteString("\n")
		b.WriteString(warnStyle.Render(m.statusMessage))
	}
	return m.renderBox(b.String())
}

func (m *Model) waitForView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Wait for"))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Choose the event to wait for:"))
	b.WriteString("\n\n")

	titles := make([]string, len(waitForStages))
	for i, stage := range waitForStages {
		titles[i] = stage.Title()
	}
	b.WriteString(m.renderWheel(titles, m.waitForIndex))

	if selected := waitForStages[m.waitForIndex]; selected.Description() != "" {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(selected.Description()))
	}

	return m.renderBox(b.String())
}

func (m *Model) explainView() string {
	e := m.explain
	if e == nil {
		return m.renderBox("No explain data.")
	}
	if e.globalSearching {
		return m.explainGlobalSearchView(e)
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("InstallConfig schema"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(e.currentPath()))
	b.WriteString("\n\n")

	if e.searching {
		b.WriteString(selectedStyle.Render("Search:"))
		b.WriteString(" ")
		b.WriteString(e.search.View())
		b.WriteString("\n\n")
	}

	b.WriteString(m.explainColumnsView(e))
	b.WriteString("\n")

	if field, ok := e.selectedField(e.depth); ok {
		b.WriteString(labelStyle.Render(field.Name))
		if field.Required {
			b.WriteString(warnStyle.Render(" -required-"))
		}
		b.WriteString(" ")
		b.WriteString(dimStyle.Render(explainFieldType(field.Schema)))
		b.WriteString("\n")
		if desc := strings.TrimSpace(field.Schema.Description); desc != "" {
			b.WriteString(dimStyle.Render(truncateText(desc, m.width-8)))
			b.WriteString("\n")
		}
	} else {
		b.WriteString(dimStyle.Render("No matching fields in this column."))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("enter explain • → children • ← back • / filter • s search all"))
	return m.renderBox(b.String())
}

func (m *Model) explainGlobalSearchView(e *explainState) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Search InstallConfig schema"))
	b.WriteString("\n")
	b.WriteString(e.globalSearch.View())
	b.WriteString("\n\n")

	if len(e.globalMatches) == 0 {
		if strings.TrimSpace(e.globalSearch.Value()) == "" {
			b.WriteString(dimStyle.Render("Type to search all installconfig fields."))
		} else {
			b.WriteString(dimStyle.Render("No matching paths."))
		}
		b.WriteString("\n")
	} else {
		maxRows := clamp(m.height-18, 5, 15)
		start := e.globalIndex - maxRows/2
		if start < 0 {
			start = 0
		}
		end := start + maxRows
		if end > len(e.globalMatches) {
			end = len(e.globalMatches)
			start = end - maxRows
			if start < 0 {
				start = 0
			}
		}

		width := clamp(m.width-10, 20, 200)
		if start > 0 {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  ... %d more", start)))
			b.WriteString("\n")
		}
		for i := start; i < end; i++ {
			path := truncateText("installconfig."+strings.Join(e.globalMatches[i].Path, "."), width-2)
			if i == e.globalIndex {
				b.WriteString(wheelSelectedStyle.Width(width).Render("▸ " + path))
			} else {
				b.WriteString(dimStyle.Render("  " + path))
			}
			b.WriteString("\n")
		}
		if end < len(e.globalMatches) {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  ... %d more", len(e.globalMatches)-end)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("%d matching paths", len(e.globalMatches))))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("type to search • ↑/↓ select • enter jump • esc cancel"))
	return m.renderBox(b.String())
}

func (m *Model) explainColumnsView(e *explainState) string {
	total := e.columnCount()
	maxVisible := clamp((m.width-8)/22, 1, 4)
	if maxVisible < 1 {
		maxVisible = 1
	}

	start := 0
	if total > maxVisible {
		start = e.depth - maxVisible/2
		if start < 0 {
			start = 0
		}
		if start > total-maxVisible {
			start = total - maxVisible
		}
	}
	end := start + maxVisible
	if end > total {
		end = total
	}

	innerWidth := m.width - 8
	if innerWidth < 20 {
		innerWidth = 20
	}
	colWidth := (innerWidth - (end - start - 1)) / (end - start)
	colWidth = clamp(colWidth, 14, 34)

	columns := make([][]string, 0, end-start)
	for depth := start; depth < end; depth++ {
		columns = append(columns, m.explainColumnLines(e, depth, colWidth))
	}

	rows := 0
	for _, column := range columns {
		if len(column) > rows {
			rows = len(column)
		}
	}

	cellStyle := lipgloss.NewStyle().Width(colWidth)
	var b strings.Builder
	for row := 0; row < rows; row++ {
		for index, column := range columns {
			if index > 0 {
				b.WriteString(dimStyle.Render(" │ "))
			}
			cell := ""
			if row < len(column) {
				cell = column[row]
			}
			b.WriteString(cellStyle.Render(cell))
		}
		b.WriteString("\n")
	}
	if start > 0 || end < total {
		b.WriteString(dimStyle.Render(fmt.Sprintf("columns %d-%d of %d", start+1, end, total)))
		b.WriteString("\n")
	}
	return b.String()
}

func (m *Model) explainColumnLines(e *explainState, depth, width int) []string {
	header := "installconfig"
	if depth > 0 && depth-1 < len(e.path) {
		header = e.path[depth-1]
	}
	header = truncateText(header, width)
	headerStyle := labelStyle
	if depth == e.depth {
		headerStyle = selectedStyle
	}
	lines := []string{headerStyle.Render(header)}

	fields := e.visibleFieldsAt(depth)
	if len(fields) == 0 {
		lines = append(lines, dimStyle.Render("  (no matches)"))
		return lines
	}

	options := make([]string, len(fields))
	for i, field := range fields {
		name := field.Name
		if field.Schema.Type == "array" {
			name += "[]"
		}
		if e.fieldHasChildren(field) {
			name += " ›"
		}
		options[i] = truncateText(name, width-2)
	}

	selected := e.selectedIndex(depth)
	if selected < 0 {
		selected = 0
	}
	lines = append(lines, wheelLines(options, selected, m.wheelRows())...)
	return lines
}

func (m *Model) explainDetailView() string {
	e := m.explain
	if e == nil {
		return m.renderBox("No explain data.")
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Explain"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(e.detailPath))
	b.WriteString("\n\n")
	b.WriteString(e.detail.View())
	return m.renderBox(b.String())
}

func explainFieldType(schema *apiextv1.JSONSchemaProps) string {
	fieldType := schema.Type
	if schema.Items != nil && schema.Items.Schema != nil {
		fieldType = "[]" + schema.Items.Schema.Type
	}
	if fieldType == "" {
		return "<unknown>"
	}
	return "<" + fieldType + ">"
}

func truncateText(text string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= width {
		return string(runes)
	}
	if width == 1 {
		return string(runes[:1])
	}
	return string(runes[:width-1]) + "…"
}

func (m *Model) menuView() string {
	title := "No installer state found"
	if m.detection != nil && !m.detection.New {
		title = "Choose the next stage"
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Choose the stage you want to run; missing prerequisites run automatically:"))
	b.WriteString("\n\n")

	titles := make([]string, len(stages))
	for i, stage := range stages {
		titles[i] = fmt.Sprintf("%d. %s", i+1, stage.Title())
	}
	b.WriteString(m.renderWheel(titles, m.menuIndex))

	if m.statusMessage != "" {
		b.WriteString("\n")
		b.WriteString(warnStyle.Render(m.statusMessage))
	}

	return m.renderBox(b.String())
}

func (m *Model) resumeView() string {
	d := m.detection
	if d == nil {
		return m.renderBox("No detection result.")
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Installer state found"))
	b.WriteString("\n\n")
	b.WriteString(labelStyle.Render("Last completed stage: "))
	b.WriteString(okStyle.Render(d.Last.Title()))
	b.WriteString("\n")

	next := d.Next()
	if next == StageNone {
		b.WriteString(labelStyle.Render("Next stage: "))
		b.WriteString(warnStyle.Render("none (cluster stage already attempted)"))
	} else {
		b.WriteString(labelStyle.Render("Suggested next stage: "))
		b.WriteString(selectedStyle.Render(next.Title()))
	}
	b.WriteString("\n")

	if len(d.Evidence) > 0 {
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Evidence:"))
		b.WriteString("\n")
		for i, e := range dedupe(d.Evidence) {
			if i >= 8 {
				b.WriteString(dimStyle.Render("  ...\n"))
				break
			}
			b.WriteString("  • " + e + "\n")
		}
	}

	if next == StageNone {
		b.WriteString("\n")
		b.WriteString(warnStyle.Render("The cluster stage was already attempted. Use the cluster check screen to gather logs."))
	} else {
		b.WriteString("\n")
		b.WriteString(selectedStyle.Render(fmt.Sprintf("Continue with %s? [y]es / [n]o, choose another stage", next.Title())))
	}

	return m.renderBox(b.String())
}

func (m *Model) clusterCheckView() string {
	d := m.detection
	if d == nil {
		return m.renderBox("No detection result.")
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Cluster stage detected"))
	b.WriteString("\n\n")

	if len(d.Errors) == 0 {
		b.WriteString(okStyle.Render("No errors found in " + logFileName + " after the last cluster run."))
		b.WriteString("\n\n")
		b.WriteString("The cluster stage may have completed successfully, or the log may not\n")
		b.WriteString("contain the failure yet.\n")
	} else {
		b.WriteString(errStyle.Render(fmt.Sprintf("%d error line(s) found in %s:", len(d.Errors), logFileName)))
		b.WriteString("\n\n")
		for i, e := range tail(d.Errors, 10) {
			_ = i
			b.WriteString(errStyle.Render("  " + e.Text))
			b.WriteString("\n")
		}
		if len(d.Errors) > 10 {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  ... %d more error line(s)\n", len(d.Errors)-10)))
		}
	}

	b.WriteString("\n")
	if d.LogBundle != "" {
		b.WriteString(okStyle.Render("Existing log bundle: " + filepath.Base(d.LogBundle)))
	} else {
		b.WriteString(dimStyle.Render("No log bundle found."))
	}
	if m.statusMessage != "" {
		b.WriteString("\n")
		b.WriteString(warnStyle.Render(m.statusMessage))
	}

	b.WriteString("\n\n")
	if len(d.Errors) > 0 {
		b.WriteString(selectedStyle.Render("Gather logs now? [g]ather"))
		b.WriteString("   ")
	}
	b.WriteString("Options: [g] gather logs  [a] analyze bundle  [c] continue cluster  [l] full log  [m] menu  [q] quit")

	return m.renderBox(b.String())
}

func (m *Model) runningView() string {
	var b strings.Builder
	b.WriteString(m.spinner.View() + " " + titleStyle.Render(m.runLabel))
	b.WriteString("\n\n")

	content := m.logViewport.View()
	if strings.TrimSpace(m.logLinesContent()) == "" {
		content = dimStyle.Render("Waiting for output...")
	}
	b.WriteString(content)

	if m.statusMessage != "" {
		b.WriteString("\n\n")
		b.WriteString(warnStyle.Render(m.statusMessage))
	}
	return b.String()
}

func (m *Model) logLinesContent() string {
	return strings.Join(m.logLines, "\n")
}

func (m *Model) clusterView() string {
	d := m.clusterDiagram
	if d == nil {
		return m.renderBox("No cluster diagram.")
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Install Cluster"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Components appear left to right as they are created."))
	b.WriteString("\n\n")
	b.WriteString(m.clusterDiagramView(d))
	b.WriteString("\n\n")

	visible := d.visibleComponents()
	if len(visible) > 0 {
		selected := d.selected
		if selected < 0 || selected >= len(d.components) || d.components[selected].State == clusterPending {
			selected = visible[0]
		}
		component := d.components[selected]
		b.WriteString(labelStyle.Render(component.Name + ": "))
		b.WriteString(dimStyle.Render(component.Description))
		b.WriteString("\n")
	} else {
		b.WriteString(dimStyle.Render("Waiting for infrastructure creation..."))
		b.WriteString("\n")
	}

	if m.statusMessage != "" {
		b.WriteString("\n")
		b.WriteString(warnStyle.Render(m.statusMessage))
	}
	b.WriteString("\n\n")
	b.WriteString(labelStyle.Render("Logs"))
	b.WriteString(" ")
	b.WriteString(dimStyle.Render("[filter: " + m.logFilter.String() + "]"))
	b.WriteString("\n")
	if len(filterLogLines(m.logLines, m.logFilter)) == 0 {
		b.WriteString(dimStyle.Render("Waiting for installer output..."))
		b.WriteString("\n")
	} else {
		b.WriteString(m.clusterLogViewport.View())
		b.WriteString("\n")
	}
	b.WriteString(dimStyle.Render("l expand logs • f filter • pgup/pgdn scroll • ←/→ scroll"))
	return m.renderBox(b.String())
}

func (m *Model) clusterLogsView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Cluster install logs"))
	b.WriteString(" ")
	b.WriteString(dimStyle.Render("[filter: " + m.logFilter.String() + "]"))
	b.WriteString("\n\n")
	b.WriteString(m.clusterLogViewport.View())
	return m.renderBox(b.String())
}

func (m *Model) clusterDiagramView(d *clusterDiagram) string {
	visible := d.visibleComponents()
	if len(visible) == 0 {
		return dimStyle.Render("Waiting for the first component...")
	}

	fullWidth := clamp(m.width-8, 60, 120)
	boxWidth := clamp(fullWidth/4-4, 16, 26)
	connector := "\n" + dimStyle.Render(" ─▶ ") + "\n"

	var rows []string
	var row []string
	rowWidth := 0
	for _, index := range visible {
		box := m.clusterComponentBox(d, index, boxWidth)
		boxLen := lipgloss.Width(box)
		addLen := boxLen
		if len(row) > 0 {
			addLen += 4
		}
		if len(row) > 0 && rowWidth+addLen > fullWidth {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, row...))
			row = nil
			rowWidth = 0
		}
		if len(row) > 0 {
			row = append(row, connector)
			rowWidth += 4
		}
		row = append(row, box)
		rowWidth += boxLen
	}
	if len(row) > 0 {
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, row...))
	}
	return strings.Join(rows, "\n")
}

func (m *Model) clusterComponentBox(d *clusterDiagram, index, width int) string {
	component := d.components[index]
	title := component.Name
	body := ""
	switch index {
	case 0:
		body = "VPC / subnets"
	case 1:
		body = m.clusterDNSBody(d)
	case 2:
		body = "api :6443 / ingress"
	case 3:
		body = "[PC] bootstrap"
	case 4:
		body = "API + etcd"
	case 5, 6:
		body = "[PC][PC][PC]"
	}
	return m.clusterBox(title, body, component, index == d.selected, width)
}

func (m *Model) clusterBox(title, body string, component clusterComponent, selected bool, width int) string {
	border := clusterBorderColor(component, selected)
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1).
		Width(width)

	titleStyle := labelStyle
	if selected {
		titleStyle = selectedStyle
	}
	content := titleStyle.Render(clusterIcon(component) + " " + title)
	content += "\n" + body
	return style.Render(content)
}

func (m *Model) clusterDNSBody(d *clusterDiagram) string {
	component := d.components[1]
	count := 0
	switch component.State {
	case clusterRunning:
		count = 1 + int(time.Since(component.ChangedAt)/(700*time.Millisecond))
		if count > 3 {
			count = 3
		}
	case clusterDone:
		count = 3
	}
	if count == 0 {
		return dimStyle.Render("waiting")
	}
	entries := []string{
		"+ api",
		"+ api-int",
		"+ *.apps",
	}
	return dimStyle.Render(strings.Join(entries[:count], " "))
}

func clusterIcon(component clusterComponent) string {
	switch component.State {
	case clusterRunning:
		return "◐"
	case clusterDone:
		return "●"
	default:
		return "○"
	}
}

func clusterBorderColor(component clusterComponent, selected bool) lipgloss.Color {
	if selected {
		return lipgloss.Color("212")
	}
	switch component.State {
	case clusterRunning:
		if recentlyChanged(component) && (time.Now().UnixMilli()/250)%2 == 0 {
			return lipgloss.Color("226")
		}
		return lipgloss.Color("214")
	case clusterDone:
		return lipgloss.Color("42")
	default:
		return lipgloss.Color("240")
	}
}

func recentlyChanged(component clusterComponent) bool {
	return time.Since(component.ChangedAt) < 2500*time.Millisecond
}

func (m *Model) installConfigView() string {
	f := m.form
	if f == nil {
		return m.renderBox("No install-config form.")
	}

	var b strings.Builder
	position, total := f.stepPosition()
	b.WriteString(titleStyle.Render("Install Config"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("Step %d of %d: %s", position, total, f.stepTitle())))
	b.WriteString("\n\n")

	var summary []string
	platform := f.currentPlatform()
	if platform == "" {
		platform = "(not selected)"
	}
	summary = append(summary, "Platform: "+platform)
	if (f.currentPlatform() == awstypes.Name || f.currentPlatform() == azuretypes.Name) && strings.TrimSpace(f.region.Value()) != "" {
		summary = append(summary, "Region: "+strings.TrimSpace(f.region.Value()))
	}
	if strings.TrimSpace(f.baseDomain.Value()) != "" {
		summary = append(summary, "Base domain: "+strings.TrimSpace(f.baseDomain.Value()))
	}
	if len(summary) > 0 {
		b.WriteString(dimStyle.Render(strings.Join(summary, "  |  ")))
		b.WriteString("\n\n")
	}

	b.WriteString(m.installConfigStepView(f))
	b.WriteString("\n")

	if f.err != "" {
		b.WriteString(errStyle.Render(f.err))
		b.WriteString("\n")
	}

	switch f.step {
	case stepPlatform:
		b.WriteString(dimStyle.Render("↑/↓ choose platform • enter next • esc back"))
	case stepAzureCredentials:
		if f.azureConnecting {
			b.WriteString(dimStyle.Render("esc back"))
		} else {
			b.WriteString(dimStyle.Render("tab next field • enter connect • esc back"))
		}
	case stepRegion:
		if f.currentPlatform() == azuretypes.Name && f.azureConnecting {
			b.WriteString(dimStyle.Render("esc back"))
		} else if f.currentPlatform() == azuretypes.Name && len(f.azureRegions) > 0 {
			b.WriteString(dimStyle.Render("type/backspace to filter • ↑/↓ choose region • enter next • esc back"))
		} else {
			b.WriteString(dimStyle.Render("enter next • esc back"))
		}
	case stepBaseDomain:
		switch {
		case f.baseDomainLoading:
			b.WriteString(dimStyle.Render("esc back"))
		case f.needsAWSCreds:
			b.WriteString(dimStyle.Render("tab next field • enter fetch base domains • ctrl+g manual • esc back"))
		case f.baseDomainIsSelect():
			b.WriteString(dimStyle.Render("↑/↓ choose base domain • enter next • esc back"))
		default:
			b.WriteString(dimStyle.Render("enter next • esc back • ctrl+r retry lookup"))
		}
	case stepClusterName:
		b.WriteString(dimStyle.Render("enter next • esc back"))
	case stepSSHKey:
		b.WriteString(dimStyle.Render("↑/↓ choose SSH key • enter next • esc back"))
	case stepPublish:
		b.WriteString(dimStyle.Render("↑/↓ choose publish mode • enter next • esc back"))
	case stepPullSecret:
		b.WriteString(dimStyle.Render("enter save • esc back"))
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("ctrl+s saves from any step. Platform-specific fields beyond AWS region are preserved from an existing install-config.yaml."))

	return m.renderBox(b.String())
}

func (m *Model) installConfigStepView(f *installConfigForm) string {
	var b strings.Builder

	label := func(text string, active bool) string {
		if active {
			return selectedStyle.Render("> " + text)
		}
		return dimStyle.Render("  " + text)
	}
	inputRow := func(text, value string, active bool) {
		b.WriteString(label(text, active))
		b.WriteString(" ")
		b.WriteString(value)
		b.WriteString("\n")
	}

	switch f.step {
	case stepPlatform:
		b.WriteString(m.renderWheel(f.platforms, f.platformIdx))
	case stepAzureCredentials:
		if f.azureConnecting {
			b.WriteString(m.inputLoading("Connecting to Azure and fetching regions and DNS zones..."))
			b.WriteString("\n")
			return b.String()
		}
		azureRow := func(text, value string, index int) {
			prefix := "  "
			style := dimStyle
			if f.azureCredsFocus == index {
				prefix = "> "
				style = selectedStyle
			}
			b.WriteString(style.Render(prefix + text))
			b.WriteString(" ")
			b.WriteString(value)
			b.WriteString("\n")
		}
		azureRow("Subscription ID:", f.azureSubscriptionID.View(), 0)
		azureRow("Tenant ID:", f.azureTenantID.View(), 1)
		azureRow("Client ID:", f.azureClientID.View(), 2)
		azureRow("Client secret:", f.azureClientSecret.View(), 3)
		b.WriteString(dimStyle.Render("Values are prefilled from ~/.azure/osServicePrincipal.json when available. Enter on the last field connects to Azure."))
		b.WriteString("\n")
	case stepRegion:
		if f.currentPlatform() == azuretypes.Name {
			if f.azureConnecting {
				b.WriteString(m.inputLoading("Connecting to Azure and fetching regions and DNS zones..."))
				b.WriteString("\n")
				return b.String()
			}
			if len(f.azureRegions) > 0 {
				b.WriteString(selectedStyle.Render("> Search:"))
				b.WriteString(" ")
				b.WriteString(f.regionSearch.View())
				b.WriteString("\n\n")
				filtered := f.filteredAzureRegions()
				if len(filtered) == 0 {
					b.WriteString(dimStyle.Render("  (no matching regions)"))
					b.WriteString("\n")
				} else {
					b.WriteString(m.renderWheel(filtered, f.azureRegionIdx))
				}
			} else {
				inputRow("Azure region:", f.region.View(), true)
			}
			return b.String()
		}
		inputRow("AWS region:", f.region.View(), true)
	case stepBaseDomain:
		if f.needsAWSCreds {
			b.WriteString(errStyle.Render("Unable to list Route 53 hosted zones with the current credentials."))
			b.WriteString("\n\n")
			credsRow := func(text, value string, index int) {
				prefix := "  "
				style := dimStyle
				if f.credsFocus == index {
					prefix = "> "
					style = selectedStyle
				}
				b.WriteString(style.Render(prefix + text))
				b.WriteString(" ")
				b.WriteString(value)
				b.WriteString("\n")
			}
			credsRow("Access key ID:", f.awsAccessKey.View(), 0)
			credsRow("Secret access key:", f.awsSecretKey.View(), 1)
			credsRow("Session token:", f.awsSessionKey.View(), 2)
			if f.baseDomainErr != "" {
				b.WriteString(errStyle.Render("  " + f.baseDomainErr))
				b.WriteString("\n")
			}
			if f.baseDomainLoading {
				b.WriteString(m.inputLoading("Looking up Route 53 hosted zones in " + strings.TrimSpace(f.region.Value()) + "..."))
				b.WriteString("\n")
			} else {
				b.WriteString(dimStyle.Render("Enter on the last field fetches the base domains. Credentials are used for this TUI session. ctrl+g switches to manual entry."))
				b.WriteString("\n")
			}
			return b.String()
		}
		switch {
		case f.baseDomainLoading:
			b.WriteString(m.inputLoading("Looking up Route 53 hosted zones in " + strings.TrimSpace(f.region.Value()) + "..."))
			b.WriteString("\n")
		case f.baseDomainIsSelect():
			b.WriteString(m.renderWheel(f.baseDomainOptions, f.baseDomainIdx))
		default:
			inputRow("Base domain:", f.baseDomain.View(), true)
			if f.baseDomainErr != "" {
				b.WriteString(errStyle.Render("  " + f.baseDomainErr))
				b.WriteString("\n")
				b.WriteString(dimStyle.Render("  Enter a domain manually, or ctrl+r to retry the lookup."))
				b.WriteString("\n")
			}
		}
	case stepClusterName:
		inputRow("Cluster name:", f.clusterName.View(), true)
	case stepPublish:
		b.WriteString(m.renderWheel(f.publishes, f.publishIdx))
	case stepSSHKey:
		if len(f.sshKeyChoices) == 0 {
			b.WriteString(dimStyle.Render("  No public keys found in ~/.ssh."))
			b.WriteString("\n")
		} else {
			b.WriteString(m.renderWheel(f.sshKeyChoiceLabels(), f.sshKeyIdx))
		}
		b.WriteString(dimStyle.Render("  Public keys are discovered in ~/.ssh."))
		b.WriteString("\n")
	case stepPullSecret:
		inputRow("Pull secret:", f.pullSecret.View(), true)
		b.WriteString(dimStyle.Render("  Input is hidden. Paste the pull secret JSON."))
		b.WriteString("\n")
	}

	return b.String()
}

// wheelRows is the number of visible wheel rows, depending on terminal
// height. It always returns an odd number so the selection can stay centered.
func (m *Model) wheelRows() int {
	rows := clamp(m.height-16, 7, 15)
	if rows%2 == 0 {
		rows--
	}
	return rows
}

func wheelLines(options []string, index, rows int) []string {
	if len(options) == 0 {
		return nil
	}
	if index < 0 {
		index = 0
	}
	if index >= len(options) {
		index = len(options) - 1
	}
	if len(options) <= rows {
		lines := make([]string, 0, len(options))
		for i, option := range options {
			switch {
			case i == index:
				lines = append(lines, wheelSelectedStyle.Render("▸ "+option))
			case i == index-1 || i == index+1:
				lines = append(lines, dimStyle.Render("  "+option))
			default:
				lines = append(lines, faintStyle.Render("  "+option))
			}
		}
		return lines
	}
	if rows%2 == 0 {
		rows++
	}
	if rows < 1 {
		rows = 1
	}

	radius := rows / 2
	lines := make([]string, 0, rows)
	for offset := -radius; offset <= radius; offset++ {
		i := index + offset
		if i < 0 || i >= len(options) {
			lines = append(lines, "")
			continue
		}
		switch {
		case offset == 0:
			lines = append(lines, wheelSelectedStyle.Render("▸ "+options[i]))
		case offset == -1 || offset == 1:
			lines = append(lines, dimStyle.Render("  "+options[i]))
		default:
			lines = append(lines, faintStyle.Render("  "+options[i]))
		}
	}
	return lines
}

// renderWheel renders a date-picker style selector: the current choice stays
// in the middle while the entries above and below fade out.
func (m *Model) renderWheel(options []string, index int) string {
	lines := wheelLines(options, index, m.wheelRows())
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m *Model) placeholderView() string {
	icon := "⏳ "
	style := warnStyle
	if m.placeholderOK {
		icon = "✓ "
		style = okStyle
	}

	var b strings.Builder
	b.WriteString(style.Render(icon + m.completionTitle))
	b.WriteString("\n\n")
	for _, line := range m.completionBody {
		b.WriteString(line + "\n")
	}
	return m.renderBox(b.String())
}

func (m *Model) stageSuccessView() string {
	var b strings.Builder
	for _, line := range tail(m.logLines, 8) {
		b.WriteString(line)
		b.WriteString("\n")
	}
	if len(m.logLines) > 0 {
		b.WriteString("\n")
	}
	b.WriteString(okStyle.Render("✓ success: " + m.completionTitle))
	b.WriteString("\n\n")
	b.WriteString("Stages:\n\n")

	completed := StageNone
	if m.detection != nil {
		completed = m.detection.Last
	}
	for _, stage := range stages {
		switch {
		case stage <= completed:
			b.WriteString(okStyle.Render("  ✓ " + stage.Title()))
		case stageKey(stage) != "":
			b.WriteString(fmt.Sprintf("  %s  %s", stageKey(stage), stage.Title()))
		default:
			b.WriteString("     " + stage.Title())
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString("  l  Full installer logs\n")
	b.WriteString("\n")
	b.WriteString("  esc  Stage menu\n")
	b.WriteString("  q    Quit\n")
	return m.renderBox(b.String())
}

func stageKey(stage Stage) string {
	switch stage {
	case StageManifests:
		return "m"
	case StageIgnitionConfigs:
		return "i"
	case StageCluster:
		return "c"
	default:
		return ""
	}
}

func (m *Model) completeView() string {
	var b strings.Builder
	b.WriteString(okStyle.Render("✓ " + m.completionTitle))
	b.WriteString("\n\n")
	for _, line := range m.completionBody {
		b.WriteString(line + "\n")
	}
	return m.renderBox(b.String())
}

func (m *Model) failedView() string {
	var b strings.Builder
	b.WriteString(errStyle.Render("Command failed"))
	if m.errMessage != "" {
		b.WriteString("\n\n")
		b.WriteString(errStyle.Render(m.errMessage))
	}
	if len(m.details) > 0 {
		b.WriteString("\n\n")
		b.WriteString(labelStyle.Render("Recent errors:"))
		b.WriteString("\n")
		for _, line := range tail(m.details, 10) {
			b.WriteString(errStyle.Render(line))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n\n")
	b.WriteString("Press l to view the full installer log. You can gather logs, analyze an existing bundle, retry, or go back to the stage menu.")
	return m.renderBox(b.String())
}

func (m *Model) footer() string {
	switch m.screen {
	case screenDir:
		if m.opts.stage == StageNone {
			return "type a path • ↑/↓ pick a directory • enter open • esc back • ctrl+c quit"
		}
		return "type a path • ↑/↓ pick a directory • enter open • ctrl+c quit"
	case screenCommand:
		return "↑/↓ select • enter run • q quit"
	case screenWaitFor:
		return "↑/↓ select • enter continue • esc commands • q quit"
	case screenExplain:
		if m.explain != nil && m.explain.globalSearching {
			return "type to search • ↑/↓ select • enter jump • esc cancel"
		}
		if m.explain != nil && m.explain.searching {
			return "type to filter • enter keep • esc clear"
		}
		return "↑/↓ select • → children • ← back • / filter • s search all • enter explain • esc commands"
	case screenExplainDetail:
		return "↑/↓ scroll • pgup/pgdown • esc back • q quit"
	case screenMenu:
		return "↑/↓ select • 1-4 jump • enter run • esc commands • q quit"
	case screenResume:
		return "y continue • n choose another stage • q quit"
	case screenClusterCheck:
		return "g gather logs • a analyze bundle • c continue cluster • l logs • m menu • q quit"
	case screenRunning:
		if !m.canCancel() {
			return "↑/↓ scroll • pgup/pgdown • ←/→ scroll • f filter • operation cannot be cancelled"
		}
		return "q/esc cancel • ↑/↓ scroll • pgup/pgdown • ←/→ scroll • f filter"
	case screenCluster:
		return "↑/↓ component • l logs • f filter • q/esc cancel"
	case screenClusterLogs:
		return "↑/↓ scroll • pgup/pgdown • ←/→ scroll • f filter • l diagram • q/esc cancel"
	case screenDeleteDir:
		return "y delete directory • n keep directory • q quit"
	case screenComplete:
		return "m menu • q quit"
	case screenFailed:
		if m.lastStage == StageInstallConfig {
			return "e edit install-config • l logs • g gather logs • a analyze bundle • r retry • m menu • q quit"
		}
		return "l logs • g gather logs • a analyze bundle • r retry • m menu • q quit"
	case screenInstallConfig:
		return "tab next • ←/→ change • ctrl+s save • esc back"
	case screenPlaceholder:
		if m.placeholderReturn == screenMenu {
			return "esc/m menu • q quit"
		}
		return "esc back • q quit"
	case screenLogs:
		return "↑/↓ scroll • pgup/pgdown • ←/→ scroll • f filter • esc back • q quit"
	case screenStageSuccess:
		return "m manifests • i ignition • c cluster • l logs • esc menu • q quit"
	default:
		return "ctrl+c quit"
	}
}

func tail[T any](in []T, n int) []T {
	if len(in) <= n {
		return in
	}
	return in[len(in)-n:]
}
