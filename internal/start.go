package internal

import "fmt"

func Start(configPath string) error {
	settings, err := LoadSettings(configPath)
	if err != nil {
		return err
	}
	if !existingConfigUsable(settings.ConfigYAMLPath()) {
		return fmt.Errorf("no usable generated config at %s; run prepare first", settings.ConfigYAMLPath())
	}

	if err := ExtractEmbeddedUI(settings.UIDir()); err != nil {
		return err
	}

	core, err := EmbeddedMihomo()
	if err != nil {
		return err
	}
	return ExecMemfd("mihomo", core, []string{"mihomo", "-d", settings.RootDir})
}
