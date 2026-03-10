package api

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"goqlprinter/brotherql"
	"goqlprinter/internal/services"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
)

// PrintPNGRaw godoc
// @Summary Print from PNG file upload
// @Description Handles PNG file upload via form-data and prints it
// @Tags print
// @Accept multipart/form-data
// @Produce json
// @Param label_size formData string true "Label size ID"
// @Param file formData file true "PNG file to print"
// @Param printer formData string false "Printer identifier (optional)"
// @Param model formData string false "Printer model override (optional)"
// @Param custom_height_mm formData number false "Custom tape length in mm (endless tape only)"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /print_png_raw [post]
func (h *Handlers) PrintPNGRaw(c *gin.Context) {
	labelSize := c.PostForm("label_size")
	if labelSize == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "label_size is required"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file upload is required"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open uploaded file"})
		return
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			slog.Warn("failed to close file", "error", cerr)
		}
	}()

	img, err := png.Decode(f)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid PNG file: " + err.Error()})
		return
	}

	label, err := brotherql.GetLabel(labelSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid label_size"})
		return
	}

	// Parse optional custom_height_mm from form data.
	var customHeightMM float64
	if v := c.PostForm("custom_height_mm"); v != "" {
		customHeightMM, _ = strconv.ParseFloat(v, 64)
	}

	// Scale and center the PNG image within the printable area.
	wantW := label.DotsPrintableWidth
	wantH := label.DotsPrintableHeight
	if wantH == 0 {
		if customHeightMM > 0 {
			wantH = mmToDots(customHeightMM)
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

	printer := c.PostForm("printer")
	model := c.PostForm("model")

	if printer == printerFile {
		filename := "debug_output/labelpng_raw_" + strconv.FormatInt(time.Now().UnixNano(), 10) + ".png"
		var buf bytes.Buffer
		if err := png.Encode(&buf, grayImg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "PNG encode failed"})
			return
		}
		if err := os.WriteFile(filename, buf.Bytes(), 0600); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save PNG: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "Image saved to file", "filename": filename})
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

	c.JSON(http.StatusOK, gin.H{"status": "print job sent successfully"})
}
