package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSettingsUsesWorkingDirectoryAsRoot(t *testing.T) {
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	configPath := filepath.Join(dir, "clash.conf")
	writeFile(t, configPath, []byte(`
subscription:
  timeout: "10s"
  user-agent: "test-agent"
  providers:
    - name: main
      url: "file:///tmp/sub.yaml"
    - name: backup
      url: "https://example.invalid/sub"
      prefix: "[backup] "
    - name: noprefix
      url: "https://example.invalid/noprefix"
      prefix: ""
mihomo:
  external-ui: "./dashboard"
  external-controller: "127.0.0.1:9099"
  tun:
    enable: true
`))

	settings, err := LoadSettings(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if settings.RootDir != dir {
		t.Fatalf("RootDir = %q, want %q", settings.RootDir, dir)
	}
	if settings.ConfigYAMLPath() != filepath.Join(dir, "config.yaml") {
		t.Fatalf("ConfigYAMLPath = %q", settings.ConfigYAMLPath())
	}
	if settings.Timeout.String() != "10s" {
		t.Fatalf("Timeout = %s", settings.Timeout)
	}
	if settings.UserAgent != "test-agent" {
		t.Fatalf("UserAgent = %q", settings.UserAgent)
	}
	if len(settings.Providers) != 3 {
		t.Fatalf("Providers len = %d", len(settings.Providers))
	}
	if settings.Providers[0].Prefix != "[main] " {
		t.Fatalf("default prefix = %q", settings.Providers[0].Prefix)
	}
	if settings.Providers[1].Prefix != "[backup] " {
		t.Fatalf("explicit prefix = %q", settings.Providers[1].Prefix)
	}
	if settings.Providers[2].Prefix != "" {
		t.Fatalf("explicit empty prefix = %q", settings.Providers[2].Prefix)
	}
	if settings.UIDir() != "./dashboard" {
		t.Fatalf("UIDir = %q, want ./dashboard", settings.UIDir())
	}
	if mapValue(settings.Mihomo, "tun") == nil {
		t.Fatal("expected arbitrary mihomo.tun config to be preserved")
	}
}

func TestLoadSettingsEmptyPathUsesCurrentDirectoryConfig(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	writeFile(t, filepath.Join(dir, "clash.conf"), []byte(`
subscription:
  providers:
    - name: main
      url: "file:///tmp/sub.yaml"
`))

	settings, err := LoadSettings("")
	if err != nil {
		t.Fatal(err)
	}
	if settings.RootDir != dir {
		t.Fatalf("RootDir = %q, want %q", settings.RootDir, dir)
	}
	if got := settings.UIDir(); got != filepath.Join(dir, "ui") {
		t.Fatalf("UIDir = %q, want %q", got, filepath.Join(dir, "ui"))
	}
}

func TestUIDirUsesMihomoExternalUI(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "clash.conf")
	writeFile(t, configPath, []byte(`
subscription:
  providers:
    - name: main
      url: "file:///tmp/sub.yaml"
mihomo:
  external-ui: "/tmp/custom-ui"
`))

	settings, err := LoadSettings(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := settings.UIDir(); got != "/tmp/custom-ui" {
		t.Fatalf("UIDir = %q, want /tmp/custom-ui", got)
	}
}

func TestLoadSettingsRejectsMissingProviders(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "clash.conf")
	writeFile(t, configPath, []byte(`
subscription:
  timeout: "30s"
`))

	if _, err := LoadSettings(configPath); err == nil {
		t.Fatal("expected missing providers to fail")
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0640); err != nil {
		t.Fatal(err)
	}
}
