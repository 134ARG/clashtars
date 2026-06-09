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
	defaultConfigName = "clash.conf"
	defaultTimeout    = 30 * time.Second
)

type Settings struct {
	ConfigPath string
	RootDir    string

	SubscriptionURL string
	Timeout         time.Duration
	UserAgent       string

	Mihomo *yaml.Node
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
		RootDir:    filepath.Dir(absPath),
		Timeout:    defaultTimeout,
		UserAgent:  "clashtars/1.0",
		Mihomo:     newMappingNode(),
	}

	if runtime := mapValue(root, "runtime"); runtime != nil {
		if runtime.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("runtime must be a mapping")
		}
		if rootDir := scalarString(mapValue(runtime, "root-dir")); rootDir != "" {
			settings.RootDir = rootDir
		}
	}

	if sub := mapValue(root, "subscription"); sub != nil {
		if sub.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("subscription must be a mapping")
		}
		settings.SubscriptionURL = scalarString(mapValue(sub, "url"))
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
	}

	if mihomo := mapValue(root, "mihomo"); mihomo != nil {
		if mihomo.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("mihomo must be a mapping")
		}
		settings.Mihomo = cloneNode(mihomo)
	}
	applyMihomoDefaults(settings)

	return settings, nil
}

func (s *Settings) ConfigYAMLPath() string {
	return filepath.Join(s.RootDir, "config.yaml")
}

func (s *Settings) SubscriptionPath() string {
	return filepath.Join(s.RootDir, "subscription.yaml")
}

func (s *Settings) ConvertedPath() string {
	return filepath.Join(s.RootDir, "converted.yaml")
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

func applyMihomoDefaults(s *Settings) {
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
		{"external-ui", scalarStr(s.UIDir())},
		{"secret", scalarStr("")},
	}
	for _, item := range defaults {
		if mapValue(s.Mihomo, item.key) == nil {
			setMapValue(s.Mihomo, item.key, item.val)
		}
	}
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
