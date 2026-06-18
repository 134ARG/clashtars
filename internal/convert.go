package internal

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func ConvertSubscription(ctx context.Context, settings *Settings, provider Provider, raw []byte) ([]byte, error) {
	if isClashProfile(raw) {
		return raw, nil
	}

	decoded, err := decodeBase64(raw)
	if err == nil && isClashProfile(decoded) {
		return decoded, nil
	}

	return runSubconverter(ctx, settings, provider)
}

func decodeBase64(data []byte) ([]byte, error) {
	trimmed := strings.TrimSpace(string(data))
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	var lastErr error
	for _, enc := range encodings {
		decoded, err := enc.DecodeString(trimmed)
		if err == nil {
			return decoded, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func runSubconverter(ctx context.Context, settings *Settings, provider Provider) ([]byte, error) {
	if err := ExtractEmbeddedSubconverter(settings.SubconverterDir()); err != nil {
		return nil, err
	}

	binary := filepath.Join(settings.SubconverterDir(), "subconverter")
	if st, err := os.Stat(binary); err != nil || st.IsDir() {
		return nil, fmt.Errorf("embedded subconverter binary is missing")
	}

	generateINI := fmt.Sprintf(`[test]
path=%s
target=clash
ver=4
url=%s
`, settings.ProviderConvertedPath(provider.Name), settings.ProviderRawPath(provider.Name))
	if err := atomicWriteFile(filepath.Join(settings.SubconverterDir(), "generate.ini"), []byte(generateINI), 0644); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, binary, "-g")
	cmd.Dir = settings.SubconverterDir()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("subconverter failed: %w", err)
	}

	converted, err := os.ReadFile(settings.ProviderConvertedPath(provider.Name))
	if err != nil {
		return nil, err
	}
	if !isClashProfile(converted) {
		return nil, fmt.Errorf("subconverter output is not a Clash profile")
	}
	return converted, nil
}
