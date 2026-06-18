package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const providerPlaceholder = "__PROVIDER_PLACEHOLDER__"

func SynthesizeConfig(settings *Settings, template []byte) ([]byte, error) {
	templateRoot, err := parseYAMLMapping(template)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	finalRoot := cloneNode(templateRoot)
	overlayMapping(finalRoot, settings.Mihomo)

	providers, err := buildProxyProviders(settings)
	if err != nil {
		return nil, err
	}
	setMapValue(finalRoot, "proxy-providers", providers)
	expandProviderPlaceholder(finalRoot, settings.ProviderNames())

	if err := validateGeneratedConfig(finalRoot); err != nil {
		return nil, fmt.Errorf("generated config: %w", err)
	}

	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{finalRoot}}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func overlayMapping(dst *yaml.Node, src *yaml.Node) {
	if dst == nil || dst.Kind != yaml.MappingNode || src == nil || src.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(src.Content); i += 2 {
		setMapValue(dst, src.Content[i].Value, cloneNode(src.Content[i+1]))
	}
}

func ExtractProviderProfile(profile []byte) ([]byte, error) {
	root, err := parseYAMLMapping(profile)
	if err != nil {
		return nil, err
	}
	if err := validateProviderRoot(root); err != nil {
		return nil, err
	}

	outRoot := newMappingNode()
	setMapValue(outRoot, "proxies", cloneNode(mapValue(root, "proxies")))
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{outRoot}}
	return yaml.Marshal(doc)
}

func (s *Settings) ProviderNames() []string {
	names := make([]string, len(s.Providers))
	for i, provider := range s.Providers {
		names[i] = provider.Name
	}
	return names
}

func buildProxyProviders(settings *Settings) (*yaml.Node, error) {
	out := newMappingNode()
	for _, provider := range settings.Providers {
		if !providerFileUsable(settings.ProviderYAMLPath(provider.Name)) {
			return nil, fmt.Errorf("provider %q has no usable local file at %s", provider.Name, settings.ProviderYAMLPath(provider.Name))
		}

		node := newMappingNode()
		setMapValue(node, "type", scalarStr("file"))
		setMapValue(node, "path", scalarStr("./"+filepath.ToSlash(filepath.Join("providers", provider.Name+".yaml"))))
		setMapValue(node, "health-check", defaultHealthCheck())
		if provider.Prefix != "" {
			override := newMappingNode()
			setMapValue(override, "additional-prefix", scalarStr(provider.Prefix))
			setMapValue(node, "override", override)
		}
		setMapValue(out, provider.Name, node)
	}
	return out, nil
}

func defaultHealthCheck() *yaml.Node {
	node := newMappingNode()
	setMapValue(node, "enable", scalarBool(true))
	setMapValue(node, "url", scalarStr("https://www.gstatic.com/generate_204"))
	setMapValue(node, "interval", scalarInt(300))
	return node
}

func expandProviderPlaceholder(root *yaml.Node, names []string) {
	groups := mapValue(root, "proxy-groups")
	if groups == nil || groups.Kind != yaml.SequenceNode {
		return
	}
	for _, group := range groups.Content {
		if group.Kind != yaml.MappingNode {
			continue
		}
		use := mapValue(group, "use")
		if use == nil || use.Kind != yaml.SequenceNode {
			continue
		}
		var content []*yaml.Node
		for _, item := range use.Content {
			if item.Kind == yaml.ScalarNode && item.Value == providerPlaceholder {
				for _, name := range names {
					content = append(content, scalarStr(name))
				}
				continue
			}
			content = append(content, item)
		}
		use.Content = content
	}
}

func providerFileUsable(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return false
	}
	root, err := parseYAMLMapping(data)
	if err != nil {
		return false
	}
	return validateProviderRoot(root) == nil
}

func isClashProfile(data []byte) bool {
	root, err := parseYAMLMapping(data)
	if err != nil {
		return false
	}
	return validateProviderRoot(root) == nil
}

func isGeneratedConfig(data []byte) bool {
	root, err := parseYAMLMapping(data)
	if err != nil {
		return false
	}
	return validateGeneratedConfig(root) == nil
}

func validateProviderRoot(root *yaml.Node) error {
	proxies := mapValue(root, "proxies")
	if proxies == nil {
		return fmt.Errorf("missing required section %q", "proxies")
	}
	if proxies.Kind != yaml.SequenceNode || len(proxies.Content) == 0 {
		return fmt.Errorf("proxies must be a non-empty sequence")
	}
	return nil
}

func validateGeneratedConfig(root *yaml.Node) error {
	providers := mapValue(root, "proxy-providers")
	if providers == nil || providers.Kind != yaml.MappingNode || len(providers.Content) == 0 {
		return fmt.Errorf("proxy-providers must be a non-empty mapping")
	}
	groups := mapValue(root, "proxy-groups")
	if groups == nil || groups.Kind != yaml.SequenceNode || len(groups.Content) == 0 {
		return fmt.Errorf("proxy-groups must be a non-empty sequence")
	}
	rules := mapValue(root, "rules")
	if rules == nil || rules.Kind != yaml.SequenceNode || len(rules.Content) == 0 {
		return fmt.Errorf("rules must be a non-empty sequence")
	}

	providerNames := map[string]bool{}
	for i := 0; i+1 < len(providers.Content); i += 2 {
		providerNames[providers.Content[i].Value] = true
	}
	for _, group := range groups.Content {
		if group.Kind != yaml.MappingNode {
			continue
		}
		use := mapValue(group, "use")
		if use == nil {
			continue
		}
		if use.Kind != yaml.SequenceNode {
			return fmt.Errorf("proxy-groups[].use must be a sequence")
		}
		for _, item := range use.Content {
			if item.Kind != yaml.ScalarNode {
				return fmt.Errorf("proxy-groups[].use entries must be scalars")
			}
			if !providerNames[item.Value] {
				return fmt.Errorf("proxy group references unknown provider %q", item.Value)
			}
		}
	}
	return nil
}
