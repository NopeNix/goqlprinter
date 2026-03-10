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
	Orientation         string  `json:"orientation"`
	ContentRotation     float64 `json:"content_rotation"`
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
		saveDebugOutput(c, grayImg, "label_png", payload.Model, payload.Orientation)
		return
	}

	printImg := rotateForPrinter(grayImg, payload.Orientation)
	err = services.ConnectToPrinter(h.Printers, payload.Printer, payload.Model, func(backend brotherql.Backend, model string) error {
		printerDev := brotherql.NewBrotherQL(model, backend)
		return printerDev.Print(printImg, label)
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

	isRotated := payload.Orientation == "rotated"

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

	// When rotated, fit constraints are swapped.
	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	var fitW, fitH int
	if isRotated {
		if tapeLengthDots > 0 {
			fitW = tapeLengthDots
		} else if srcW > 0 {
			fitW = printHeadDots // use printHead as base for proportional calc
		}
		fitH = printHeadDots
		if fitW == 0 {
			if srcW > 0 {
				fitW = srcH * fitH / srcW // derive proportionally
			}
			if fitW == 0 {
				fitW = 300
			}
		}
	} else {
		fitW = printHeadDots
		if tapeLengthDots > 0 {
			fitH = tapeLengthDots
		} else {
			if srcW > 0 {
				fitH = srcH * fitW / srcW
			}
			if fitH == 0 {
				fitH = 300
			}
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

	resized := imaging.Fit(img, scaledW, scaledH, imaging.Lanczos)
	grayResized := convertToGrayscale(resized)

	// Apply content rotation if requested.
	if payload.ContentRotation == 90 || payload.ContentRotation == 270 {
		grayResized = brotherql.RotateImage(grayResized, payload.ContentRotation)
	}

	pngW := grayResized.Bounds().Dx()
	pngH := grayResized.Bounds().Dy()

	// Determine canvas dimensions based on orientation.
	var canvasWidth, canvasHeight int
	if isRotated {
		canvasHeight = printHeadDots
		if tapeLengthDots > 0 {
			canvasWidth = tapeLengthDots
		} else {
			canvasWidth = pngW + 2*defaultPadding
		}
	} else {
		canvasWidth = printHeadDots
		if tapeLengthDots > 0 {
			canvasHeight = tapeLengthDots
		} else {
			canvasHeight = pngH + 2*defaultPadding
		}
	}

	canvas := brotherql.CreateBlankImage(canvasWidth, canvasHeight)

	// Horizontal alignment.
	var xPos int
	switch payload.HorizontalAlignment {
	case "start":
		xPos = defaultPadding
	case "end":
		xPos = canvasWidth - pngW - defaultPadding
	default: // "center" or unspecified
		xPos = (canvasWidth - pngW) / 2
	}
	if xPos+pngW > canvasWidth {
		xPos = canvasWidth - pngW
	}
	if xPos < 0 {
		xPos = 0
	}

	// Vertical alignment.
	var yPos int
	switch payload.VerticalAlignment {
	case "center":
		yPos = (canvasHeight - pngH) / 2
	case "end":
		yPos = canvasHeight - pngH - defaultPadding
	default: // "start" or unspecified
		yPos = defaultPadding
	}
	if yPos+pngH > canvasHeight {
		yPos = canvasHeight - pngH
	}
	if yPos < 0 {
		yPos = 0
	}

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
