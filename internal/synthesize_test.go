package internal

import (
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSynthesizeConfigInjectsProxyProviders(t *testing.T) {
	settings := testSynthesizeSettings(t)
	writeFile(t, settings.ProviderYAMLPath("main"), []byte(validProviderYAML))
	writeFile(t, settings.ProviderYAMLPath("backup"), []byte(validProviderYAML))

	out, err := SynthesizeConfig(settings, []byte(validTemplateYAML))
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, want := range []string{
		"proxy-providers:",
		"main:",
		"backup:",
		"path: ./providers/main.yaml",
		"additional-prefix: '[main] '",
		"external-controller: 127.0.0.1:9099",
		"external-ui: " + filepath.Join(settings.RootDir, "ui"),
		"proxy-groups:",
		"- main",
		"- backup",
		"rules:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated config missing %q:\n%s", want, text)
		}
	}
}

func testSynthesizeSettings(t *testing.T) *Settings {
	t.Helper()
	dir := t.TempDir()
	return &Settings{
		ConfigPath: filepath.Join(dir, "clash.conf"),
		RootDir:    dir,
		Timeout:    defaultTimeout,
		UserAgent:  "clashtars-test",
		Mihomo:     testMihomo(dir),
		Providers: []Provider{
			{Name: "main", URL: "https://example.invalid/main", Prefix: "[main] "},
			{Name: "backup", URL: "https://example.invalid/backup", Prefix: "[backup] "},
		},
	}
}

func testMihomo(rootDir string) *yaml.Node {
	node := newMappingNode()
	setMapValue(node, "external-controller", scalarStr("127.0.0.1:9099"))
	applyMihomoDefaults(node, rootDir)
	return node
}

func TestExtractProviderProfileKeepsOnlyProxies(t *testing.T) {
	out, err := ExtractProviderProfile([]byte(validProfileYAML))
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, "proxies:") {
		t.Fatalf("provider profile missing proxies:\n%s", text)
	}
	if strings.Contains(text, "proxy-groups:") || strings.Contains(text, "rules:") {
		t.Fatalf("provider profile kept non-provider sections:\n%s", text)
	}
}

const validTemplateYAML = `
external-controller: 0.0.0.0:9090
proxy-groups:
  - name: Proxies
    type: select
    proxies:
      - DIRECT
    use:
      - __PROVIDER_PLACEHOLDER__
  - name: Final
    type: select
    proxies:
      - Proxies
      - DIRECT
rules:
  - MATCH,Final
`

const validProviderYAML = `
proxies:
  - name: node-a
    type: ss
    server: 127.0.0.1
    port: 8388
    cipher: aes-128-gcm
    password: test
`

const validProfileYAML = `
proxies:
  - name: node-a
    type: ss
    server: 127.0.0.1
    port: 8388
    cipher: aes-128-gcm
    password: test
proxy-groups:
  - name: Proxies
    type: select
    proxies:
      - node-a
rules:
  - MATCH,Proxies
`
