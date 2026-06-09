package internal

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

var subscriptionSections = []string{
	"proxies",
	"proxy-groups",
	"rules",
	"proxy-providers",
	"rule-providers",
}

var requiredSections = []string{
	"proxies",
	"proxy-groups",
	"rules",
}

func SynthesizeConfig(settings *Settings, profile []byte) ([]byte, error) {
	profileRoot, err := parseYAMLMapping(profile)
	if err != nil {
		return nil, fmt.Errorf("parse converted profile: %w", err)
	}
	if err := validateRequiredSections(profileRoot); err != nil {
		return nil, fmt.Errorf("converted profile: %w", err)
	}

	finalRoot := cloneNode(settings.Mihomo)
	for _, section := range subscriptionSections {
		if value := mapValue(profileRoot, section); value != nil {
			setMapValue(finalRoot, section, cloneNode(value))
		}
	}
	if err := validateRequiredSections(finalRoot); err != nil {
		return nil, fmt.Errorf("generated config: %w", err)
	}

	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{finalRoot}}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func isClashProfile(data []byte) bool {
	root, err := parseYAMLMapping(data)
	if err != nil {
		return false
	}
	return validateRequiredSections(root) == nil
}

func validateRequiredSections(root *yaml.Node) error {
	for _, key := range requiredSections {
		if mapValue(root, key) == nil {
			return fmt.Errorf("missing required section %q", key)
		}
	}
	return nil
}
