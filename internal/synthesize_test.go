package internal

import (
	"strings"
	"testing"
)

func TestSynthesizeConfigMergesProfileSections(t *testing.T) {
	settings := &Settings{
		RootDir: "/var/lib/clashtars",
		Mihomo:  newMappingNode(),
	}
	setMapValue(settings.Mihomo, "tun", newMappingNode())
	applyMihomoDefaults(settings)

	out, err := SynthesizeConfig(settings, []byte(validProfileYAML))
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, want := range []string{
		"external-controller: 0.0.0.0:9091",
		"external-ui: /var/lib/clashtars/ui",
		"proxies:",
		"proxy-groups:",
		"rules:",
		"tun:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated config missing %q:\n%s", want, text)
		}
	}
}

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
