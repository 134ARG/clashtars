package internal

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractTarGzipStripsSingleTopLevelDir(t *testing.T) {
	dst := t.TempDir()

	if err := extractTarGzip(testTarGzip(t, map[string]string{
		"dist/index.html": "ok",
	}), dst, true); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dst, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "ok" {
		t.Fatalf("index.html = %q, want ok", data)
	}
}

func TestCleanArchivePathRejectsTraversal(t *testing.T) {
	if _, err := cleanArchivePath("../bad", ""); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
}

func testTarGzip(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(body)),
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
