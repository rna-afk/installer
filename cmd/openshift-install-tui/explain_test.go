package main

import (
	"io"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func testExplainSchema() *apiextv1.JSONSchemaProps {
	return &apiextv1.JSONSchemaProps{
		Type: "object",
		Properties: map[string]apiextv1.JSONSchemaProps{
			"apiVersion": {Type: "string"},
			"metadata": {
				Type: "object",
				Properties: map[string]apiextv1.JSONSchemaProps{
					"name": {Type: "string"},
				},
			},
			"platform": {
				Type: "object",
				Properties: map[string]apiextv1.JSONSchemaProps{
					"azure": {
						Type: "object",
						Properties: map[string]apiextv1.JSONSchemaProps{
							"dataDisks": {
								Type: "array",
								Items: &apiextv1.JSONSchemaPropsOrArray{
									Schema: &apiextv1.JSONSchemaProps{
										Type: "object",
										Properties: map[string]apiextv1.JSONSchemaProps{
											"managedDisk": {
												Type: "object",
												Properties: map[string]apiextv1.JSONSchemaProps{
													"id": {Type: "string"},
												},
											},
										},
									},
								},
							},
							"region": {Type: "string"},
						},
					},
				},
			},
		},
	}
}

func useTestExplain(t *testing.T) {
	t.Helper()
	oldLoad, oldRender := loadExplainSchema, renderExplain
	loadExplainSchema = func() (*apiextv1.JSONSchemaProps, error) {
		return testExplainSchema(), nil
	}
	renderExplain = func(path string, w io.Writer) error {
		_, err := io.WriteString(w, "explain "+path)
		return err
	}
	t.Cleanup(func() {
		loadExplainSchema = oldLoad
		renderExplain = oldRender
	})
}

func newExplainTestModel(t *testing.T) *Model {
	t.Helper()
	useTestExplain(t)
	m := newModel(options{})
	m.runCommand(commandExplain)
	return m
}

func TestExplainNavigation(t *testing.T) {
	m := newExplainTestModel(t)
	if m.screen != screenExplain {
		t.Fatalf("screen = %v, want explain", m.screen)
	}
	if got := m.explain.path[0]; got != "apiVersion" {
		t.Fatalf("initial selection = %q, want apiVersion", got)
	}

	m.handleExplainKey(tea.KeyMsg{Type: tea.KeyDown})
	if got := m.explain.path[0]; got != "metadata" {
		t.Fatalf("selection = %q, want metadata", got)
	}
	m.handleExplainKey(tea.KeyMsg{Type: tea.KeyDown})
	if got := m.explain.path[0]; got != "platform" {
		t.Fatalf("selection = %q, want platform", got)
	}

	m.handleExplainKey(tea.KeyMsg{Type: tea.KeyRight})
	if m.explain.depth != 1 || m.explain.path[1] != "azure" {
		t.Fatalf("path = %v depth = %d, want platform.azure", m.explain.path, m.explain.depth)
	}
	m.handleExplainKey(tea.KeyMsg{Type: tea.KeyRight})
	if m.explain.depth != 2 || m.explain.path[2] != "dataDisks" {
		t.Fatalf("path = %v depth = %d, want platform.azure.dataDisks", m.explain.path, m.explain.depth)
	}
	m.handleExplainKey(tea.KeyMsg{Type: tea.KeyRight})
	if m.explain.depth != 3 || m.explain.path[3] != "managedDisk" {
		t.Fatalf("path = %v depth = %d, want managedDisk", m.explain.path, m.explain.depth)
	}
	m.handleExplainKey(tea.KeyMsg{Type: tea.KeyRight})
	if m.explain.depth != 4 || m.explain.path[4] != "id" {
		t.Fatalf("path = %v depth = %d, want id", m.explain.path, m.explain.depth)
	}

	m.handleExplainKey(tea.KeyMsg{Type: tea.KeyLeft})
	if m.explain.depth != 3 {
		t.Fatalf("depth = %d, want 3", m.explain.depth)
	}
}

func TestExplainSearch(t *testing.T) {
	m := newExplainTestModel(t)

	m.handleExplainKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.explain.searching {
		t.Fatal("expected search mode")
	}
	for _, r := range "meta" {
		m.updateFocused(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if got := m.explain.path[0]; got != "metadata" {
		t.Fatalf("selection = %q, want metadata", got)
	}

	m.handleExplainKey(tea.KeyMsg{Type: tea.KeyEscape})
	if m.explain.searching {
		t.Fatal("expected search mode to end")
	}
	if got := m.explain.filters[0]; got != "" {
		t.Fatalf("filter = %q, want empty", got)
	}
}

func TestExplainGlobalSearch(t *testing.T) {
	m := newExplainTestModel(t)

	m.handleExplainKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if !m.explain.globalSearching {
		t.Fatal("expected global search mode")
	}
	for _, r := range "managed" {
		m.updateFocused(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if len(m.explain.globalMatches) == 0 {
		t.Fatal("expected global search matches")
	}

	m.handleExplainKey(tea.KeyMsg{Type: tea.KeyEnter})
	if m.explain.globalSearching {
		t.Fatal("expected global search to close")
	}
	if got := strings.Join(m.explain.path, "."); !strings.Contains(got, "managedDisk") {
		t.Fatalf("jumped to %q, want a managedDisk path", got)
	}
	if m.explain.depth != len(m.explain.path)-1 {
		t.Fatalf("depth = %d, path length = %d", m.explain.depth, len(m.explain.path))
	}
}

func TestExplainGlobalSearchView(t *testing.T) {
	m := newExplainTestModel(t)
	m.width, m.height = 120, 40
	m.resize()

	m.handleExplainKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	for _, r := range "azure" {
		m.updateFocused(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	view := m.View()
	for _, want := range []string{"Search InstallConfig schema", "matching paths", "installconfig.platform.azure"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view is missing %q:\n%s", want, view)
		}
	}
}

func TestExplainDetailScreen(t *testing.T) {
	m := newExplainTestModel(t)

	m.handleExplainKey(tea.KeyMsg{Type: tea.KeyEnter})

	if m.screen != screenExplainDetail {
		t.Fatalf("screen = %v, want explain detail", m.screen)
	}
	if got := m.explain.detailPath; got != "installconfig.apiVersion" {
		t.Fatalf("detailPath = %q", got)
	}
	if !strings.Contains(m.explain.detail.View(), "explain installconfig.apiVersion") {
		t.Fatalf("detail content = %q", m.explain.detail.View())
	}

	m.handleExplainDetailKey(tea.KeyMsg{Type: tea.KeyEscape})
	if m.screen != screenExplain {
		t.Fatalf("screen = %v, want explain", m.screen)
	}
}

func TestExplainView(t *testing.T) {
	m := newExplainTestModel(t)
	m.width, m.height = 120, 40
	m.resize()

	view := m.View()
	for _, want := range []string{"InstallConfig schema", "apiVersion", "platform", "enter explain"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view is missing %q:\n%s", want, view)
		}
	}
}
