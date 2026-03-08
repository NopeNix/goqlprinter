package brotherql

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateBlankImage_NotNil(t *testing.T) {
	t.Parallel()

	img := CreateBlankImage(100, 50)
	if img == nil {
		t.Fatal("CreateBlankImage returned nil")
	}
}

func TestCreateBlankImage_CorrectDimensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		width, height int
	}{
		{100, 50},
		{1, 1},
		{720, 300},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			img := CreateBlankImage(tc.width, tc.height)
			if img.Bounds().Dx() != tc.width {
				t.Errorf("CreateBlankImage(%d,%d).Dx() = %d, want %d", tc.width, tc.height, img.Bounds().Dx(), tc.width)
			}
			if img.Bounds().Dy() != tc.height {
				t.Errorf("CreateBlankImage(%d,%d).Dy() = %d, want %d", tc.width, tc.height, img.Bounds().Dy(), tc.height)
			}
		})
	}
}

func TestCreateBlankImage_AllPixelsWhite(t *testing.T) {
	t.Parallel()

	width, height := 10, 10
	img := CreateBlankImage(width, height)

	for y := range height {
		for x := range width {
			px := img.GrayAt(x, y)
			if px.Y != 255 {
				t.Errorf("pixel at (%d,%d) = %d, want 255 (white)", x, y, px.Y)
			}
		}
	}
}

// TestRasterThreshold verifies the pixel threshold logic used in rasterize:
// pixels with gray value < 250 are treated as black (printed), >= 250 as white (not printed).
// This is a pure logic test with no printer I/O.
func TestRasterThreshold(t *testing.T) {
	t.Parallel()

	// We use a BrotherQL with a nil backend since rasterize() doesn't use the backend.
	bql := &BrotherQL{model: "QL-800"}

	tests := []struct {
		name      string
		grayValue uint8
		wantBlack bool // true means the bit should be set (printed as black)
	}{
		{"value=0 (black)", 0, true},
		{"value=249 (just below threshold)", 249, true},
		{"value=250 (at threshold)", 250, false},
		{"value=255 (white)", 255, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create an 8-pixel wide, 1-pixel tall image (bytesPerRow=1).
			bytesPerRow := 1
			width := 8
			img := image.NewGray(image.Rect(0, 0, width, 1))

			for x := range width {
				img.SetGray(x, 0, color.Gray{Y: 255})
			}
			// Set pixel 0 to the test value.
			img.SetGray(0, 0, color.Gray{Y: tc.grayValue})

			rows := bql.rasterize(img, bytesPerRow, width)
			if len(rows) != 1 {
				t.Fatalf("rasterize returned %d rows, want 1", len(rows))
			}

			row := rows[0]
			if len(row) != bytesPerRow {
				t.Fatalf("rasterize row length = %d, want %d", len(row), bytesPerRow)
			}

			// Pixel 0 maps to byte 0, bit 7 (MSB): bitIndex = 7 - (0 % 8) = 7
			bitSet := (row[0] & (1 << 7)) != 0
			if bitSet != tc.wantBlack {
				t.Errorf("pixel value %d: bit set = %v, want %v (wantBlack=%v)",
					tc.grayValue, bitSet, tc.wantBlack, tc.wantBlack)
			}
		})
	}
}

// findTestFont searches for any .ttf file available for font-dependent tests.
func findTestFont() string {
	candidates := []string{
		"./KOMIKAX_.ttf",
		"./fonts/",
		"./testdata/",
		"../fonts/",
	}
	for _, c := range candidates {
		info, err := os.Stat(c)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			return c
		}
		entries, err := os.ReadDir(c)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if filepath.Ext(e.Name()) == ".ttf" {
				return filepath.Join(c, e.Name())
			}
		}
	}
	return ""
}

func TestMeasureText_WithFont(t *testing.T) {
	t.Parallel()

	fontPath := findTestFont()
	if fontPath == "" {
		t.Skip("no .ttf font file found; skipping MeasureText test")
	}

	w, h, err := MeasureText("Hello", fontPath, 24)
	if err != nil {
		t.Fatalf("MeasureText returned error: %v", err)
	}
	if w <= 0 {
		t.Errorf("MeasureText width = %d, want > 0", w)
	}
	if h <= 0 {
		t.Errorf("MeasureText height = %d, want > 0", h)
	}
}

func TestDrawText_WithFont(t *testing.T) {
	t.Parallel()

	fontPath := findTestFont()
	if fontPath == "" {
		t.Skip("no .ttf font file found; skipping DrawText test")
	}

	img := CreateBlankImage(300, 100)
	err := DrawText(img, "Test", fontPath, 20, 0, 0, 0)
	if err != nil {
		t.Fatalf("DrawText returned error: %v", err)
	}
}

func TestDrawText_EmptyTextIsNoop(t *testing.T) {
	t.Parallel()

	img := CreateBlankImage(100, 50)
	err := DrawText(img, "", "nonexistent.ttf", 12, 0, 0, 0)
	if err != nil {
		t.Errorf("DrawText with empty text returned error: %v, want nil", err)
	}
}
