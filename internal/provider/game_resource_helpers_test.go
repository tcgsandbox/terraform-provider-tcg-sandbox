package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- readImageAsDataURL ---

func TestReadImageAsDataURL_PNG(t *testing.T) {
	t.Parallel()
	result, err := readImageAsDataURL("testdata/test_banner.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(result, "data:image/png;base64,") {
		t.Errorf("expected data URL with image/png mime type, got: %s", result[:min(len(result), 50)])
	}
}

func TestReadImageAsDataURL_JPEG(t *testing.T) {
	t.Parallel()
	// Write a minimal JPEG (SOI + EOI markers — valid enough to read).
	path := filepath.Join(t.TempDir(), "test.jpg")
	if err := os.WriteFile(path, []byte{0xFF, 0xD8, 0xFF, 0xD9}, 0600); err != nil {
		t.Fatal(err)
	}
	result, err := readImageAsDataURL(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(result, "data:image/jpeg;base64,") {
		t.Errorf("expected image/jpeg mime type, got prefix: %s", result[:min(len(result), 50)])
	}
}

func TestReadImageAsDataURL_FileNotFound(t *testing.T) {
	t.Parallel()
	_, err := readImageAsDataURL("/nonexistent/path/image.png")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestReadImageAsDataURL_UnknownExtension(t *testing.T) {
	t.Parallel()
	// An unknown extension falls back to http.DetectContentType.
	// Write minimal PNG bytes but with an unrecognised extension.
	path := filepath.Join(t.TempDir(), "image.tiff")
	pngBytes := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
	}
	if err := os.WriteFile(path, pngBytes, 0600); err != nil {
		t.Fatal(err)
	}
	result, err := readImageAsDataURL(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// http.DetectContentType should sniff the PNG magic bytes.
	if !strings.HasPrefix(result, "data:image/png;base64,") {
		t.Errorf("expected sniffed image/png for PNG bytes, got prefix: %s", result[:min(len(result), 60)])
	}
}

// --- hashImageFile ---

func TestHashImageFile_Deterministic(t *testing.T) {
	t.Parallel()
	h1, err := hashImageFile("testdata/test_banner.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	h2, err := hashImageFile("testdata/test_banner.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h1 != h2 {
		t.Errorf("hash is not deterministic: %s != %s", h1, h2)
	}
	// SHA-256 hex digest is always 64 chars.
	if len(h1) != 64 {
		t.Errorf("expected 64-char hex digest, got len=%d", len(h1))
	}
}

func TestHashImageFile_DifferentContent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	pathA := filepath.Join(dir, "a.png")
	pathB := filepath.Join(dir, "b.png")
	if err := os.WriteFile(pathA, []byte{0x01, 0x02}, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathB, []byte{0x03, 0x04}, 0600); err != nil {
		t.Fatal(err)
	}

	hA, err := hashImageFile(pathA)
	if err != nil {
		t.Fatal(err)
	}
	hB, err := hashImageFile(pathB)
	if err != nil {
		t.Fatal(err)
	}
	if hA == hB {
		t.Error("expected different hashes for different file contents")
	}
}

func TestHashImageFile_FileNotFound(t *testing.T) {
	t.Parallel()
	_, err := hashImageFile("/nonexistent/file.png")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}
