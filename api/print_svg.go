package api

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"

	"goqlprinter/brotherql"
	"goqlprinter/internal/services"
)

// rasterizeSVG converts SVG content to an image using pure Go (oksvg + rasterx).
// maxWidth and maxHeight are upper bounds (0 means unconstrained). The SVG is
// rendered at its natural viewBox dimensions multiplied by scale, then shrunk
// proportionally to fit within the constraints while preserving aspect ratio.
func rasterizeSVG(_ context.Context, svg string, maxWidth, maxHeight int, scale float64) (image.Image, error) {
	icon, err := oksvg.ReadIconStream(strings.NewReader(svg))
	if err != nil {
		return nil, fmt.Errorf("SVG parse failed: %w", err)
	}

	vbW := icon.ViewBox.W
	vbH := icon.ViewBox.H
	if vbW <= 0 || vbH <= 0 {
		return nil, fmt.Errorf("SVG has zero dimensions (viewBox: %.0fx%.0f)", vbW, vbH)
	}

	// Start with natural dimensions scaled.
	w := vbW * scale
	h := vbH * scale

	// Shrink proportionally to fit within constraints.
	if maxWidth > 0 && w > float64(maxWidth) {
		ratio := float64(maxWidth) / w
		w = float64(maxWidth)
		h *= ratio
	}
	if maxHeight > 0 && h > float64(maxHeight) {
		ratio := float64(maxHeight) / h
		h = float64(maxHeight)
		w *= ratio
	}

	iw, ih := int(w), int(h)
	if iw <= 0 || ih <= 0 {
		return nil, fmt.Errorf("SVG has zero dimensions after scaling (viewBox: %.0fx%.0f, scale: %.2f)", vbW, vbH, scale)
	}

	rgba := image.NewRGBA(image.Rect(0, 0, iw, ih))
	// Fill white background
	draw.Draw(rgba, rgba.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)

	icon.SetTarget(0, 0, w, h)
	scanner := rasterx.NewScannerGV(iw, ih, rgba, rgba.Bounds())
	dasher := rasterx.NewDasher(iw, ih, scanner)
	icon.Draw(dasher, 1.0)

	return rgba, nil
}

func convertToGrayscale(img image.Image) *image.Gray {
	bounds := img.Bounds()
	gray := image.NewGray(bounds)
	draw.Draw(gray, bounds, img, bounds.Min, draw.Src)
	return gray
}

// PrintSVGRequest defines the structure for SVG printing
// @description Request body for printing SVG labels
type PrintSVGRequest struct {
	LabelSize           string  `json:"label_size" binding:"required"`
	SVGData             string  `json:"svg_data" binding:"required"`
	Printer             string  `json:"printer"` // Optional
	Model               string  `json:"model"`   // Optional
	SVGScale            float64 `json:"svg_scale"`
	Orientation         string  `json:"orientation"`
	ContentRotation     float64 `json:"content_rotation"`
	HorizontalAlignment string  `json:"horizontal_alignment"`
	VerticalAlignment   string  `json:"vertical_alignment"`
	CustomHeightMM      float64 `json:"custom_height_mm"`
}

// PrintSVG godoc
// @Summary Print SVG label
// @Description Prints a label from SVG content
// @Tags print
// @Accept json
// @Produce json
// @Param request body PrintSVGRequest true "SVG print request parameters"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /print_svg [post]
func (h *Handlers) PrintSVG(c *gin.Context) {
	var req PrintSVGRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	label, err := brotherql.GetLabel(req.LabelSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label size"})
		return
	}

	img, err := processSVG(c.Request.Context(), req, label)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if req.Printer == printerFile {
		saveDebugOutput(c, img, "label_svg", req.Model, req.Orientation)
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

	c.JSON(http.StatusOK, gin.H{"status": "print job sent successfully"})
}

// processSVG handles SVG conversion logic.
// The SVG is rendered at its natural aspect ratio (scaled by svg_scale),
// fitted within the printable area, and positioned according to alignment
// — matching the approach used for text labels.
func processSVG(ctx context.Context, req PrintSVGRequest, label brotherql.LabelSize) (*image.Gray, error) {
	isRotated := req.Orientation == orientRotated

	scale := req.SVGScale
	if scale <= 0 {
		scale = 1.0
	}

	printHeadDots := label.DotsPrintableWidth

	// tapeLengthDots: custom > label-defined > 0 (dynamic).
	var tapeLengthDots int
	if req.CustomHeightMM > 0 && !label.IsDieCut {
		tapeLengthDots = mmToDots(req.CustomHeightMM)
	} else {
		tapeLengthDots = label.DotsPrintableHeight // 0 for endless tape
	}

	// Determine rendering constraints — swapped when rotated.
	var maxWidth, maxHeight int
	if isRotated {
		if tapeLengthDots > 0 {
			maxWidth = tapeLengthDots
		}
		maxHeight = printHeadDots
	} else {
		maxWidth = printHeadDots
		if tapeLengthDots > 0 {
			maxHeight = tapeLengthDots
		}
	}

	svgImg, err := rasterizeSVG(ctx, req.SVGData, maxWidth, maxHeight, scale)
	if err != nil {
		return nil, fmt.Errorf("SVG processing failed: %w", err)
	}

	grayImg := convertToGrayscale(svgImg)

	// Apply content rotation if requested.
	if req.ContentRotation == 90 || req.ContentRotation == 270 {
		grayImg = brotherql.RotateImage(grayImg, req.ContentRotation)
	}

	svgW := grayImg.Bounds().Dx()
	svgH := grayImg.Bounds().Dy()

	// Determine canvas dimensions based on orientation.
	var canvasWidth, canvasHeight int
	if isRotated {
		canvasHeight = printHeadDots
		if tapeLengthDots > 0 {
			canvasWidth = tapeLengthDots
		} else {
			canvasWidth = svgW + 2*defaultPadding
		}
	} else {
		canvasWidth = printHeadDots
		if tapeLengthDots > 0 {
			canvasHeight = tapeLengthDots
		} else {
			canvasHeight = svgH + 2*defaultPadding
		}
	}

	img := brotherql.CreateBlankImage(canvasWidth, canvasHeight)

	// Horizontal alignment.
	var xPos int
	switch req.HorizontalAlignment {
	case alignStart:
		xPos = defaultPadding
	case alignEnd:
		xPos = canvasWidth - svgW - defaultPadding
	default: // "center" or unspecified
		xPos = (canvasWidth - svgW) / 2
	}

	// Clamp to canvas bounds.
	if xPos+svgW > canvasWidth {
		xPos = canvasWidth - svgW
	}
	if xPos < 0 {
		xPos = 0
	}

	// Vertical alignment.
	var yPos int
	switch req.VerticalAlignment {
	case alignCenter:
		yPos = (canvasHeight - svgH) / 2
	case alignEnd:
		yPos = canvasHeight - svgH - defaultPadding
	default: // "start" or unspecified
		yPos = defaultPadding
	}

	// Clamp to canvas bounds.
	if yPos+svgH > canvasHeight {
		yPos = canvasHeight - svgH
	}
	if yPos < 0 {
		yPos = 0
	}

	draw.Draw(img,
		image.Rect(xPos, yPos, xPos+svgW, yPos+svgH),
		grayImg,
		image.Point{0, 0},
		draw.Over)

	return img, nil
}
