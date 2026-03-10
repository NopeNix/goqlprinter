package api

import (
	"fmt"
	"image"
	"net/http"
	"time"

	"goqlprinter/brotherql"
	"goqlprinter/internal/services"

	"github.com/gin-gonic/gin"
)

const (
	defaultModel   = "QL-820NWB"
	defaultPadding = 10

	printerFile   = "file"
	orientRotated = "rotated"
	alignStart    = "start"
	alignCenter   = "center"
	alignEnd      = "end"
)

// PrintRequest defines the structure for the print request
// @description Request body for printing labels with text or SVG content
type PrintRequest struct {
	Text                string  `json:"text"`
	LabelSize           string  `json:"label_size" binding:"required"`
	FontFamily          string  `json:"font_family"`
	FontSize            float64 `json:"font_size"`
	Printer             string  `json:"printer"` // e.g., "QL-800", "usb:001:005" or "file" (optional)
	Model               string  `json:"model"`   // e.g., "QL-800" (optional)
	Orientation         string  `json:"orientation"`
	HorizontalAlignment string  `json:"horizontal_alignment"`
	VerticalAlignment   string  `json:"vertical_alignment"`
	TextRotation        float64 `json:"text_rotation"`
	TextAlign           string  `json:"text_align"`
	SVGData             string  `json:"svg_data"`
	SVGScale            float64 `json:"svg_scale"`
	CustomHeightMM      float64 `json:"custom_height_mm"`
}

// mmToDots converts millimeters to dots at 300 DPI.
func mmToDots(mm float64) int {
	return int(mm * 300.0 / 25.4)
}

// rotateForPrinter rotates a "rotated"-orientation image 90° so that its
// width matches the print head. Standard-orientation images are returned as-is.
func rotateForPrinter(img *image.Gray, orientation string) *image.Gray {
	if orientation == orientRotated {
		return brotherql.RotateImage(img, 90)
	}
	return img
}

// saveDebugOutput saves img to a debug PNG file (and its raster variant) and
// writes a JSON response. It is used by all print handlers when printer == printerFile.
// orientation controls whether the raster variant is rotated to match the print head.
func saveDebugOutput(c *gin.Context, img *image.Gray, prefix string, model string, orientation string) {
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("debug_output/%s_%s.png", prefix, timestamp)
	err := brotherql.SaveImageToFile(img, filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save image to file: %v", err)})
		return
	}
	// Also save the raster as the printer would receive it (right-align + flip).
	rasterFile := fmt.Sprintf("debug_output/%s_%s_raster.png", prefix, timestamp)
	modelName := model
	if modelName == "" {
		modelName = defaultModel
	}
	printImg := rotateForPrinter(img, orientation)
	if rasterImg, err := brotherql.PrepareForPrint(printImg, modelName); err == nil {
		_ = brotherql.SaveImageToFile(rasterImg, rasterFile)
	}
	c.JSON(http.StatusOK, gin.H{
		"status":      "Image saved to file successfully",
		"filename":    filename,
		"raster_file": rasterFile,
	})
}

// renderTextLabel renders text onto a grayscale image sized for the given label.
func (h *Handlers) renderTextLabel(req PrintRequest, label brotherql.LabelSize) (*image.Gray, error) {
	isRotated := req.Orientation == orientRotated

	fontPath, err := h.Fonts.GetFontPath(req.FontFamily)
	if err != nil {
		return nil, fmt.Errorf("failed to get font path: %w", err)
	}

	scaledFontSize := req.FontSize * 4
	unrotatedWidth, unrotatedHeight, err := brotherql.MeasureText(req.Text, fontPath, scaledFontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to measure text: %w", err)
	}

	// Swap dimensions when rotated 90° or 270°.
	var textBoundsWidth, textBoundsHeight int
	if req.TextRotation == 90 || req.TextRotation == 270 {
		textBoundsWidth = unrotatedHeight
		textBoundsHeight = unrotatedWidth
	} else {
		textBoundsWidth = unrotatedWidth
		textBoundsHeight = unrotatedHeight
	}

	// printHeadDots is always the fixed print-head dimension.
	printHeadDots := label.DotsPrintableWidth

	// tapeLengthDots: custom > label-defined > 0 (dynamic).
	var tapeLengthDots int
	if req.CustomHeightMM > 0 && !label.IsDieCut {
		tapeLengthDots = mmToDots(req.CustomHeightMM)
	} else {
		tapeLengthDots = label.DotsPrintableHeight // 0 for endless tape
	}

	// Determine canvas dimensions based on orientation.
	var canvasWidth, canvasHeight int
	if isRotated {
		// Rotated: we render on a swapped canvas, then rotate 90°.
		canvasHeight = printHeadDots
		if tapeLengthDots > 0 {
			canvasWidth = tapeLengthDots
		} else {
			canvasWidth = textBoundsWidth + (2 * defaultPadding) // dynamic
		}
	} else {
		canvasWidth = printHeadDots
		if tapeLengthDots > 0 {
			canvasHeight = tapeLengthDots
		} else {
			canvasHeight = textBoundsHeight + (2 * defaultPadding) // dynamic
		}
	}

	img := brotherql.CreateBlankImage(canvasWidth, canvasHeight)

	var x int
	switch req.HorizontalAlignment {
	case alignStart:
		x = defaultPadding
	case alignCenter:
		x = (canvasWidth - textBoundsWidth) / 2
	case alignEnd:
		x = canvasWidth - textBoundsWidth - defaultPadding
	default:
		x = defaultPadding
	}

	var y int
	switch req.VerticalAlignment {
	case alignStart:
		y = defaultPadding
	case alignCenter:
		y = (canvasHeight - textBoundsHeight) / 2
	case alignEnd:
		y = canvasHeight - textBoundsHeight - defaultPadding
	default:
		y = defaultPadding
	}

	textAlign := req.TextAlign
	if textAlign == "" {
		textAlign = "left"
	}
	err = brotherql.DrawText(img, req.Text, fontPath, scaledFontSize, x, y, req.TextRotation, textAlign)
	if err != nil {
		return nil, fmt.Errorf("failed to draw text: %w", err)
	}

	return img, nil
}

// PrintLabel godoc
// @Summary Print a text or SVG label
// @Description Prints a label with either text or SVG content
// @Tags print
// @Accept json
// @Produce json
// @Param request body PrintRequest true "Print request parameters"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /print [post]
func (h *Handlers) PrintLabel(c *gin.Context) {
	var req PrintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	label, err := brotherql.GetLabel(req.LabelSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label size"})
		return
	}

	var img *image.Gray

	if req.SVGData != "" {
		var err error
		svgReq := PrintSVGRequest{
			LabelSize:           req.LabelSize,
			SVGData:             req.SVGData,
			Printer:             req.Printer,
			Model:               req.Model,
			SVGScale:            req.SVGScale,
			Orientation:         req.Orientation,
			HorizontalAlignment: req.HorizontalAlignment,
			VerticalAlignment:   req.VerticalAlignment,
			CustomHeightMM:      req.CustomHeightMM,
		}
		img, err = processSVG(c.Request.Context(), svgReq, label)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		var err error
		img, err = h.renderTextLabel(req, label)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if req.Printer == printerFile {
		saveDebugOutput(c, img, "label", req.Model, req.Orientation)
		return
	}

	printImg := rotateForPrinter(img, req.Orientation)
	err = services.ConnectToPrinter(h.Printers, req.Printer, req.Model, func(backend brotherql.Backend, model string) error {
		printerDev := brotherql.NewBrotherQL(model, backend)
		return printerDev.Print(printImg, label)
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Print failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Print job sent successfully"})
}
