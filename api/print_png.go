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
	Printer        string  `json:"printer"` // Optional
	Model          string  `json:"model"`   // Optional
	LabelSize      string  `json:"label_size" binding:"required"`
	PNGData        string  `json:"png_data" binding:"required"`
	CustomHeightMM float64 `json:"custom_height_mm"`
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

	pngBytes, err := base64.StdEncoding.DecodeString(payload.PNGData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid base64 PNG data: " + err.Error()})
		return
	}

	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid PNG file: " + err.Error()})
		return
	}

	printer := payload.Printer
	model := payload.Model
	labelSize := payload.LabelSize

	label, err := brotherql.GetLabel(labelSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label_size"})
		return
	}

	// Scale and center the PNG image within the printable area.
	wantW := label.DotsPrintableWidth
	wantH := label.DotsPrintableHeight
	if wantH == 0 {
		if payload.CustomHeightMM > 0 {
			wantH = mmToDots(payload.CustomHeightMM)
		} else {
			// Proportional scaling from source image aspect ratio.
			srcBounds := img.Bounds()
			if srcBounds.Dx() > 0 {
				wantH = srcBounds.Dy() * wantW / srcBounds.Dx()
			}
			if wantH == 0 {
				wantH = 300 // ultimate fallback
			}
		}
	}

	resized := imaging.Fit(img, wantW, wantH, imaging.Lanczos)
	grayImg := image.NewGray(image.Rect(0, 0, wantW, wantH))
	draw.Draw(grayImg, grayImg.Bounds(), image.White, image.Point{}, draw.Src)
	offsetX := (wantW - resized.Bounds().Dx()) / 2
	offsetY := (wantH - resized.Bounds().Dy()) / 2
	draw.Draw(grayImg, image.Rect(offsetX, offsetY, offsetX+resized.Bounds().Dx(), offsetY+resized.Bounds().Dy()), resized, image.Point{0, 0}, draw.Over)

	if printer == "file" {
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

	err = services.ConnectToPrinter(h.Printers, printer, model, func(backend brotherql.Backend, model string) error {
		printerDev := brotherql.NewBrotherQL(model, backend)
		return printerDev.Print(grayImg, label)
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Print failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Print job sent successfully"})
}

func writeFile(filename string, data []byte) error {
	err := os.MkdirAll("debug_output", 0750)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0600)
}
