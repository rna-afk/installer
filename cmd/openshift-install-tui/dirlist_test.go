package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRefreshDirChoices(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"alpha", "beta", "zebra", "papa"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, ".hidden"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModel(options{})
	m.dirInput.SetValue(dir)
	m.refreshDirChoices()

	assertArgs(t, dirChoiceNames(m.dirChoices), []string{"alpha", "beta", "papa", "zebra"})
	if m.dirListBase != dir {
		t.Fatalf("dirListBase = %q, want %q", m.dirListBase, dir)
	}
}

func TestRefreshDirChoicesPrefix(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"alpha", "papa", "pavel"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	m := newModel(options{})
	m.dirInput.SetValue(filepath.Join(dir, "pa"))
	m.refreshDirChoices()

	assertArgs(t, dirChoiceNames(m.dirChoices), []string{"papa", "pavel"})
	if m.dirListBase != dir {
		t.Fatalf("dirListBase = %q, want %q", m.dirListBase, dir)
	}
}

func TestRefreshDirChoicesOfExistingDir(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(child, "grandchild"), 0o755); err != nil {
		t.Fatal(err)
	}

	m := newModel(options{})
	m.dirInput.SetValue(child)
	m.refreshDirChoices()

	assertArgs(t, dirChoiceNames(m.dirChoices), []string{"grandchild"})
	if m.dirListBase != child {
		t.Fatalf("dirListBase = %q, want %q", m.dirListBase, child)
	}
}

func TestOpenDirectoryDefaultsToFirstChoice(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"beta", "alpha"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)

	m := newModel(options{})
	m.openDirectory()

	if got, want := m.dir, filepath.Join(dir, "alpha"); got != want {
		t.Fatalf("dir = %q, want %q", got, want)
	}
}

func TestOpenDirectoryFallsBackToCurrentDir(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	m := newModel(options{})
	m.openDirectory()

	if got := m.dir; got != dir {
		t.Fatalf("dir = %q, want %q", got, dir)
	}
}

func TestMoveDirChoice(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"alpha", "beta", "zebra"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	m := newModel(options{})
	m.dirInput.SetValue(dir)
	m.refreshDirChoices()

	m.moveDirChoice(1)
	if got := m.dirInput.Value(); got != filepath.Join(dir, "beta") {
		t.Fatalf("input = %q", got)
	}

	m.moveDirChoice(-1)
	if got := m.dirInput.Value(); got != filepath.Join(dir, "alpha") {
		t.Fatalf("input = %q", got)
	}

	m.moveDirChoice(-1)
	if got := m.dirInput.Value(); got != filepath.Join(dir, "alpha") {
		t.Fatalf("input should stay at the first entry, got %q", got)
	}

	m.moveDirChoice(100)
	if got := m.dirInput.Value(); got != filepath.Join(dir, "zebra") {
		t.Fatalf("input = %q", got)
	}
}

func dirChoiceNames(choices []dirChoice) []string {
	out := make([]string, len(choices))
	for i, c := range choices {
		out[i] = c.Name
	}
	return out
}
