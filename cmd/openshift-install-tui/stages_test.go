package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestStagesFrom(t *testing.T) {
	tests := []struct {
		name    string
		current Stage
		target  Stage
		want    []Stage
	}{
		{"new to manifests", StageNone, StageManifests, []Stage{StageInstallConfig, StageManifests}},
		{"install-config to cluster", StageInstallConfig, StageCluster, []Stage{StageManifests, StageIgnitionConfigs, StageCluster}},
		{"rerun target", StageManifests, StageManifests, []Stage{StageManifests}},
		{"rerun earlier", StageCluster, StageManifests, []Stage{StageManifests}},
		{"none target", StageInstallConfig, StageNone, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertArgs(t, stageStrings(stagesFrom(tt.current, tt.target)), stageStrings(tt.want))
		})
	}
}

func stageStrings(stages []Stage) []string {
	out := make([]string, len(stages))
	for i, stage := range stages {
		out[i] = stage.String()
	}
	return out
}

func TestDetectEmptyDirectory(t *testing.T) {
	d, err := Detect(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !d.New {
		t.Fatalf("expected New=true, got %+v", d)
	}
	if d.Last != StageNone {
		t.Fatalf("expected StageNone, got %s", d.Last)
	}
}

func TestDetectMissingDirectory(t *testing.T) {
	d, err := Detect(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatal(err)
	}
	if !d.New {
		t.Fatalf("expected New=true, got %+v", d)
	}
}

func TestDetectIgnoresUnrelatedFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "notes.txt", "hello")

	d, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !d.New || d.Last != StageNone {
		t.Fatalf("expected new empty installer state, got %+v", d)
	}
}

func TestDetectInstallConfigFromDisk(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, installConfigName, "apiVersion: v1\n")

	d, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if d.New {
		t.Fatal("expected non-new directory")
	}
	if d.Last != StageInstallConfig {
		t.Fatalf("expected install-config, got %s", d.Last)
	}
	if d.Next() != StageManifests {
		t.Fatalf("expected next manifests, got %s", d.Next())
	}
}

func TestDetectFromStateFile(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want Stage
		next Stage
	}{
		{"install-config", "*installconfig.InstallConfig", StageInstallConfig, StageManifests},
		{"manifests", "*manifests.Manifests", StageManifests, StageIgnitionConfigs},
		{"ignition", "*bootstrap.Bootstrap", StageIgnitionConfigs, StageCluster},
		{"cluster", "*cluster.Cluster", StageCluster, StageNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, stateFileName, `{"`+tt.key+`": {}}`)

			d, err := Detect(dir)
			if err != nil {
				t.Fatal(err)
			}
			if d.Last != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, d.Last)
			}
			if d.Next() != tt.next {
				t.Fatalf("expected next %s, got %s", tt.next, d.Next())
			}
		})
	}
}

func TestDetectPrefersDiskOverStaleState(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, stateFileName, `{"*bootstrap.Bootstrap": {}}`)
	writeFile(t, dir, "manifests/cluster.yaml", "x")

	d, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if d.Last != StageManifests {
		t.Fatalf("expected manifests from disk, got %s", d.Last)
	}
	if d.Next() != StageIgnitionConfigs {
		t.Fatalf("next = %s, want ignition-configs", d.Next())
	}
}

func TestDetectFromDiskArtifacts(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  Stage
	}{
		{"manifests dir", []string{"manifests/cluster.yaml"}, StageManifests},
		{"openshift dir", []string{"openshift/99_test.yaml"}, StageManifests},
		{"ignition", []string{"bootstrap.ign"}, StageIgnitionConfigs},
		{"auth kubeconfig", []string{"auth/kubeconfig"}, StageIgnitionConfigs},
		{"metadata", []string{metadataName}, StageIgnitionConfigs},
		{"terraform", []string{"terraform.platform.auto.tfvars.json"}, StageCluster},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.files {
				writeFile(t, dir, f, "x")
			}
			d, err := Detect(dir)
			if err != nil {
				t.Fatal(err)
			}
			if d.Last != tt.want {
				t.Fatalf("expected %s, got %s (evidence %v)", tt.want, d.Last, d.Evidence)
			}
		})
	}
}

func TestDetectUsesLogWhenNothingElse(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, logFileName,
		`time="2026-01-01" level=debug msg="Running: openshift-install create manifests --dir ."`+"\n")

	d, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if d.New {
		t.Fatal("expected non-new directory")
	}
	if d.LastRun != StageManifests {
		t.Fatalf("expected LastRun manifests, got %s", d.LastRun)
	}
	// Attempted stage may not have completed, so detection steps back one.
	if d.Last != StageInstallConfig {
		t.Fatalf("expected Last install-config, got %s", d.Last)
	}
	if d.Next() != StageManifests {
		t.Fatalf("expected next manifests, got %s", d.Next())
	}
}

func TestDetectClusterErrors(t *testing.T) {
	dir := t.TempDir()
	log := `time="t" level=error msg="old error before cluster run"` + "\n" +
		`time="t" level=debug msg="Running: openshift-install create cluster --dir ."` + "\n" +
		`time="t" level=info msg="Creating infrastructure resources..."` + "\n" +
		`time="t" level=error msg="failed to create cluster: boom"` + "\n" +
		`time="t" level=fatal msg="failed to create cluster: boom"` + "\n"
	writeFile(t, dir, logFileName, log)

	d, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !d.LastClusterRun {
		t.Fatal("expected cluster run to be detected")
	}
	if len(d.Errors) != 2 {
		t.Fatalf("expected 2 errors after cluster run, got %d: %+v", len(d.Errors), d.Errors)
	}
}

func TestFindLogBundle(t *testing.T) {
	dir := t.TempDir()
	if got := findLogBundle(dir); got != "" {
		t.Fatalf("expected no bundle, got %q", got)
	}
	writeFile(t, dir, "log-bundle-20260101.tar.gz", "old")
	writeFile(t, dir, "log-bundle-20260202.tar.gz", "new")

	got := findLogBundle(dir)
	if filepath.Base(got) != "log-bundle-20260202.tar.gz" {
		t.Fatalf("expected newest bundle, got %q", got)
	}
}

func TestStageArgs(t *testing.T) {
	got := StageManifests.CreateArgs("/tmp/assets")
	want := []string{"create", "manifests", "--dir", "/tmp/assets"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}
