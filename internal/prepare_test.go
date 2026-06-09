package internal

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestPrepareUsesExistingConfigWhenFetchFails(t *testing.T) {
	settings := testSettings(t)
	writeFile(t, settings.ConfigYAMLPath(), []byte(validProfileYAML))

	err := PrepareSettings(context.Background(), settings, PrepareOptions{
		Fetch: func(context.Context, *Settings) ([]byte, error) {
			return nil, errors.New("offline")
		},
		Log: func(string, ...any) {},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPrepareFailsWhenFetchFailsWithoutExistingConfig(t *testing.T) {
	settings := testSettings(t)

	err := PrepareSettings(context.Background(), settings, PrepareOptions{
		Fetch: func(context.Context, *Settings) ([]byte, error) {
			return nil, errors.New("offline")
		},
		Log: func(string, ...any) {},
	})
	if err == nil {
		t.Fatal("expected prepare to fail without an existing config")
	}
}

func TestPrepareWritesGeneratedConfig(t *testing.T) {
	settings := testSettings(t)

	err := PrepareSettings(context.Background(), settings, PrepareOptions{
		Fetch: func(context.Context, *Settings) ([]byte, error) {
			return []byte(validProfileYAML), nil
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

func testSettings(t *testing.T) *Settings {
	t.Helper()
	dir := t.TempDir()
	settings := &Settings{
		ConfigPath:      filepath.Join(dir, "clash.conf"),
		RootDir:         dir,
		SubscriptionURL: "https://example.invalid/sub",
		Timeout:         defaultTimeout,
		UserAgent:       "clashtars-test",
		Mihomo:          newMappingNode(),
	}
	applyMihomoDefaults(settings)
	return settings
}
