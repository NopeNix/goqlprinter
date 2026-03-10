package api

import (
	"fmt"
	"image"
	"image/draw"
	"net/http"

	"github.com/gin-gonic/gin"

	"goqlprinter/brotherql"
	"goqlprinter/internal/services"
)

// PrintQRRequest defines the structure for printing QR codes
// @description Request body for printing QR code labels
type PrintQRRequest struct {
	Printer             string  `json:"printer"` // Optional
	Model               string  `json:"model"`   // Optional
	LabelSize           string  `json:"label_size" binding:"required"`
	Data                string  `json:"data" binding:"required"`
	CustomHeightMM      float64 `json:"custom_height_mm"`
	QRScale             float64 `json:"qr_scale"`
	Orientation         string  `json:"orientation"`
	ContentRotation     float64 `json:"content_rotation"`
	HorizontalAlignment string  `json:"horizontal_alignment"`
	VerticalAlignment   string  `json:"vertical_alignment"`
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
// processQR renders a QR code image with scale and alignment, returning the image.
func processQR(req PrintQRRequest, label brotherql.LabelSize) (*image.Gray, error) {
	isRotated := req.Orientation == "rotated"

	scale := req.QRScale
	if scale <= 0 {
		scale = 1.0
	}

	printHeadDots := label.DotsPrintableWidth

	// Determine tape length for continuous tape with custom height.
	tapeLengthDots := label.DotsPrintableHeight
	if tapeLengthDots == 0 && req.CustomHeightMM > 0 && !label.IsDieCut {
		tapeLengthDots = mmToDots(req.CustomHeightMM)
	}

	// For QR size calculation, use the effective canvas dimensions.
	effectiveWidth := printHeadDots
	effectiveHeight := tapeLengthDots
	if isRotated {
		effectiveWidth = tapeLengthDots
		effectiveHeight = printHeadDots
		if effectiveWidth == 0 {
			// Endless tape rotated: width is unconstrained, use printHeadDots for sizing.
			effectiveWidth = printHeadDots
		}
	}

	var baseQRSize int
	if effectiveHeight > 0 {
		drawableWidth := effectiveWidth - 2*defaultPadding
		drawableHeight := effectiveHeight - 2*defaultPadding
		baseQRSize = min(drawableWidth, drawableHeight)
	} else {
		baseQRSize = effectiveWidth / 2
	}

	qrSize := int(float64(baseQRSize) * scale)
	if qrSize < 21 {
		return nil, fmt.Errorf("label too small to print a QR code")
	}

	// Determine canvas dimensions based on orientation.
	var canvasWidth, canvasHeight int
	if isRotated {
		canvasHeight = printHeadDots
		if tapeLengthDots > 0 {
			canvasWidth = tapeLengthDots
		} else {
			canvasWidth = qrSize + 2*defaultPadding
		}
	} else {
		canvasWidth = printHeadDots
		if tapeLengthDots > 0 {
			canvasHeight = tapeLengthDots
		} else {
			canvasHeight = qrSize + 2*defaultPadding
		}
	}

	// Clamp QR size to canvas.
	if qrSize > canvasWidth-2*defaultPadding {
		qrSize = canvasWidth - 2*defaultPadding
	}
	if qrSize > canvasHeight-2*defaultPadding {
		qrSize = canvasHeight - 2*defaultPadding
	}

	// Calculate position based on alignment.
	var xPos int
	switch req.HorizontalAlignment {
	case "start":
		xPos = defaultPadding
	case "end":
		xPos = canvasWidth - qrSize - defaultPadding
	default:
		xPos = (canvasWidth - qrSize) / 2
	}

	var yPos int
	switch req.VerticalAlignment {
	case "start":
		yPos = defaultPadding
	case "end":
		yPos = canvasHeight - qrSize - defaultPadding
	default:
		yPos = (canvasHeight - qrSize) / 2
	}

	// Clamp to canvas bounds.
	if xPos < 0 {
		xPos = 0
	}
	if yPos < 0 {
		yPos = 0
	}
	if xPos+qrSize > canvasWidth {
		xPos = canvasWidth - qrSize
	}
	if yPos+qrSize > canvasHeight {
		yPos = canvasHeight - qrSize
	}

	img := brotherql.CreateBlankImage(canvasWidth, canvasHeight)

	err := brotherql.DrawQRCode(img, req.Data, xPos, yPos, qrSize)
	if err != nil {
		return nil, fmt.Errorf("failed to draw QR code: %v", err)
	}

	// Apply content rotation if requested.
	if req.ContentRotation == 90 || req.ContentRotation == 270 {
		// Rotate just the content area, then re-place on a clean canvas.
		contentImg := brotherql.RotateImage(img, req.ContentRotation)
		// Since QR is square and canvas may differ, create a new canvas and center.
		cW := contentImg.Bounds().Dx()
		cH := contentImg.Bounds().Dy()
		if cW != canvasWidth || cH != canvasHeight {
			newImg := brotherql.CreateBlankImage(canvasWidth, canvasHeight)
			ox := (canvasWidth - cW) / 2
			oy := (canvasHeight - cH) / 2
			draw.Draw(newImg, image.Rect(ox, oy, ox+cW, oy+cH), contentImg, contentImg.Bounds().Min, draw.Over)
			img = newImg
		} else {
			img = contentImg
		}
	}

	return img, nil
}

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

	img, err := processQR(req, label)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if req.Printer == "file" {
		saveDebugOutput(c, img, "label_qr", req.Model, req.Orientation)
		return
	}

	printImg := rotateForPrinter(img, req.Orientation)
	err = services.ConnectToPrinter(h.Printers, req.Printer, req.Model, func(backend brotherql.Backend, model string) error {
		printerDev := brotherql.NewBrotherQL(model, backend)
		return printerDev.Print(printImg, label)
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Print failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Print job sent successfully"})
}
