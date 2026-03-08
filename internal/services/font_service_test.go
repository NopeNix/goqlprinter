package services_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goqlprinter/internal/services"
)

func TestNewFontService(t *testing.T) {
	t.Parallel()

	svc := services.NewFontService([]string{})
	if svc == nil {
		t.Fatal("expected non-nil FontService")
	}
}

func TestListFonts_Empty(t *testing.T) {
	t.Parallel()

	svc := services.NewFontService([]string{})

	fonts, err := svc.ListFonts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fonts) != 0 {
		t.Fatalf("expected 0 fonts, got %d", len(fonts))
	}
}

func TestListFonts_NonexistentDir(t *testing.T) {
	t.Parallel()

	svc := services.NewFontService([]string{"/nonexistent/path/that/does/not/exist"})

	fonts, err := svc.ListFonts()
	if err != nil {
		t.Fatalf("expected no error for nonexistent dir, got: %v", err)
	}
	if len(fonts) != 0 {
		t.Fatalf("expected 0 fonts from nonexistent dir, got %d", len(fonts))
	}
}

func TestListFonts_FindsFontFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "TestFont.ttf"), []byte{}, 0644); err != nil {
		t.Fatalf("failed to create font file: %v", err)
	}

	svc := services.NewFontService([]string{dir})

	fonts, err := svc.ListFonts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fonts) != 1 {
		t.Fatalf("expected 1 font, got %d", len(fonts))
	}
	if fonts[0] != "TestFont" {
		t.Errorf("expected font name %q, got %q", "TestFont", fonts[0])
	}
}

func TestListFonts_MultipleDirs(t *testing.T) {
	t.Parallel()

	dir1 := t.TempDir()
	dir2 := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir1, "FontA.ttf"), []byte{}, 0644); err != nil {
		t.Fatalf("failed to create font file in dir1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir2, "FontB.ttf"), []byte{}, 0644); err != nil {
		t.Fatalf("failed to create font file in dir2: %v", err)
	}

	svc := services.NewFontService([]string{dir1, dir2})

	fonts, err := svc.ListFonts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fonts) != 2 {
		t.Fatalf("expected 2 fonts total, got %d: %v", len(fonts), fonts)
	}
}

func TestGetFontPath_Found(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fontFile := filepath.Join(dir, "MyFont.ttf")
	if err := os.WriteFile(fontFile, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create font file: %v", err)
	}

	svc := services.NewFontService([]string{dir})

	path, err := svc.GetFontPath("MyFont")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(path, "MyFont.ttf") {
		t.Errorf("expected path ending in MyFont.ttf, got %q", path)
	}
}

func TestGetFontPath_NotFound(t *testing.T) {
	t.Parallel()

	svc := services.NewFontService([]string{})

	_, err := svc.GetFontPath("MissingFont")
	if err == nil {
		t.Fatal("expected error for missing font, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error to contain 'not found', got: %v", err)
	}
}

func TestGetFontPath_OnlyOtfIfNoTtf(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fontFile := filepath.Join(dir, "MyFont.otf")
	if err := os.WriteFile(fontFile, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create .otf font file: %v", err)
	}

	svc := services.NewFontService([]string{dir})

	path, err := svc.GetFontPath("MyFont")
	if err != nil {
		t.Fatalf("unexpected error for .otf font: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path for .otf font")
	}
}
