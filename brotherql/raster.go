package brotherql

import (
	"image"
	"image/color"
	"image/draw"
	"io/ioutil"
	//	"math"

	"github.com/disintegration/imaging"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/skip2/go-qrcode"
	"golang.org/x/image/font"
)

// CreateBlankImage creates a new, blank grayscale image.
func CreateBlankImage(width, height int) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, width, height))
	// Fill the image with white color
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetGray(x, y, color.Gray{Y: 255})
		}
	}
	return img
}

// DrawText draws text onto an image, with optional rotation.
// x, y are the top-left coordinates of the text's bounding box *before* rotation.
func DrawText(img *image.Gray, text string, fontPath string, size float64, x, y int, rotationAngle float64) error {
	if text == "" {
		return nil
	}

	fontBytes, err := ioutil.ReadFile(fontPath)
	if err != nil {
		return err
	}

	parsedFont, err := truetype.Parse(fontBytes)
	if err != nil {
		return err
	}

	face := truetype.NewFace(parsedFont, &truetype.Options{
		Size:    size,
		DPI:     72, // Standard DPI
		Hinting: font.HintingFull,
	})

	// Measure text dimensions
	textWidth, textHeight, err := MeasureText(text, fontPath, size)
	if err != nil {
		return err
	}
	if textWidth == 0 || textHeight == 0 {
		return nil // Nothing to draw
	}

	// Get actual bounds to handle negative offsets (italic/script fonts)
	d := &font.Drawer{Face: face}
	bounds, _ := d.BoundString(text)

	// Calculate offset for glyphs that start before x=0 (left-leaning italic)
	leftOffset := 0
	if bounds.Min.X.Round() < 0 {
		leftOffset = -bounds.Min.X.Round()
	}

	// Create a temporary RGBA image with extra space for overhanging glyphs
	tempRGBA := image.NewRGBA(image.Rect(0, 0, textWidth+leftOffset, textHeight))
	draw.Draw(tempRGBA, tempRGBA.Bounds(), image.White, image.Point{}, draw.Src) // White background for rotation

	// Get font metrics
	metrics := face.Metrics()
	ascent := metrics.Ascent.Round()

	// Draw text onto the temporary image. The origin for drawing is the baseline.
	// Add leftOffset to handle glyphs that extend to the left of origin.
	d = &font.Drawer{
		Dst:  tempRGBA,
		Src:  image.Black,
		Face: face,
		Dot:  freetype.Pt(leftOffset, ascent),
	}
	d.DrawString(text)

	// Rotate the temporary image if angle is not 0
	var finalImage image.Image = tempRGBA
	if rotationAngle != 0 {
		finalImage = imaging.Rotate(tempRGBA, rotationAngle, color.White)
	}

	// Convert the (potentially rotated) RGBA image to Gray
	rotatedGray := image.NewGray(finalImage.Bounds())
	draw.Draw(rotatedGray, rotatedGray.Bounds(), finalImage, finalImage.Bounds().Min, draw.Src)

	// Calculate the top-left corner for drawing the rotated image on the main image.
	// Use actual tempRGBA dimensions (includes leftOffset for overhanging glyphs)
	tempWidth := textWidth + leftOffset
	tempHeight := textHeight
	targetCenterX := x + tempWidth/2
	targetCenterY := y + tempHeight/2

	rotatedWidth := rotatedGray.Bounds().Dx()
	rotatedHeight := rotatedGray.Bounds().Dy()

	drawX := targetCenterX - rotatedWidth/2
	drawY := targetCenterY - rotatedHeight/2

	// Draw the final text image onto the main image
	draw.Draw(img, image.Rect(drawX, drawY, drawX+rotatedWidth, drawY+rotatedHeight), rotatedGray, image.Point{}, draw.Over)

	return nil
}

// MeasureText measures the width and height of the rendered text.
// Uses BoundString for accurate bounds that include glyph overhangs (important for italic/script fonts).
func MeasureText(text string, fontPath string, size float64) (int, int, error) {
	fontBytes, err := ioutil.ReadFile(fontPath)
	if err != nil {
		return 0, 0, err
	}

	parsedFont, err := truetype.Parse(fontBytes)
	if err != nil {
		return 0, 0, err
	}

	face := truetype.NewFace(parsedFont, &truetype.Options{
		Size:    size,
		DPI:     72, // Standard DPI
		Hinting: font.HintingFull,
	})

	d := &font.Drawer{
		Face: face,
	}

	// Use BoundString for actual pixel bounds (includes glyph overhangs)
	bounds, advance := d.BoundString(text)

	// Calculate width: max of advance width and actual bounds
	// This handles fonts where glyphs extend past the advance width
	advanceWidth := advance.Round()
	boundsWidth := (bounds.Max.X - bounds.Min.X).Round()
	width := advanceWidth
	if boundsWidth > width {
		width = boundsWidth
	}

	// Calculate height from actual bounds
	metrics := face.Metrics()
	metricsHeight := (metrics.Ascent + metrics.Descent).Round()
	boundsHeight := (bounds.Max.Y - bounds.Min.Y).Round()
	height := metricsHeight
	if boundsHeight > height {
		height = boundsHeight
	}

	return width, height, nil
}

// RotateImage rotates an image by the given angle.
// This function is no longer used for text rotation, but kept for potential future use.
func RotateImage(img image.Image, angle float64) *image.Gray {
	// The imaging.Rotate function returns an image.RGBA, so we need to convert it to image.Gray
	rotatedRGBA := imaging.Rotate(img, angle, color.White)

	// Create a new grayscale image with the same bounds as the rotated RGBA image
	rotatedGray := image.NewGray(rotatedRGBA.Bounds())

	// Draw the rotated RGBA image onto the new grayscale image
	draw.Draw(rotatedGray, rotatedGray.Bounds(), rotatedRGBA, rotatedRGBA.Bounds().Min, draw.Src)

	return rotatedGray
}

// DrawQRCode generates and draws a QR code onto an image.
func DrawQRCode(img *image.Gray, content string, x, y, size int) error {
	qr, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return err
	}

	qrImg := qr.Image(size)

	// Define the position where the QR code will be drawn
	pos := image.Pt(x, y)
	r := image.Rectangle{Min: pos, Max: pos.Add(qrImg.Bounds().Size())}

	draw.Draw(img, r, qrImg, image.Point{}, draw.Over)

	return nil
}
