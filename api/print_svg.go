package api

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"net/http"
	"os/exec"
	"time"

	"goqlprinter/brotherql"
	"goqlprinter/internal/services"
	"github.com/gin-gonic/gin"
)

// SVGPrintParams defines the parameters for SVG printing
// @description Parameters used for controlling SVG printing
type SVGPrintParams struct {
	SVGData                string  `json:"svg_data"`                 // SVG content to print
	SVGScale               float64 `json:"svg_scale"`                // Scaling factor for SVG (default 1.0)
	SVGHorizontalAlignment string  `json:"svg_horizontal_alignment"` // "start", "center", or "end" (default "center")
	SVGVerticalAlignment   string  `json:"svg_vertical_alignment"`   // "start", "center", or "end" (default "center")
}

// @name SVGPrintAlignment
// @enum SVG alignment options
const (
	SVGAlignStart  = "start"
	SVGAlignCenter = "center"
	SVGAlignEnd    = "end"
)

// rasterizeSVG converts SVG to PNG image
// @param svg The SVG content to rasterize
// @param widthPx The target width in pixels (0 for auto)
// @param heightPx The target height in pixels (0 for auto)
// @param scale Scaling factor to apply
// @return image.Image The rasterized image
// @return error Any error that occurred

func rasterizeSVG(svg string, widthPx, heightPx int, scale float64) (image.Image, error) {
	args := []string{
		"--format", "png",
		"--background-color", "white",
		"--keep-aspect-ratio", // Always keep aspect ratio to prevent distortion.
	}

	if widthPx > 0 {
		scaledWidth := int(float64(widthPx) * scale)
		args = append(args, "--width", fmt.Sprintf("%d", scaledWidth))
	}
	if heightPx > 0 {
		scaledHeight := int(float64(heightPx) * scale)
		args = append(args, "--height", fmt.Sprintf("%d", scaledHeight))
	}

	cmd := exec.Command("rsvg-convert", args...)
	cmd.Stdin = bytes.NewBufferString(svg)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("SVG rasterization failed: %w. Varmista, että 'rsvg-convert' on asennettu ja PATH-ympäristömuuttujassa", err)
	}
	return png.Decode(&out)
}

func convertToGrayscale(img image.Image) *image.Gray {
	bounds := img.Bounds()
	gray := image.NewGray(bounds)
	draw.Draw(gray, bounds, img, bounds.Min, draw.Src)
	return gray
}

// processSVG handles SVG conversion logic
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

	// Get label info
	label, err := brotherql.GetLabel(req.LabelSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label size"})
		return
	}

	// Convert SVG to printable image
	img, err := processSVG(req, label)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Handle printing
	h.printImageToPrinter(c, img, label, req.Printer, req.Model)
}

// processSVG handles SVG conversion logic
func processSVG(req PrintSVGRequest, label brotherql.LabelSize) (*image.Gray, error) {
	scale := req.SVGScale
	if scale <= 0 {
		scale = 1.0
	}

	var svgImg image.Image
	var err error
	var imageHeight int

	if label.DotsPrintableHeight > 0 { // Die-cut label
		imageHeight = label.DotsPrintableHeight
		svgImg, err = rasterizeSVG(req.SVGData, label.DotsPrintableWidth, imageHeight, scale)
	} else { // Continuous tape
		svgImg, err = rasterizeSVG(req.SVGData, label.DotsPrintableWidth, 0, scale)
		if err == nil {
			imageHeight = svgImg.Bounds().Dy()
		}
	}

	if err != nil {
		return nil, fmt.Errorf("SVG processing failed: %v", err)
	}

	img := brotherql.CreateBlankImage(label.DotsPrintableWidth, imageHeight)
	grayImg := convertToGrayscale(svgImg)

	var xPos, yPos int
	switch req.SVGHorizontalAlignment {
	case "center":
		xPos = (label.DotsPrintableWidth - grayImg.Bounds().Dx()) / 2
	case "end":
		xPos = label.DotsPrintableWidth - grayImg.Bounds().Dx()
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

// printImageToPrinter handles common printing logic
func (h *Handlers) printImageToPrinter(c *gin.Context, img *image.Gray, label brotherql.LabelSize, printer string, model string) {
	// Handle "print to file" case separately
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

	// Use our new USB connection helper
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
