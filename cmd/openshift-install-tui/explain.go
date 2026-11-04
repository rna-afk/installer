package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// loadExplainSchema is a variable for tests that need to avoid the embedded
// installer data assets.
var (
	loadExplainSchema = loadInstallConfigSchema
	renderExplain     = renderExplainPath
)

// explainState holds the schema browser state for the Explain screen.
type explainState struct {
	schema *apiextv1.JSONSchemaProps

	// path holds the selected field name for each visible column. Column 0
	// contains the fields of installconfig itself.
	path    []string
	filters []string
	depth   int

	searching bool
	search    textinput.Model

	allPaths        []explainPathEntry
	globalSearching bool
	globalSearch    textinput.Model
	globalMatches   []explainPathEntry
	globalIndex     int

	detail     viewport.Model
	detailPath string
}

type explainPathEntry struct {
	Path  []string
	Field schemaField
}

func newExplainState(schema *apiextv1.JSONSchemaProps) *explainState {
	search := textinput.New()
	search.Prompt = "/ "
	search.Placeholder = "filter this column"
	search.CharLimit = 128
	search.Width = 48

	globalSearch := textinput.New()
	globalSearch.Prompt = "? "
	globalSearch.Placeholder = "search all installconfig paths"
	globalSearch.CharLimit = 256
	globalSearch.Width = 64

	detail := viewport.New(80, 20)

	e := &explainState{
		schema:       schema,
		search:       search,
		globalSearch: globalSearch,
		detail:       detail,
	}
	e.filters = []string{""}
	e.allPaths = collectExplainPaths(schema)
	e.ensureSelection(0)
	return e
}

func collectExplainPaths(schema *apiextv1.JSONSchemaProps) []explainPathEntry {
	var out []explainPathEntry

	var walk func(prefix []string, current *apiextv1.JSONSchemaProps)
	walk = func(prefix []string, current *apiextv1.JSONSchemaProps) {
		for _, field := range schemaFields(current) {
			path := append(append([]string{}, prefix...), field.Name)
			out = append(out, explainPathEntry{Path: path, Field: field})
			walk(path, field.Schema)
		}
	}
	walk(nil, schema)
	return out
}

func (e *explainState) fieldsAt(depth int) []schemaField {
	schema := e.schema
	if depth > 0 {
		if depth > len(e.path) {
			return nil
		}
		next, err := lookupSchema(e.schema, e.path[:depth])
		if err != nil {
			return nil
		}
		schema = next
	}
	return schemaFields(schema)
}

func (e *explainState) visibleFieldsAt(depth int) []schemaField {
	fields := e.fieldsAt(depth)
	if depth >= len(e.filters) {
		return fields
	}
	filter := strings.ToLower(strings.TrimSpace(e.filters[depth]))
	if filter == "" {
		return fields
	}

	out := make([]schemaField, 0, len(fields))
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field.Name), filter) {
			out = append(out, field)
		}
	}
	return out
}

func (e *explainState) selectedIndex(depth int) int {
	if depth >= len(e.path) {
		return -1
	}
	for i, field := range e.visibleFieldsAt(depth) {
		if field.Name == e.path[depth] {
			return i
		}
	}
	return -1
}

func (e *explainState) fieldHasChildren(field schemaField) bool {
	return len(schemaFields(field.Schema)) > 0
}

func (e *explainState) selectedField(depth int) (schemaField, bool) {
	index := e.selectedIndex(depth)
	if index < 0 {
		return schemaField{}, false
	}
	return e.visibleFieldsAt(depth)[index], true
}

func (e *explainState) ensureFilters(depth int) {
	for len(e.filters) <= depth {
		e.filters = append(e.filters, "")
	}
}

func (e *explainState) ensureSelection(depth int) {
	fields := e.visibleFieldsAt(depth)
	if len(fields) == 0 {
		if depth < len(e.path) {
			e.path = e.path[:depth]
		}
		return
	}
	if e.selectedIndex(depth) >= 0 {
		return
	}

	e.ensureFilters(depth)
	if depth < len(e.path) {
		e.path[depth] = fields[0].Name
	} else {
		e.path = append(e.path, fields[0].Name)
	}
	if len(e.path) > depth+1 {
		e.path = e.path[:depth+1]
	}
	if len(e.filters) > depth+1 {
		e.filters = e.filters[:depth+1]
	}
}

func (e *explainState) selectIndex(depth, index int) {
	if depth > len(e.path) {
		return
	}
	fields := e.visibleFieldsAt(depth)
	if len(fields) == 0 {
		return
	}
	if index < 0 {
		index = 0
	}
	if index >= len(fields) {
		index = len(fields) - 1
	}

	e.ensureFilters(depth)
	e.depth = depth
	if depth < len(e.path) {
		e.path[depth] = fields[index].Name
		e.path = e.path[:depth+1]
	} else {
		e.path = append(e.path[:depth], fields[index].Name)
	}
	if len(e.filters) > depth+1 {
		e.filters = e.filters[:depth+1]
	}
}

func (e *explainState) move(delta int) {
	e.ensureSelection(e.depth)
	index := e.selectedIndex(e.depth)
	if index < 0 {
		return
	}
	e.selectIndex(e.depth, index+delta)
}

func (e *explainState) expand() {
	e.ensureSelection(e.depth)
	field, ok := e.selectedField(e.depth)
	if !ok {
		return
	}
	children := schemaFields(field.Schema)
	if len(children) == 0 {
		return
	}

	nextName := children[0].Name
	if len(e.path) > e.depth+1 {
		for _, child := range children {
			if child.Name == e.path[e.depth+1] {
				nextName = child.Name
				break
			}
		}
	}

	e.ensureFilters(e.depth + 1)
	e.path = append(e.path[:e.depth+1], nextName)
	e.filters = append(e.filters[:e.depth+1], "")
	e.depth++
	e.ensureSelection(e.depth)
}

func (e *explainState) collapse() {
	if e.depth > 0 {
		e.depth--
		e.ensureSelection(e.depth)
	}
}

func (e *explainState) currentPath() string {
	if len(e.path) == 0 {
		return "installconfig"
	}
	end := e.depth + 1
	if end > len(e.path) {
		end = len(e.path)
	}
	return "installconfig." + strings.Join(e.path[:end], ".")
}

func (e *explainState) columnCount() int {
	count := len(e.path)
	if e.depth+1 > count {
		count = e.depth + 1
	}
	if count == 0 {
		count = 1
	}
	return count
}

func (e *explainState) startSearch() {
	e.ensureFilters(e.depth)
	e.searching = true
	e.search.SetValue(e.filters[e.depth])
	e.search.CursorEnd()
	e.search.Focus()
}

func (e *explainState) stopSearch(clear bool) {
	if clear {
		e.ensureFilters(e.depth)
		e.filters[e.depth] = ""
		e.ensureSelection(e.depth)
	}
	e.searching = false
	e.search.Blur()
}

func (e *explainState) applySearch(value string) {
	e.ensureFilters(e.depth)
	e.filters[e.depth] = value
	e.ensureSelection(e.depth)
}

func (e *explainState) startGlobalSearch() {
	e.globalSearching = true
	e.globalSearch.SetValue("")
	e.globalSearch.CursorEnd()
	e.globalSearch.Focus()
	e.applyGlobalSearch("")
}

func (e *explainState) stopGlobalSearch() {
	e.globalSearching = false
	e.globalSearch.Blur()
}

func (e *explainState) applyGlobalSearch(value string) {
	query := strings.ToLower(strings.TrimSpace(value))
	if query == "" {
		e.globalMatches = nil
		e.globalIndex = 0
		return
	}

	e.globalMatches = e.globalMatches[:0]
	for _, entry := range e.allPaths {
		path := strings.ToLower("installconfig." + strings.Join(entry.Path, "."))
		if strings.Contains(path, query) {
			e.globalMatches = append(e.globalMatches, entry)
		}
	}

	const maxMatches = 250
	if len(e.globalMatches) > maxMatches {
		e.globalMatches = e.globalMatches[:maxMatches]
	}
	if e.globalIndex >= len(e.globalMatches) {
		e.globalIndex = len(e.globalMatches) - 1
	}
	if e.globalIndex < 0 {
		e.globalIndex = 0
	}
}

func (e *explainState) moveGlobalMatch(delta int) {
	if len(e.globalMatches) == 0 {
		return
	}
	e.globalIndex += delta
	if e.globalIndex < 0 {
		e.globalIndex = 0
	}
	if e.globalIndex >= len(e.globalMatches) {
		e.globalIndex = len(e.globalMatches) - 1
	}
}

func (e *explainState) jumpTo(entry explainPathEntry) {
	e.path = append([]string{}, entry.Path...)
	e.depth = len(e.path) - 1
	e.filters = make([]string, len(e.path))
	e.stopGlobalSearch()
	e.ensureSelection(e.depth)
}

func (m *Model) openExplain() tea.Cmd {
	schema, err := loadExplainSchema()
	if err != nil {
		m.lastStage = StageNone
		m.screen = screenFailed
		m.errMessage = err.Error()
		m.details = nil
		return nil
	}

	m.explain = newExplainState(schema)
	m.resize()
	m.screen = screenExplain
	return nil
}

func (m *Model) handleExplainKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	e := m.explain
	if e == nil {
		m.screen = screenCommand
		return m, nil
	}

	if e.globalSearching {
		switch msg.String() {
		case "esc":
			e.stopGlobalSearch()
			return m, nil
		case "enter":
			if len(e.globalMatches) > 0 {
				e.jumpTo(e.globalMatches[e.globalIndex])
			}
			return m, nil
		case "up":
			e.moveGlobalMatch(-1)
			return m, nil
		case "down":
			e.moveGlobalMatch(1)
			return m, nil
		}
		return m, m.updateFocused(msg)
	}

	if e.searching {
		switch msg.String() {
		case "esc":
			e.stopSearch(true)
			return m, nil
		case "enter":
			e.stopSearch(false)
			return m, nil
		case "up":
			e.move(-1)
			return m, nil
		case "down":
			e.move(1)
			return m, nil
		}
		return m, m.updateFocused(msg)
	}

	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		m.screen = screenCommand
	case "/":
		e.startSearch()
	case "s":
		e.startGlobalSearch()
	case "up", "k":
		e.move(-1)
	case "down", "j":
		e.move(1)
	case "right", "l":
		e.expand()
	case "left", "h":
		e.collapse()
	case "enter":
		return m, m.showExplainDetail()
	}
	return m, nil
}

func (m *Model) handleExplainDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		m.screen = screenExplain
		return m, nil
	}

	if m.explain == nil {
		m.screen = screenExplain
		return m, nil
	}
	var cmd tea.Cmd
	m.explain.detail, cmd = m.explain.detail.Update(msg)
	return m, cmd
}

func (m *Model) showExplainDetail() tea.Cmd {
	e := m.explain
	if e == nil {
		return nil
	}

	path := e.currentPath()
	var b strings.Builder
	if err := renderExplain(path, &b); err != nil {
		m.lastStage = StageNone
		m.screen = screenFailed
		m.errMessage = err.Error()
		m.details = nil
		return nil
	}

	e.detailPath = path
	e.detail.SetContent(b.String())
	e.detail.GotoTop()
	m.screen = screenExplainDetail
	return nil
}
