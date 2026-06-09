package internal

import (
	"context"
	"fmt"
	"os"
)

type PrepareOptions struct {
	Fetch FetchFunc
	Log   func(string, ...any)
}

func Prepare(ctx context.Context, configPath string) error {
	settings, err := LoadSettings(configPath)
	if err != nil {
		return err
	}
	return PrepareSettings(ctx, settings, PrepareOptions{})
}

func PrepareSettings(ctx context.Context, settings *Settings, opts PrepareOptions) error {
	if opts.Fetch == nil {
		opts.Fetch = FetchSubscription
	}
	if opts.Log == nil {
		opts.Log = func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, format+"\n", args...)
		}
	}

	if err := ensureRuntimeDirs(settings); err != nil {
		return err
	}

	raw, err := opts.Fetch(ctx, settings)
	if err != nil {
		return useExistingConfig(settings, opts.Log, fmt.Sprintf("subscription refresh failed: %v", err))
	}
	if err := atomicWriteFile(settings.SubscriptionPath(), raw, 0640); err != nil {
		return useExistingConfig(settings, opts.Log, fmt.Sprintf("writing subscription failed: %v", err))
	}

	converted, err := ConvertSubscription(ctx, settings, raw)
	if err != nil {
		return useExistingConfig(settings, opts.Log, fmt.Sprintf("subscription conversion failed: %v", err))
	}
	if err := atomicWriteFile(settings.ConvertedPath(), converted, 0640); err != nil {
		return useExistingConfig(settings, opts.Log, fmt.Sprintf("writing converted profile failed: %v", err))
	}

	finalConfig, err := SynthesizeConfig(settings, converted)
	if err != nil {
		return useExistingConfig(settings, opts.Log, fmt.Sprintf("config synthesis failed: %v", err))
	}
	if err := atomicWriteFile(settings.ConfigYAMLPath(), finalConfig, 0640); err != nil {
		return useExistingConfig(settings, opts.Log, fmt.Sprintf("writing generated config failed: %v", err))
	}

	opts.Log("prepared config: %s", settings.ConfigYAMLPath())
	return nil
}

func ensureRuntimeDirs(settings *Settings) error {
	for _, dir := range []string{
		settings.RootDir,
		settings.CacheDir(),
		settings.UIDir(),
	} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return err
		}
	}
	return nil
}

func useExistingConfig(settings *Settings, log func(string, ...any), reason string) error {
	if existingConfigUsable(settings.ConfigYAMLPath()) {
		log("warning: %s; using existing config: %s", reason, settings.ConfigYAMLPath())
		return nil
	}
	return fmt.Errorf("%s; no usable existing config at %s", reason, settings.ConfigYAMLPath())
}

func existingConfigUsable(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return false
	}
	return isClashProfile(data)
}
