package api

import (
	"goqlprinter/brotherql"
	"goqlprinter/internal/services"
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
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /print_png_raw [post]
func (h *Handlers) PrintPNGRaw(c *gin.Context) {
	// Parse form data
	labelSize := c.PostForm("label_size")
	if labelSize == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "label_size is required"})
		return
	}

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file upload is required"})
		return
	}

	// Open uploaded file
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

	// Decode PNG
	img, err := png.Decode(f)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid PNG file: " + err.Error()})
		return
	}

	// Get label info
	label, err := brotherql.GetLabel(labelSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid label_size"})
		return
	}

	// Resize and center image
	wantW := label.DotsPrintableWidth
	wantH := label.DotsPrintableHeight
	if wantH == 0 {
		wantH = 300 // fallback
	}

	resized := imaging.Fit(img, wantW, wantH, imaging.Lanczos)
	grayImg := image.NewGray(image.Rect(0, 0, wantW, wantH))
	draw.Draw(grayImg, grayImg.Bounds(), image.White, image.Point{}, draw.Src)
	offsetX := (wantW - resized.Bounds().Dx()) / 2
	offsetY := (wantH - resized.Bounds().Dy()) / 2
	draw.Draw(grayImg, image.Rect(offsetX, offsetY, offsetX+resized.Bounds().Dx(), offsetY+resized.Bounds().Dy()), resized, image.Point{0, 0}, draw.Over)

	// Get printer from query param or use default
	printer := c.PostForm("printer")
	model := c.PostForm("model")

	// Handle "print to file" case separately
	if printer == "file" {
		filename := "debug_output/labelpng_raw_" + strconv.FormatInt(time.Now().UnixNano(), 10) + ".png"
		var buf bytes.Buffer
		if err := png.Encode(&buf, grayImg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "PNG encode failed"})
			return
		}
		if err := os.WriteFile(filename, buf.Bytes(), 0644); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save PNG: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "Image saved to file", "filename": filename})
		return
	}

	// Use our new USB connection helper
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
