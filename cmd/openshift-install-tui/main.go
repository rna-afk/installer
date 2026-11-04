package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/openshift/installer/pkg/version"
)

func main() {
	opts, err := parseOptions(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		os.Exit(2)
	}

	if opts.showVersion {
		if err := printVersion(); err != nil {
			fmt.Fprintln(os.Stderr, "openshift-install-tui:", err)
			os.Exit(1)
		}
		return
	}

	m := newModel(opts)
	p := tea.NewProgram(m, tea.WithAltScreen())
	m.program = p
	m.installLogging()
	defer m.closeLogging()

	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "openshift-install-tui:", err)
		os.Exit(1)
	}
}

func printVersion() error {
	versionString, err := version.Version()
	if err != nil {
		return err
	}
	fmt.Printf("%s %s\n", os.Args[0], versionString)
	if version.Commit != "" {
		fmt.Printf("built from commit %s\n", version.Commit)
	}
	return nil
}
