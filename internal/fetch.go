package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type FetchFunc func(context.Context, *Settings, Provider) ([]byte, error)

func FetchSubscription(ctx context.Context, settings *Settings, provider Provider) ([]byte, error) {
	if provider.URL == "" {
		return nil, fmt.Errorf("provider %q url is empty", provider.Name)
	}

	parsed, err := url.Parse(provider.URL)
	if err == nil && parsed.Scheme == "file" {
		return os.ReadFile(parsed.Path)
	}
	if err == nil && parsed.Scheme == "" && strings.HasPrefix(provider.URL, "/") {
		return os.ReadFile(provider.URL)
	}

	ctx, cancel := context.WithTimeout(ctx, settings.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", settings.UserAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("subscription fetch returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("subscription response is empty")
	}
	return body, nil
}
