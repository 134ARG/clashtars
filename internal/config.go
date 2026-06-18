package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultConfigName   = "clash.conf"
	defaultTemplateName = "template.yaml"
	defaultTimeout      = 30 * time.Second
)

type Settings struct {
	ConfigPath string
	RootDir    string

	Timeout   time.Duration
	UserAgent string

	Mihomo    *yaml.Node
	Providers []Provider
}

type Provider struct {
	Name   string
	URL    string
	Prefix string
}

func LoadSettings(configPath string) (*Settings, error) {
	if configPath == "" {
		configPath = defaultConfigName
	}
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	root, err := parseYAMLMapping(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", absPath, err)
	}

	settings := &Settings{
		ConfigPath: absPath,
		RootDir:    mustGetwd(),
		Timeout:    defaultTimeout,
		UserAgent:  "clashtars/1.0",
		Mihomo:     newMappingNode(),
	}

	if sub := mapValue(root, "subscription"); sub != nil {
		if sub.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("subscription must be a mapping")
		}
		if timeout := scalarString(mapValue(sub, "timeout")); timeout != "" {
			parsed, err := time.ParseDuration(timeout)
			if err != nil {
				return nil, fmt.Errorf("subscription.timeout: %w", err)
			}
			settings.Timeout = parsed
		}
		if ua := scalarString(mapValue(sub, "user-agent")); ua != "" {
			settings.UserAgent = ua
		}
		providers, err := parseProviders(mapValue(sub, "providers"))
		if err != nil {
			return nil, err
		}
		settings.Providers = providers
	}

	if mihomo := mapValue(root, "mihomo"); mihomo != nil {
		if mihomo.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("mihomo must be a mapping")
		}
		settings.Mihomo = cloneNode(mihomo)
	}
	applyMihomoDefaults(settings.Mihomo, settings.RootDir)

	if len(settings.Providers) == 0 {
		return nil, fmt.Errorf("subscription.providers must contain at least one provider")
	}

	return settings, nil
}

func (s *Settings) ConfigYAMLPath() string {
	return filepath.Join(s.RootDir, "config.yaml")
}

func (s *Settings) ProvidersDir() string {
	return filepath.Join(s.RootDir, "providers")
}

func (s *Settings) ProviderRawPath(name string) string {
	return filepath.Join(s.ProvidersDir(), name+".raw")
}

func (s *Settings) ProviderConvertedPath(name string) string {
	return filepath.Join(s.ProvidersDir(), name+".converted.yaml")
}

func (s *Settings) ProviderYAMLPath(name string) string {
	return filepath.Join(s.ProvidersDir(), name+".yaml")
}

func (s *Settings) CacheDir() string {
	return filepath.Join(s.RootDir, "cache")
}

func (s *Settings) UIDir() string {
	if uiDir := scalarString(mapValue(s.Mihomo, "external-ui")); uiDir != "" {
		return uiDir
	}
	return filepath.Join(s.RootDir, "ui")
}

func (s *Settings) SubconverterDir() string {
	return filepath.Join(s.CacheDir(), "subconverter")
}

func applyMihomoDefaults(root *yaml.Node, rootDir string) {
	defaults := []struct {
		key string
		val *yaml.Node
	}{
		{"port", scalarInt(7890)},
		{"socks-port", scalarInt(7891)},
		{"redir-port", scalarInt(7892)},
		{"allow-lan", scalarBool(true)},
		{"mode", scalarStr("rule")},
		{"log-level", scalarStr("silent")},
		{"external-controller", scalarStr("0.0.0.0:9091")},
		{"external-ui", scalarStr(filepath.Join(rootDir, "ui"))},
		{"secret", scalarStr("")},
	}
	for _, item := range defaults {
		if mapValue(root, item.key) == nil {
			setMapValue(root, item.key, item.val)
		}
	}
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func parseProviders(node *yaml.Node) ([]Provider, error) {
	if node == nil {
		return nil, fmt.Errorf("subscription.providers is required")
	}
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("subscription.providers must be a sequence")
	}

	seen := map[string]bool{}
	var providers []Provider
	for i, item := range node.Content {
		if item.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("subscription.providers[%d] must be a mapping", i)
		}
		provider := Provider{
			Name: scalarString(mapValue(item, "name")),
			URL:  scalarString(mapValue(item, "url")),
		}
		if provider.Name == "" {
			return nil, fmt.Errorf("subscription.providers[%d].name is required", i)
		}
		if !validProviderName(provider.Name) {
			return nil, fmt.Errorf("subscription.providers[%d].name %q must use only letters, digits, '.', '_' or '-'", i, provider.Name)
		}
		if seen[provider.Name] {
			return nil, fmt.Errorf("duplicate provider name %q", provider.Name)
		}
		seen[provider.Name] = true
		if provider.URL == "" {
			return nil, fmt.Errorf("subscription.providers[%d].url is required", i)
		}
		if prefix := mapValue(item, "prefix"); prefix != nil {
			provider.Prefix = scalarRawString(prefix)
		} else {
			provider.Prefix = fmt.Sprintf("[%s] ", provider.Name)
		}
		providers = append(providers, provider)
	}
	return providers, nil
}

func validProviderName(name string) bool {
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			continue
		}
		if r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return name != "." && name != ".."
}

func parseYAMLMapping(data []byte) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("empty YAML document")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("top-level YAML must be a mapping")
	}
	return root, nil
}

func newMappingNode() *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
}

func mapValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

func setMapValue(mapping *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1] = value
			return
		}
	}
	mapping.Content = append(mapping.Content, scalarStr(key), value)
}

func scalarString(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	return strings.TrimSpace(node.Value)
}

func scalarRawString(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	return node.Value
}

func scalarStr(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

func scalarInt(value int) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(value)}
}

func scalarBool(value bool) *yaml.Node {
	if value {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"}
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "false"}
}

func cloneNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	out := *node
	if len(node.Content) > 0 {
		out.Content = make([]*yaml.Node, len(node.Content))
		for i, child := range node.Content {
			out.Content[i] = cloneNode(child)
		}
	}
	return &out
}
