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

// PrintRequest defines the structure for the print request
// @description Request body for printing labels with text or SVG content
type PrintRequest struct {
	Text                   string  `json:"text"`
	LabelSize              string  `json:"label_size" binding:"required"`
	FontFamily             string  `json:"font_family"`
	FontSize               float64 `json:"font_size"`
	Printer                string  `json:"printer"` // e.g., "QL-800", "usb:001:005" or "file" (optional)
	Model                  string  `json:"model"`   // e.g., "QL-800" (optional)
	Orientation            string  `json:"orientation"`
	HorizontalAlignment    string  `json:"horizontal_alignment"`
	VerticalAlignment      string  `json:"vertical_alignment"`
	TextRotation           float64 `json:"text_rotation"`
	SVGData                string  `json:"svg_data"`
	SVGScale               float64 `json:"svg_scale"`
	SVGHorizontalAlignment string  `json:"svg_horizontal_alignment"`
	SVGVerticalAlignment   string  `json:"svg_vertical_alignment"`
}

// renderTextLabel renders text onto a grayscale image sized for the given label.
func (h *Handlers) renderTextLabel(req PrintRequest, label brotherql.LabelSize) (*image.Gray, error) {
	padding := 10

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

	// Die-cut labels have a fixed height; continuous tape is sized to content.
	var imageHeight int
	if label.DotsPrintableHeight > 0 {
		imageHeight = label.DotsPrintableHeight
	} else {
		imageHeight = textBoundsHeight + (2 * padding)
	}

	img := brotherql.CreateBlankImage(label.DotsPrintableWidth, imageHeight)

	var x int
	switch req.HorizontalAlignment {
	case "start":
		x = padding
	case "center":
		x = (label.DotsPrintableWidth - textBoundsWidth) / 2
	case "end":
		x = label.DotsPrintableWidth - textBoundsWidth - padding
	default:
		x = padding
	}

	var y int
	switch req.VerticalAlignment {
	case "start":
		y = padding
	case "center":
		y = (imageHeight - textBoundsHeight) / 2
	case "end":
		y = imageHeight - textBoundsHeight - padding
	default:
		y = padding
	}

	err = brotherql.DrawText(img, req.Text, fontPath, scaledFontSize, x, y, req.TextRotation)
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
			LabelSize:              req.LabelSize,
			SVGData:                req.SVGData,
			Printer:                req.Printer,
			Model:                  req.Model,
			SVGScale:               req.SVGScale,
			SVGHorizontalAlignment: req.SVGHorizontalAlignment,
			SVGVerticalAlignment:   req.SVGVerticalAlignment,
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

	if req.Printer == "file" {
		timestamp := time.Now().Format("20060102150405")
		filename := fmt.Sprintf("debug_output/label_%s.png", timestamp)
		err := brotherql.SaveImageToFile(img, filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save image to file: %v", err)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "Image saved to file successfully", "filename": filename})
		return
	}

	err = services.ConnectToPrinter(h.Printers, req.Printer, req.Model, func(backend brotherql.Backend, model string) error {
		printerDev := brotherql.NewBrotherQL(model, backend)
		return printerDev.Print(img, label)
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Print failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Print job sent successfully"})
}
