package api

import (
	"fmt"
	"net/http"
	"time"

	"goqlprinter/brotherql"
	"goqlprinter/internal/services"
	"github.com/gin-gonic/gin"
)

// PrintQRRequest defines the structure for printing QR codes
// @description Request body for printing QR code labels
type PrintQRRequest struct {
	Printer   string `json:"printer"` // Optional
	Model     string `json:"model"`   // Optional
	LabelSize string `json:"label_size" binding:"required"`
	Data      string `json:"data" binding:"required"`
}

// PrintQR godoc
// @Summary Print QR code label
// @Description Prints a label containing a QR code with the specified data
// @Tags print
// @Accept json
// @Produce json
// @Param request body PrintQRRequest true "Print request parameters"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /print_qr [post]
func (h *Handlers) PrintQR(c *gin.Context) {
	var req PrintQRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Get label info
	label, err := brotherql.GetLabel(req.LabelSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label size"})
		return
	}

	// 2. Calculate image and QR code dimensions
	padding := 10
	var imageHeight int
	var qrSize int

	if label.DotsPrintableHeight > 0 {
		// For die-cut labels, fit the QR code within the printable area.
		imageHeight = label.DotsPrintableHeight
		drawableWidth := label.DotsPrintableWidth - 2*padding
		drawableHeight := imageHeight - 2*padding
		qrSize = min(drawableWidth, drawableHeight)
	} else {
		// For continuous tape, user wants QR code to be half of the label width.
		// Since it's a square, height is the same as width.
		qrSize = label.DotsPrintableWidth / 2
	}

	if qrSize < 21 { // QR codes have a minimum size
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Label too small to print a QR code"})
		return
	}

	// 3. Create image just large enough for QR code with padding
	img := brotherql.CreateBlankImage(qrSize+2*padding, qrSize+2*padding)

	// Draw QR code centered in the image
	err = brotherql.DrawQRCode(img, req.Data, padding, padding, qrSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to draw QR code: %v", err)})
		return
	}

	// Handle "print to file" case separately
	if req.Printer == "file" {
		timestamp := time.Now().Format("20060102150405")
		filename := fmt.Sprintf("debug_output/label_qr_%s.png", timestamp)
		err := brotherql.SaveImageToFile(img, filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save image to file: %v", err)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "Image saved to file successfully", "filename": filename})
		return
	}

	// Use our new USB connection helper
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
