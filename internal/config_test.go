package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSettingsUsesConfigDirectoryAsRoot(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "clash.conf")
	writeFile(t, configPath, []byte(`
subscription:
  url: "file:///tmp/sub.yaml"
mihomo:
  secret: "test-secret"
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
	if mapValue(settings.Mihomo, "tun") == nil {
		t.Fatal("expected arbitrary mihomo.tun config to be preserved")
	}
	if mapValue(settings.Mihomo, "external-ui") == nil {
		t.Fatal("expected external-ui default")
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
  url: "file:///tmp/sub.yaml"
`))

	settings, err := LoadSettings("")
	if err != nil {
		t.Fatal(err)
	}
	if settings.RootDir != dir {
		t.Fatalf("RootDir = %q, want %q", settings.RootDir, dir)
	}
	if got := scalarString(mapValue(settings.Mihomo, "external-ui")); got != filepath.Join(dir, "ui") {
		t.Fatalf("external-ui = %q, want %q", got, filepath.Join(dir, "ui"))
	}
}

func TestUIDirUsesExplicitExternalUI(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "clash.conf")
	writeFile(t, configPath, []byte(`
subscription:
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

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0640); err != nil {
		t.Fatal(err)
	}
}
