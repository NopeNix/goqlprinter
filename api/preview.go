package api

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"net/http"

	"goqlprinter/brotherql"

	"github.com/gin-gonic/gin"
)

// PreviewRequest defines the structure for the preview request
type PreviewRequest struct {
	Text                   string  `json:"text"`
	LabelSize              string  `json:"label_size" binding:"required"`
	FontFamily             string  `json:"font_family"`
	FontSize               float64 `json:"font_size"`
	Orientation            string  `json:"orientation"`
	HorizontalAlignment    string  `json:"horizontal_alignment"`
	VerticalAlignment      string  `json:"vertical_alignment"`
	TextRotation           float64 `json:"text_rotation"`
	SVGData                string  `json:"svg_data"`
	SVGScale               float64 `json:"svg_scale"`
	SVGHorizontalAlignment string  `json:"svg_horizontal_alignment"`
	SVGVerticalAlignment   string  `json:"svg_vertical_alignment"`
}

// PreviewResponse defines the response structure
type PreviewResponse struct {
	Image           string `json:"image"`            // Base64 data URL
	Width           int    `json:"width"`            // Image width in pixels
	Height          int    `json:"height"`           // Image height in pixels
	PrintableWidth  int    `json:"printable_width"`  // Printable area width
	PrintableHeight int    `json:"printable_height"` // Printable area height
}

// PreviewLabel handles POST /api/preview
// PreviewLabel godoc
// @Summary Preview a label without printing
// @Description Renders a label preview and returns it as a base64-encoded PNG
// @Tags preview
// @Accept json
// @Produce json
// @Param request body PreviewRequest true "Preview request parameters"
// @Success 200 {object} PreviewResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /preview [post]
func PreviewLabel(c *gin.Context) {
	var req PreviewRequest
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

	var img *image.Gray

	if req.SVGData != "" {
		// Use SVG rendering
		svgReq := PrintSVGRequest{
			LabelSize:              req.LabelSize,
			SVGData:                req.SVGData,
			SVGScale:               req.SVGScale,
			SVGHorizontalAlignment: req.SVGHorizontalAlignment,
			SVGVerticalAlignment:   req.SVGVerticalAlignment,
		}
		img, err = processSVG(svgReq, label)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		// Use text rendering - convert PreviewRequest to PrintRequest
		printReq := PrintRequest{
			Text:                req.Text,
			LabelSize:           req.LabelSize,
			FontFamily:          req.FontFamily,
			FontSize:            req.FontSize,
			Orientation:         req.Orientation,
			HorizontalAlignment: req.HorizontalAlignment,
			VerticalAlignment:   req.VerticalAlignment,
			TextRotation:        req.TextRotation,
		}
		img, err = renderTextLabel(printReq, label)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode image"})
		return
	}

	// Create base64 data URL
	base64Img := base64.StdEncoding.EncodeToString(buf.Bytes())
	dataURL := "data:image/png;base64," + base64Img

	c.JSON(http.StatusOK, PreviewResponse{
		Image:           dataURL,
		Width:           img.Bounds().Dx(),
		Height:          img.Bounds().Dy(),
		PrintableWidth:  label.DotsPrintableWidth,
		PrintableHeight: label.DotsPrintableHeight,
	})
}
