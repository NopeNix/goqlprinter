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
	SVGData  string  `json:"svg_data"`
	SVGScale float64 `json:"svg_scale"`
	CustomHeightMM         float64 `json:"custom_height_mm"`
}

// PreviewResponse defines the response structure
type PreviewResponse struct {
	Image           string `json:"image"`            // Base64 data URL
	Width           int    `json:"width"`            // Image width in pixels
	Height          int    `json:"height"`           // Image height in pixels
	PrintableWidth  int    `json:"printable_width"`  // Printable area width
	PrintableHeight int    `json:"printable_height"` // Printable area height
}

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
func (h *Handlers) PreviewLabel(c *gin.Context) {
	var req PreviewRequest
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
		svgReq := PrintSVGRequest{
			LabelSize:           req.LabelSize,
			SVGData:             req.SVGData,
			SVGScale:            req.SVGScale,
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
		printReq := PrintRequest{
			Text:                req.Text,
			LabelSize:           req.LabelSize,
			FontFamily:          req.FontFamily,
			FontSize:            req.FontSize,
			Orientation:         req.Orientation,
			HorizontalAlignment: req.HorizontalAlignment,
			VerticalAlignment:   req.VerticalAlignment,
			TextRotation:        req.TextRotation,
			CustomHeightMM:      req.CustomHeightMM,
		}
		img, err = h.renderTextLabel(printReq, label)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode image"})
		return
	}

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
