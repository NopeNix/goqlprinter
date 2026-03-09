package brotherql

import (
	"image"
	"image/color"
	"image/draw"
	"os"

	"github.com/disintegration/imaging"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/skip2/go-qrcode"
	"golang.org/x/image/font"
)

// CreateBlankImage creates a new white grayscale image.
func CreateBlankImage(width, height int) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.SetGray(x, y, color.Gray{Y: 255})
		}
	}
	return img
}

// DrawText draws text onto an image, with optional rotation.
// x, y are the top-left coordinates where the final (possibly rotated) text
// bounding box should be placed on the target image.
func DrawText(img *image.Gray, text string, fontPath string, size float64, x, y int, rotationAngle float64) error {
	if text == "" {
		return nil
	}

	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		return err
	}

	parsedFont, err := truetype.Parse(fontBytes)
	if err != nil {
		return err
	}

	face := truetype.NewFace(parsedFont, &truetype.Options{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})

	textWidth, textHeight, err := MeasureText(text, fontPath, size)
	if err != nil {
		return err
	}
	if textWidth == 0 || textHeight == 0 {
		return nil
	}

	// BoundString gives the true pixel bounds, which may start at negative x
	// for italic or script fonts with left-leaning glyphs.
	d := &font.Drawer{Face: face}
	bounds, _ := d.BoundString(text)

	leftOffset := 0
	if bounds.Min.X.Round() < 0 {
		leftOffset = -bounds.Min.X.Round()
	}

	// Extra width accommodates glyphs that overhang to the left.
	tempRGBA := image.NewRGBA(image.Rect(0, 0, textWidth+leftOffset, textHeight))
	draw.Draw(tempRGBA, tempRGBA.Bounds(), image.White, image.Point{}, draw.Src)

	metrics := face.Metrics()
	ascent := metrics.Ascent.Round()

	d = &font.Drawer{
		Dst:  tempRGBA,
		Src:  image.Black,
		Face: face,
		Dot:  freetype.Pt(leftOffset, ascent),
	}
	d.DrawString(text)

	var finalImage image.Image = tempRGBA
	if rotationAngle != 0 {
		finalImage = imaging.Rotate(tempRGBA, rotationAngle, color.White)
	}

	rotatedGray := image.NewGray(finalImage.Bounds())
	draw.Draw(rotatedGray, rotatedGray.Bounds(), finalImage, finalImage.Bounds().Min, draw.Src)

	// Place the (rotated) text at the given x, y position.
	rotatedWidth := rotatedGray.Bounds().Dx()
	rotatedHeight := rotatedGray.Bounds().Dy()
	draw.Draw(img, image.Rect(x, y, x+rotatedWidth, y+rotatedHeight), rotatedGray, image.Point{}, draw.Over)

	return nil
}

// MeasureText measures the width and height of the rendered text.
// Uses BoundString for accurate bounds that include glyph overhangs (important for italic/script fonts).
func MeasureText(text string, fontPath string, size float64) (int, int, error) {
	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		return 0, 0, err
	}

	parsedFont, err := truetype.Parse(fontBytes)
	if err != nil {
		return 0, 0, err
	}

	face := truetype.NewFace(parsedFont, &truetype.Options{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})

	d := &font.Drawer{
		Face: face,
	}

	// BoundString gives actual pixel bounds including glyph overhangs.
	bounds, advance := d.BoundString(text)

	// Use the larger of advance width and bounds width to handle fonts where
	// glyphs extend beyond the advance width.
	advanceWidth := advance.Round()
	boundsWidth := (bounds.Max.X - bounds.Min.X).Round()
	width := max(advanceWidth, boundsWidth)

	metrics := face.Metrics()
	metricsHeight := (metrics.Ascent + metrics.Descent).Round()
	boundsHeight := (bounds.Max.Y - bounds.Min.Y).Round()
	height := max(metricsHeight, boundsHeight)

	return width, height, nil
}

// RotateImage rotates an image by the given angle and returns a grayscale result.
func RotateImage(img image.Image, angle float64) *image.Gray {
	rotatedRGBA := imaging.Rotate(img, angle, color.White)
	rotatedGray := image.NewGray(rotatedRGBA.Bounds())
	draw.Draw(rotatedGray, rotatedGray.Bounds(), rotatedRGBA, rotatedRGBA.Bounds().Min, draw.Src)
	return rotatedGray
}

// DrawQRCode generates a QR code and draws it onto img at position (x, y).
func DrawQRCode(img *image.Gray, content string, x, y, size int) error {
	qr, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return err
	}

	qrImg := qr.Image(size)

	pos := image.Pt(x, y)
	r := image.Rectangle{Min: pos, Max: pos.Add(qrImg.Bounds().Size())}

	draw.Draw(img, r, qrImg, image.Point{}, draw.Over)

	return nil
}
