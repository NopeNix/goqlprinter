package api

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"

	"goqlprinter/brotherql"
	"goqlprinter/internal/services"
)

// SVGPrintParams defines the parameters for SVG printing
// @description Parameters used for controlling SVG printing
type SVGPrintParams struct {
	SVGData                string  `json:"svg_data"`                 // SVG content to print
	SVGScale               float64 `json:"svg_scale"`                // Scaling factor for SVG (default 1.0)
	SVGHorizontalAlignment string  `json:"svg_horizontal_alignment"` // "start", "center", or "end" (default "center")
	SVGVerticalAlignment   string  `json:"svg_vertical_alignment"`   // "start", "center", or "end" (default "center")
}

const (
	SVGAlignStart  = "start"
	SVGAlignCenter = "center"
	SVGAlignEnd    = "end"
)

// rasterizeSVG converts SVG content to an image using pure Go (oksvg + rasterx).
// widthPx and heightPx set the target dimensions (0 means auto from SVG viewBox). scale is applied to both.
func rasterizeSVG(_ context.Context, svg string, widthPx, heightPx int, scale float64) (image.Image, error) {
	icon, err := oksvg.ReadIconStream(strings.NewReader(svg))
	if err != nil {
		return nil, fmt.Errorf("SVG parse failed: %w", err)
	}

	// Determine render dimensions: use provided size or fall back to SVG's natural size
	w := float64(widthPx)
	h := float64(heightPx)
	if w <= 0 {
		w = icon.ViewBox.W
	}
	if h <= 0 {
		h = icon.ViewBox.H
	}
	w *= scale
	h *= scale

	// Keep aspect ratio when only one dimension is constrained
	if widthPx > 0 && heightPx == 0 {
		h = w * icon.ViewBox.H / icon.ViewBox.W
	} else if heightPx > 0 && widthPx == 0 {
		w = h * icon.ViewBox.W / icon.ViewBox.H
	}

	iw, ih := int(w), int(h)
	if iw <= 0 || ih <= 0 {
		return nil, fmt.Errorf("SVG has zero dimensions (viewBox: %.0fx%.0f)", icon.ViewBox.W, icon.ViewBox.H)
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
	LabelSize              string  `json:"label_size" binding:"required"`
	SVGData                string  `json:"svg_data" binding:"required"`
	Printer                string  `json:"printer"` // Optional
	Model                  string  `json:"model"`   // Optional
	SVGScale               float64 `json:"svg_scale"`
	SVGHorizontalAlignment string  `json:"svg_horizontal_alignment"`
	SVGVerticalAlignment   string  `json:"svg_vertical_alignment"`
	CustomHeightMM         float64 `json:"custom_height_mm"`
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

	h.printImageToPrinter(c, img, label, req.Printer, req.Model)
}

// processSVG handles SVG conversion logic
func processSVG(ctx context.Context, req PrintSVGRequest, label brotherql.LabelSize) (*image.Gray, error) {
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

	var svgImg image.Image
	var err error
	var imageHeight int

	if tapeLengthDots > 0 { // die-cut or custom height: fixed
		imageHeight = tapeLengthDots
		svgImg, err = rasterizeSVG(ctx, req.SVGData, printHeadDots, imageHeight, scale)
	} else { // continuous tape, no custom height: derive from rendered SVG
		svgImg, err = rasterizeSVG(ctx, req.SVGData, printHeadDots, 0, scale)
		if err == nil {
			imageHeight = svgImg.Bounds().Dy()
		}
	}

	if err != nil {
		return nil, fmt.Errorf("SVG processing failed: %v", err)
	}

	img := brotherql.CreateBlankImage(printHeadDots, imageHeight)
	grayImg := convertToGrayscale(svgImg)

	var xPos, yPos int
	switch req.SVGHorizontalAlignment {
	case "center":
		xPos = (printHeadDots - grayImg.Bounds().Dx()) / 2
	case "end":
		xPos = printHeadDots - grayImg.Bounds().Dx()
	default:
		xPos = 0
	}

	switch req.SVGVerticalAlignment {
	case "center":
		yPos = (imageHeight - grayImg.Bounds().Dy()) / 2
	case "end":
		yPos = imageHeight - grayImg.Bounds().Dy()
	default:
		yPos = 0
	}

	draw.Draw(img,
		image.Rect(xPos, yPos, xPos+grayImg.Bounds().Dx(), yPos+grayImg.Bounds().Dy()),
		grayImg,
		image.Point{0, 0},
		draw.Over)

	return img, nil
}

// printImageToPrinter sends img to the printer identified by printer/model,
// or saves it to a debug file when printer == "file".
func (h *Handlers) printImageToPrinter(c *gin.Context, img *image.Gray, label brotherql.LabelSize, printer string, model string) {
	if printer == "file" {
		timestamp := time.Now().Format("20060102150405")
		filename := fmt.Sprintf("debug_output/label_svg_%s.png", timestamp)
		err := brotherql.SaveImageToFile(img, filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save image to file: %v", err)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "Image saved to file successfully", "filename": filename})
		return
	}

	err := services.ConnectToPrinter(h.Printers, printer, model, func(backend brotherql.Backend, model string) error {
		printerDev := brotherql.NewBrotherQL(model, backend)
		return printerDev.Print(img, label)
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Print failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "print job sent successfully"})
}
