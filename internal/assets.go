package internal

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

//go:embed assets/mihomo assets/subconverter assets/ui assets/geo
var embeddedAssets embed.FS

// ExtractEmbeddedGeo seeds the GeoIP database into rootDir so Mihomo does not
// need to download it on first run. Existing files are left untouched so
// Mihomo's own auto-updates persist across restarts.
func ExtractEmbeddedGeo(rootDir string) error {
	data, err := embeddedAssets.ReadFile("assets/geo/geoip.metadb")
	if err != nil {
		return fmt.Errorf("embedded geoip.metadb asset missing; run scripts/stage-assets.sh before release builds")
	}
	if len(data) == 0 {
		return fmt.Errorf("embedded geoip.metadb asset is empty")
	}
	target := filepath.Join(rootDir, "geoip.metadb")
	if _, err := os.Stat(target); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(rootDir, 0750); err != nil {
		return err
	}
	return atomicWriteFile(target, data, 0644)
}

func EmbeddedMihomo() ([]byte, error) {
	data, err := embeddedAssets.ReadFile("assets/mihomo/mihomo")
	if err != nil {
		return nil, fmt.Errorf("embedded mihomo asset missing; run scripts/stage-assets.sh before release builds")
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("embedded mihomo asset is empty")
	}
	return data, nil
}

func ExtractEmbeddedSubconverter(dst string) error {
	data, err := embeddedArchive("assets/subconverter", ".tar.gz")
	if err != nil {
		return err
	}
	return extractTarGzip(data, dst, false)
}

func ExtractEmbeddedUI(dst string) error {
	data, err := embeddedArchive("assets/ui", ".tgz", ".tar.gz")
	if err != nil {
		return err
	}
	return extractTarGzip(data, dst, true)
}

func extractTarGzip(data []byte, dst string, stripSingleRoot bool) error {
	stripRoot := ""
	if stripSingleRoot {
		root, err := singleArchiveRoot(data)
		if err != nil {
			return err
		}
		stripRoot = root
	}

	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("embedded archive is not gzip data: %w", err)
	}
	defer gz.Close()

	if err := os.MkdirAll(dst, 0750); err != nil {
		return err
	}

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		cleanName, err := cleanArchivePath(header.Name, stripRoot)
		if err != nil {
			return err
		}
		if cleanName == "" {
			continue
		}
		target := filepath.Join(dst, filepath.FromSlash(cleanName))

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0750); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0750); err != nil {
				return err
			}
			mode := os.FileMode(header.Mode) & 0777
			if mode == 0 {
				mode = 0644
			}
			if filepath.Base(target) == "subconverter" {
				mode = 0755
			}
			fileData := make([]byte, header.Size)
			if _, err := io.ReadFull(tr, fileData); err != nil {
				return err
			}
			if err := atomicWriteFile(target, fileData, mode); err != nil {
				return err
			}
		}
	}
}

func embeddedArchive(dir string, suffixes ...string) ([]byte, error) {
	entries, err := fs.ReadDir(embeddedAssets, dir)
	if err != nil {
		return nil, fmt.Errorf("embedded asset directory missing %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !hasAnySuffix(entry.Name(), suffixes) {
			continue
		}
		data, err := embeddedAssets.ReadFile(path.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("embedded archive is empty: %s", entry.Name())
		}
		return data, nil
	}
	return nil, fmt.Errorf("embedded archive missing in %s; run scripts/stage-assets.sh before release builds", dir)
}

func hasAnySuffix(name string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

func singleArchiveRoot(data []byte) (string, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("embedded archive is not gzip data: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	root := ""
	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				return root, nil
			}
			return "", err
		}
		cleanName, err := cleanArchivePath(header.Name, "")
		if err != nil {
			return "", err
		}
		if cleanName == "" {
			continue
		}
		first, _, ok := strings.Cut(cleanName, "/")
		if !ok && header.Typeflag == tar.TypeDir {
			first = cleanName
		} else if !ok {
			return "", nil
		}
		if root == "" {
			root = first
			continue
		}
		if root != first {
			return "", nil
		}
	}
}

func cleanArchivePath(name, stripRoot string) (string, error) {
	cleanName := path.Clean(strings.TrimPrefix(name, "./"))
	if cleanName == "." {
		return "", nil
	}
	if path.IsAbs(cleanName) || cleanName == ".." || strings.HasPrefix(cleanName, "../") {
		return "", fmt.Errorf("unsafe embedded archive path: %s", name)
	}
	if stripRoot != "" {
		if cleanName == stripRoot {
			return "", nil
		}
		cleanName = strings.TrimPrefix(cleanName, stripRoot+"/")
	}
	if cleanName == "." {
		return "", nil
	}
	return cleanName, nil
}
