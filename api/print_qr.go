package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"goqlprinter/brotherql"
	"goqlprinter/internal/services"
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

	label, err := brotherql.GetLabel(req.LabelSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label size"})
		return
	}

	padding := 10
	var imageHeight int
	var qrSize int

	if label.DotsPrintableHeight > 0 {
		// Die-cut: fit the QR code within the printable area.
		imageHeight = label.DotsPrintableHeight
		drawableWidth := label.DotsPrintableWidth - 2*padding
		drawableHeight := imageHeight - 2*padding
		qrSize = min(drawableWidth, drawableHeight)
	} else {
		// Continuous tape: use half the label width (QR is square).
		qrSize = label.DotsPrintableWidth / 2
	}

	if qrSize < 21 { // minimum valid QR code size
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Label too small to print a QR code"})
		return
	}

	img := brotherql.CreateBlankImage(qrSize+2*padding, qrSize+2*padding)

	err = brotherql.DrawQRCode(img, req.Data, padding, padding, qrSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to draw QR code: %v", err)})
		return
	}

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
