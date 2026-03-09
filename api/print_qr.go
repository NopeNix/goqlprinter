package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"goqlprinter/brotherql"
	"goqlprinter/internal/services"
)

// PrintQRRequest defines the structure for printing QR codes
// @description Request body for printing QR code labels
type PrintQRRequest struct {
	Printer        string  `json:"printer"` // Optional
	Model          string  `json:"model"`   // Optional
	LabelSize      string  `json:"label_size" binding:"required"`
	Data           string  `json:"data" binding:"required"`
	CustomHeightMM float64 `json:"custom_height_mm"`
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

	var qrSize int

	// Determine tape length for continuous tape with custom height.
	tapeLengthDots := label.DotsPrintableHeight
	if tapeLengthDots == 0 && req.CustomHeightMM > 0 && !label.IsDieCut {
		tapeLengthDots = mmToDots(req.CustomHeightMM)
	}

	if tapeLengthDots > 0 {
		// Die-cut or custom height: fit the QR code within the printable area.
		drawableWidth := label.DotsPrintableWidth - 2*defaultPadding
		drawableHeight := tapeLengthDots - 2*defaultPadding
		qrSize = min(drawableWidth, drawableHeight)
	} else {
		// Continuous tape, no custom height: use half the label width (QR is square).
		qrSize = label.DotsPrintableWidth / 2
	}

	if qrSize < 21 { // minimum valid QR code size
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Label too small to print a QR code"})
		return
	}

	canvasWidth := label.DotsPrintableWidth
	var canvasHeight int
	if tapeLengthDots > 0 {
		canvasHeight = tapeLengthDots
	} else {
		canvasHeight = qrSize + 2*defaultPadding
	}

	xPos := (canvasWidth - qrSize) / 2
	yPos := (canvasHeight - qrSize) / 2

	img := brotherql.CreateBlankImage(canvasWidth, canvasHeight)

	err = brotherql.DrawQRCode(img, req.Data, xPos, yPos, qrSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to draw QR code: %v", err)})
		return
	}

	if req.Printer == "file" {
		saveDebugOutput(c, img, "label_qr", req.Model)
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
