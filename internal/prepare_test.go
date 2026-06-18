package internal

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestPrepareUsesExistingConfigWhenFetchFails(t *testing.T) {
	settings, templatePath := testPrepareSettings(t)
	writeFile(t, settings.ConfigYAMLPath(), []byte(validGeneratedConfigYAML))

	err := PrepareSettings(context.Background(), settings, templatePath, PrepareOptions{
		Fetch: func(context.Context, *Settings, Provider) ([]byte, error) {
			return nil, errors.New("offline")
		},
		Log: func(string, ...any) {},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPrepareUsesExistingProviderWhenFetchFails(t *testing.T) {
	settings, templatePath := testPrepareSettings(t)
	writeFile(t, settings.ProviderYAMLPath("main"), []byte(validProviderYAML))

	err := PrepareSettings(context.Background(), settings, templatePath, PrepareOptions{
		Fetch: func(context.Context, *Settings, Provider) ([]byte, error) {
			return nil, errors.New("offline")
		},
		Log: func(string, ...any) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !existingConfigUsable(settings.ConfigYAMLPath()) {
		t.Fatalf("generated config is not usable: %s", settings.ConfigYAMLPath())
	}
}

func TestPrepareFailsWhenFetchFailsWithoutExistingConfig(t *testing.T) {
	settings, templatePath := testPrepareSettings(t)

	err := PrepareSettings(context.Background(), settings, templatePath, PrepareOptions{
		Fetch: func(context.Context, *Settings, Provider) ([]byte, error) {
			return nil, errors.New("offline")
		},
		Log: func(string, ...any) {},
	})
	if err == nil {
		t.Fatal("expected prepare to fail without an existing config")
	}
}

func TestPrepareWritesGeneratedConfig(t *testing.T) {
	settings, templatePath := testPrepareSettings(t)

	err := PrepareSettings(context.Background(), settings, templatePath, PrepareOptions{
		Fetch: func(context.Context, *Settings, Provider) ([]byte, error) {
			return []byte(validProfileYAML), nil
		},
		Log: func(string, ...any) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !providerFileUsable(settings.ProviderYAMLPath("main")) {
		t.Fatalf("provider file is not usable: %s", settings.ProviderYAMLPath("main"))
	}
	if !existingConfigUsable(settings.ConfigYAMLPath()) {
		t.Fatalf("generated config is not usable: %s", settings.ConfigYAMLPath())
	}
}

func testPrepareSettings(t *testing.T) (*Settings, string) {
	t.Helper()
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "template.yaml")
	writeFile(t, templatePath, []byte(validTemplateYAML))
	settings := &Settings{
		ConfigPath: filepath.Join(dir, "clash.conf"),
		RootDir:    dir,
		Timeout:    defaultTimeout,
		UserAgent:  "clashtars-test",
		Mihomo:     testMihomo(dir),
		Providers: []Provider{
			{
				Name:   "main",
				URL:    "https://example.invalid/sub",
				Prefix: "[main] ",
			},
		},
	}
	return settings, templatePath
}

const validGeneratedConfigYAML = `
proxy-providers:
  main:
    type: file
    path: ./providers/main.yaml
proxy-groups:
  - name: Proxies
    type: select
    use:
      - main
rules:
  - MATCH,Proxies
`
