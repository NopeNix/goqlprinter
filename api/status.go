package api

import (
	"goqlprinter/brotherql"
	"goqlprinter/services"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// StatusRequest defines the expected JSON body for the /api/status endpoint
// @description Request body for getting printer status
type StatusRequest struct {
	Printer string `json:"printer" example:"usb:001:005"` // Optional printer identifier. If empty, uses default printer.
}

// StatusResponse defines the response structure for the status endpoint
type StatusResponse struct {
	Status   brotherql.PrinterStatus `json:"status"`    // Parsed status information
	RawHex   string                  `json:"raw_hex"`   // Full raw response in hexadecimal
	RawBytes int                     `json:"raw_bytes"` // Number of raw bytes received
}

// GetStatus godoc
// @Summary Get printer status information
// @Description Returns detailed printer status including model, media information, errors and raw response data.
// @Description The printer will be queried for its current state and may return multiple status packets.
// @Tags printer
// @Accept json
// @Produce json
// @Param request body StatusRequest true "Optional printer specification"
// @Success 200 {object} StatusResponse
// @Failure 400 {object} map[string]string "Invalid request or printer not found"
// @Failure 500 {object} map[string]string "Printer communication error"
// @Router /status [post]
func GetStatus(c *gin.Context) {
	services.PrinterLock.Lock()
	defer services.PrinterLock.Unlock()
	var req StatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create a channel to receive the status response
	statusCh := make(chan StatusResponse, 1)

	// Use our new USB connection helper with custom handler for status requests
	if err := services.ConnectToPrinter(req.Printer, "", func(backend brotherql.Backend, model string) error {
		// Build status request command sequence
		var cmdBuf []byte
		cmdBuf = append(cmdBuf, bytes.Repeat([]byte{0x00}, 200)...) // Invalidate buffer
		cmdBuf = append(cmdBuf, 0x1B, 0x69, 0x53)                   // Status Information Request (ESC i S)

		// Send command to printer
		if _, err := backend.Write(cmdBuf); err != nil {
			return fmt.Errorf("failed to send status request: %w", err)
		}

		// Wait for printer to respond
		time.Sleep(100 * time.Millisecond)

		// Read all available responses (printer may send multiple status packets)
		var allData []byte
		tmpBuf := make([]byte, 64)

		// Read with timeout - try multiple times if needed
		timeout := time.Now().Add(150 * time.Millisecond)
		for time.Now().Before(timeout) {
			n, readErr := backend.Read(tmpBuf)
			if n > 0 {
				allData = append(allData, tmpBuf[:n]...)
				// Status response is exactly 32 bytes - stop when we have enough
				if len(allData) >= 32 {
					break
				}
			} else if readErr != nil {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}

		if len(allData) < 32 {
			return fmt.Errorf("no response from printer")
		}

		// Parse the first 32 bytes for status
		status, parseErr := brotherql.ParseStatusResponse(allData[:32])
		if parseErr != nil {
			return fmt.Errorf("failed to parse status response: %w", parseErr)
		}

		// Log status to console
		log.Printf("--- Printer Status Report ---")
		log.Printf("Ready: %v, Busy: %v", status.Ready, status.Busy)
		log.Printf("Media Type: %s, Width: %dmm",
			status.MediaType, status.MediaWidth)
		if status.Error != "" {
			log.Printf("Error: %s", status.Error)
		} else {
			log.Println("Error: None")
		}
		log.Printf("Raw data (%d bytes): %x", len(allData), allData)
		log.Printf("-----------------------------")

		// Send status via channel
		statusCh <- StatusResponse{
			Status:   status,
			RawHex:   fmt.Sprintf("%x", allData),
			RawBytes: len(allData),
		}

		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get printer status: %v", err)})
		return
	}

	// Wait for status response with timeout
	select {
	case status := <-statusCh:
		c.JSON(http.StatusOK, status)
	case <-time.After(1 * time.Second):
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "Timeout waiting for printer status"})
	}
}
