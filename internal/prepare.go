package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type PrepareOptions struct {
	Fetch FetchFunc
	Log   func(string, ...any)
}

func Prepare(ctx context.Context, configPath string, templatePath string) error {
	if templatePath == "" {
		templatePath = defaultTemplateName
	}
	absTemplatePath, err := filepath.Abs(templatePath)
	if err != nil {
		return err
	}
	settings, err := LoadSettings(configPath)
	if err != nil {
		return err
	}
	return PrepareSettings(ctx, settings, absTemplatePath, PrepareOptions{})
}

func PrepareSettings(ctx context.Context, settings *Settings, templatePath string, opts PrepareOptions) error {
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

	for _, provider := range settings.Providers {
		if err := refreshProvider(ctx, settings, provider, opts); err != nil {
			opts.Log("warning: provider %q refresh failed: %v; using existing provider file if available", provider.Name, err)
		}
	}

	template, err := os.ReadFile(templatePath)
	if err != nil {
		return useExistingConfig(settings, opts.Log, fmt.Sprintf("reading template failed: %v", err))
	}

	finalConfig, err := SynthesizeConfig(settings, template)
	if err != nil {
		return useExistingConfig(settings, opts.Log, fmt.Sprintf("config synthesis failed: %v", err))
	}
	if err := atomicWriteFile(settings.ConfigYAMLPath(), finalConfig, 0640); err != nil {
		return useExistingConfig(settings, opts.Log, fmt.Sprintf("writing generated config failed: %v", err))
	}

	opts.Log("prepared config: %s", settings.ConfigYAMLPath())
	return nil
}

func refreshProvider(ctx context.Context, settings *Settings, provider Provider, opts PrepareOptions) error {
	raw, err := opts.Fetch(ctx, settings, provider)
	if err != nil {
		return err
	}
	if err := atomicWriteFile(settings.ProviderRawPath(provider.Name), raw, 0640); err != nil {
		return fmt.Errorf("writing raw subscription: %w", err)
	}

	converted, err := ConvertSubscription(ctx, settings, provider, raw)
	if err != nil {
		return err
	}
	if err := atomicWriteFile(settings.ProviderConvertedPath(provider.Name), converted, 0640); err != nil {
		return fmt.Errorf("writing converted profile: %w", err)
	}

	providerProfile, err := ExtractProviderProfile(converted)
	if err != nil {
		return err
	}
	if err := atomicWriteFile(settings.ProviderYAMLPath(provider.Name), providerProfile, 0640); err != nil {
		return fmt.Errorf("writing provider profile: %w", err)
	}
	return nil
}

func ensureRuntimeDirs(settings *Settings) error {
	for _, dir := range []string{
		settings.RootDir,
		settings.CacheDir(),
		settings.UIDir(),
		settings.ProvidersDir(),
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
	return isGeneratedConfig(data)
}
