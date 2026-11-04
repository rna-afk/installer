package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
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
	if !strings.Contains(lines[len(lines)-1], "enter run") {
		t.Fatalf("footer is not on the last line: %q", lines[len(lines)-1])
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

func TestBoxFillsTerminalWidth(t *testing.T) {
	m := newModel(options{})
	m.width = 120
	m.height = 40

	view := m.View()
	for _, line := range strings.Split(view, "\n") {
		if !strings.Contains(line, "╭") {
			continue
		}
		if got := lipgloss.Width(line); got != m.width-2 {
			t.Fatalf("box width = %d, want %d", got, m.width-2)
		}
		return
	}
	t.Fatal("did not find the top border")
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
	if !strings.Contains(lines[len(lines)-1], "cancel") {
		t.Fatalf("footer is not on the last line: %q", lines[len(lines)-1])
	}
}
