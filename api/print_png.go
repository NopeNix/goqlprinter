package api

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"net/http"
	"os"
	"time"

	"goqlprinter/brotherql"
	"goqlprinter/internal/services"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
)

// PrintPNGPayload defines the structure for the JSON request
// @description Request body for printing PNG images
type PrintPNGPayload struct {
	Printer             string  `json:"printer"` // Optional
	Model               string  `json:"model"`   // Optional
	LabelSize           string  `json:"label_size" binding:"required"`
	PNGData             string  `json:"png_data" binding:"required"`
	PNGScale            float64 `json:"png_scale"`
	HorizontalAlignment string  `json:"horizontal_alignment"`
	VerticalAlignment   string  `json:"vertical_alignment"`
	CustomHeightMM      float64 `json:"custom_height_mm"`
}

// PrintPNGLabel godoc
// @Summary Print from PNG data
// @Description Prints a label from base64 encoded PNG data
// @Tags print
// @Accept json
// @Produce json
// @Param request body PrintPNGPayload true "Print request parameters"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /print_png [post]
func (h *Handlers) PrintPNGLabel(c *gin.Context) {
	var payload PrintPNGPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	label, err := brotherql.GetLabel(payload.LabelSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label_size"})
		return
	}

	grayImg, err := processPNG(payload, label)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if payload.Printer == "file" {
		var buf bytes.Buffer
		if err := png.Encode(&buf, grayImg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "PNG encode failed"})
			return
		}
		filename := fmt.Sprintf("debug_output/labelpng_%d.png", time.Now().UnixNano())
		if err := writeFile(filename, buf.Bytes()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save PNG: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "Image saved to file successfully", "filename": filename})
		return
	}

	err = services.ConnectToPrinter(h.Printers, payload.Printer, payload.Model, func(backend brotherql.Backend, model string) error {
		printerDev := brotherql.NewBrotherQL(model, backend)
		return printerDev.Print(grayImg, label)
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Print failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Print job sent successfully"})
}

// processPNG decodes base64 PNG data, scales it to fit the printable area
// (multiplied by the user scale factor), and positions it on the canvas
// using the requested alignment — matching the approach used for SVG/QR.
func processPNG(payload PrintPNGPayload, label brotherql.LabelSize) (*image.Gray, error) {
	pngBytes, err := base64.StdEncoding.DecodeString(payload.PNGData)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 PNG data: %w", err)
	}

	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, fmt.Errorf("invalid PNG file: %w", err)
	}

	scale := payload.PNGScale
	if scale <= 0 {
		scale = 1.0
	}

	printHeadDots := label.DotsPrintableWidth

	// tapeLengthDots: custom > label-defined > 0 (dynamic).
	var tapeLengthDots int
	if payload.CustomHeightMM > 0 && !label.IsDieCut {
		tapeLengthDots = mmToDots(payload.CustomHeightMM)
	} else {
		tapeLengthDots = label.DotsPrintableHeight // 0 for endless tape
	}

	// Base-fit: scale the image to fill the printable area, then apply user scale.
	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	var fitW, fitH int
	if tapeLengthDots > 0 {
		// Die-cut or custom height: fit within printable rectangle.
		fitW = printHeadDots
		fitH = tapeLengthDots
	} else {
		// Continuous tape: fit width, derive height proportionally.
		fitW = printHeadDots
		if srcW > 0 {
			fitH = srcH * fitW / srcW
		}
		if fitH == 0 {
			fitH = 300 // ultimate fallback
		}
	}

	// Apply user scale to the base-fit dimensions.
	scaledW := int(float64(fitW) * scale)
	scaledH := int(float64(fitH) * scale)
	if scaledW < 1 {
		scaledW = 1
	}
	if scaledH < 1 {
		scaledH = 1
	}

	// Resize source image to (scaledW x scaledH) while preserving aspect ratio.
	resized := imaging.Fit(img, scaledW, scaledH, imaging.Lanczos)
	pngW := resized.Bounds().Dx()
	pngH := resized.Bounds().Dy()

	// Canvas height: fixed for die-cut/custom, dynamic for continuous tape.
	imageHeight := tapeLengthDots
	if imageHeight <= 0 {
		imageHeight = pngH + 2*defaultPadding
	}

	canvas := brotherql.CreateBlankImage(printHeadDots, imageHeight)

	// Horizontal alignment.
	var xPos int
	switch payload.HorizontalAlignment {
	case "start":
		xPos = defaultPadding
	case "end":
		xPos = printHeadDots - pngW - defaultPadding
	default: // "center" or unspecified
		xPos = (printHeadDots - pngW) / 2
	}

	// Clamp to canvas bounds.
	if xPos+pngW > printHeadDots {
		xPos = printHeadDots - pngW
	}
	if xPos < 0 {
		xPos = 0
	}

	// Vertical alignment.
	var yPos int
	switch payload.VerticalAlignment {
	case "center":
		yPos = (imageHeight - pngH) / 2
	case "end":
		yPos = imageHeight - pngH - defaultPadding
	default: // "start" or unspecified
		yPos = defaultPadding
	}

	// Clamp to canvas bounds.
	if yPos+pngH > imageHeight {
		yPos = imageHeight - pngH
	}
	if yPos < 0 {
		yPos = 0
	}

	// Convert resized image to grayscale and draw onto canvas.
	grayResized := convertToGrayscale(resized)
	draw.Draw(canvas,
		image.Rect(xPos, yPos, xPos+pngW, yPos+pngH),
		grayResized,
		image.Point{0, 0},
		draw.Over)

	return canvas, nil
}

func writeFile(filename string, data []byte) error {
	err := os.MkdirAll("debug_output", 0750)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0600)
}
