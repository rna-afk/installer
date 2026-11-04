package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

// options holds the command-line flags accepted by the TUI. They mirror the
// installer's root flags plus the flags for the commands the TUI can drive.
type options struct {
	dir               string
	dirSet            bool
	logLevel          string
	showVersion       bool
	skipPasswordPrint bool

	// stage, when set, runs that stage immediately instead of showing the
	// resume/menu screens.
	stage Stage

	bootstrap    string
	masters      stringSlice
	keys         stringSlice
	skipAnalysis bool

	analyzeFile string
}

// stringSlice is a repeatable string flag.
type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

const usageText = `A terminal UI for the OpenShift installer.

Usage:
  openshift-install-tui [flags]

Flags:
      --dir string         assets directory; when explicitly set, the TUI opens it immediately (default ".")
      --log-level string   log level for in-process asset generation (e.g. "debug | info | warn | error") (default "info")
      --version            print version information and exit
      --config             open the install-config form immediately
      --manifests          run the manifests stage immediately
      --ignition           run the ignition-configs stage immediately
      --ignition-configs   alias for --ignition
      --cluster            show the cluster-stage placeholder immediately
      --skip-password-print
                           accepted for compatibility (cluster is a placeholder)
      --bootstrap string   bootstrap host prefilled in the gather form
      --master string      control plane host prefilled in the gather form (repeatable)
      --key string         SSH private key prefilled in the gather form (repeatable)
      --skipAnalysis       skip analysis of the gathered bundle
      --file string        accepted for compatibility (analyze is a placeholder)
  -h, --help               help for openshift-install-tui
`

func parseOptions(args []string) (options, error) {
	var opts options

	fs := flag.NewFlagSet("openshift-install-tui", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.dir, "dir", ".", "assets directory")
	fs.StringVar(&opts.logLevel, "log-level", "info", "log level")
	fs.BoolVar(&opts.showVersion, "version", false, "print version information")
	fs.BoolVar(&opts.skipPasswordPrint, "skip-password-print", false, "do not print the generated user password")
	fs.StringVar(&opts.bootstrap, "bootstrap", "", "bootstrap host")
	fs.Var(&opts.masters, "master", "control plane host (repeatable)")
	fs.Var(&opts.keys, "key", "SSH private key (repeatable)")
	fs.BoolVar(&opts.skipAnalysis, "skipAnalysis", false, "skip analysis of the gathered data")
	fs.StringVar(&opts.analyzeFile, "file", "", "log bundle for analyze")

	var installConfigFlag, manifestsFlag, ignitionFlag, ignitionConfigsFlag, clusterFlag bool
	fs.BoolVar(&installConfigFlag, "config", false, "run the install-config stage")
	fs.BoolVar(&manifestsFlag, "manifests", false, "run the manifests stage")
	fs.BoolVar(&ignitionFlag, "ignition", false, "run the ignition-configs stage")
	fs.BoolVar(&ignitionConfigsFlag, "ignition-configs", false, "run the ignition-configs stage")
	fs.BoolVar(&clusterFlag, "cluster", false, "run the cluster stage")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(os.Stdout, usageText)
			return opts, err
		}
		fmt.Fprintf(os.Stderr, "openshift-install-tui: %v\n\n%s", err, usageText)
		return opts, err
	}

	fs.Visit(func(f *flag.Flag) {
		if f.Name == "dir" {
			opts.dirSet = true
		}
	})

	if fs.NArg() > 0 {
		err := fmt.Errorf("unexpected argument: %s", fs.Arg(0))
		fmt.Fprintf(os.Stderr, "openshift-install-tui: %v\n\n%s", err, usageText)
		return opts, err
	}

	if !validLogLevel(opts.logLevel) {
		err := fmt.Errorf("invalid --log-level %q; expected debug, info, warn, or error", opts.logLevel)
		fmt.Fprintf(os.Stderr, "openshift-install-tui: %v\n\n%s", err, usageText)
		return opts, err
	}

	selected, err := resolveStageFlags(installConfigFlag, manifestsFlag, ignitionFlag, ignitionConfigsFlag, clusterFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "openshift-install-tui: %v\n\n%s", err, usageText)
		return opts, err
	}
	opts.stage = selected

	return opts, nil
}

func resolveStageFlags(installConfig, manifests, ignition, ignitionConfigs, cluster bool) (Stage, error) {
	var selected []Stage
	if installConfig {
		selected = append(selected, StageInstallConfig)
	}
	if manifests {
		selected = append(selected, StageManifests)
	}
	if ignition || ignitionConfigs {
		selected = append(selected, StageIgnitionConfigs)
	}
	if cluster {
		selected = append(selected, StageCluster)
	}

	switch len(selected) {
	case 0:
		return StageNone, nil
	case 1:
		return selected[0], nil
	default:
		return StageNone, errors.New("only one of --config, --manifests, --ignition, or --cluster may be used")
	}
}

func validLogLevel(level string) bool {
	switch strings.ToLower(level) {
	case "trace", "debug", "info", "warn", "warning", "error", "fatal", "panic":
		return true
	default:
		return false
	}
}
