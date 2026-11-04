package main

import (
	"errors"
	"flag"
	"testing"
)

func TestParseOptionsDefaults(t *testing.T) {
	opts, err := parseOptions(nil)
	if err != nil {
		t.Fatal(err)
	}
	if opts.dir != "." {
		t.Fatalf("dir = %q, want .", opts.dir)
	}
	if opts.dirSet {
		t.Fatal("dirSet should be false when --dir is not given")
	}
	if opts.logLevel != "info" {
		t.Fatalf("logLevel = %q, want info", opts.logLevel)
	}
}

func TestParseOptionsDirExplicit(t *testing.T) {
	opts, err := parseOptions([]string{"--dir", "/tmp/assets"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.dirSet {
		t.Fatal("dirSet should be true")
	}
	if opts.dir != "/tmp/assets" {
		t.Fatalf("dir = %q", opts.dir)
	}
}

func TestParseOptionsRepeatable(t *testing.T) {
	opts, err := parseOptions([]string{
		"--master", "a", "--master", "b",
		"--key", "k1", "--key", "k2",
		"--skipAnalysis",
		"--skip-password-print",
		"--bootstrap", "1.2.3.4",
		"--file", "bundle.tar.gz",
		"--log-level", "debug",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.masters) != 2 || opts.masters[0] != "a" || opts.masters[1] != "b" {
		t.Fatalf("masters = %v", opts.masters)
	}
	if len(opts.keys) != 2 || opts.keys[0] != "k1" || opts.keys[1] != "k2" {
		t.Fatalf("keys = %v", opts.keys)
	}
	if !opts.skipAnalysis || !opts.skipPasswordPrint {
		t.Fatal("boolean flags not set")
	}
	if opts.bootstrap != "1.2.3.4" || opts.analyzeFile != "bundle.tar.gz" {
		t.Fatalf("unexpected opts: %+v", opts)
	}
	if opts.logLevel != "debug" {
		t.Fatalf("logLevel = %q", opts.logLevel)
	}
}

func TestParseOptionsVersion(t *testing.T) {
	opts, err := parseOptions([]string{"--version"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.showVersion {
		t.Fatal("showVersion should be true")
	}
}

func TestParseStageFlags(t *testing.T) {
	tests := []struct {
		flag string
		want Stage
	}{
		{"--config", StageInstallConfig},
		{"--manifests", StageManifests},
		{"--ignition", StageIgnitionConfigs},
		{"--ignition-configs", StageIgnitionConfigs},
		{"--cluster", StageCluster},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			opts, err := parseOptions([]string{tt.flag})
			if err != nil {
				t.Fatal(err)
			}
			if opts.stage != tt.want {
				t.Fatalf("stage = %s, want %s", opts.stage, tt.want)
			}
		})
	}
}

func TestParseMultipleStageFlags(t *testing.T) {
	_, err := parseOptions([]string{"--manifests", "--cluster"})
	if err == nil {
		t.Fatal("expected an error for multiple stage flags")
	}
}

func TestParseIgnitionAliasesTogether(t *testing.T) {
	opts, err := parseOptions([]string{"--ignition", "--ignition-configs"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.stage != StageIgnitionConfigs {
		t.Fatalf("stage = %s, want ignition-configs", opts.stage)
	}
}

func TestParseOptionsHelp(t *testing.T) {
	_, err := parseOptions([]string{"--help"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", err)
	}
}

func TestParseOptionsInvalidLogLevel(t *testing.T) {
	_, err := parseOptions([]string{"--log-level", "nonsense"})
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestParseOptionsUnknownFlag(t *testing.T) {
	_, err := parseOptions([]string{"--nope"})
	if err == nil {
		t.Fatal("expected an error")
	}
}

func assertArgs(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
