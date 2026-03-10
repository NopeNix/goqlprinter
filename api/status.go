package api

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"goqlprinter/brotherql"
	"goqlprinter/internal/services"

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
func (h *Handlers) GetStatus(c *gin.Context) {
	var req StatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	statusCh := make(chan StatusResponse, 1)

	if err := services.ConnectToPrinter(h.Printers, req.Printer, "", func(backend brotherql.Backend, model string) error {
		type ReadTimeoutSetter interface {
			SetReadTimeout(d time.Duration)
		}

		// Drain any stale data left in kernel buffer from previous operations
		if ts, ok := backend.(ReadTimeoutSetter); ok {
			ts.SetReadTimeout(150 * time.Millisecond)
		}
		drainBuf := make([]byte, 256)
		for {
			n, _ := backend.Read(drainBuf)
			if n > 0 {
				slog.Info("Drained stale bytes before status request", "n", n, "hex", fmt.Sprintf("%x", drainBuf[:n]))
			} else {
				break
			}
		}

		// Restore normal read timeout for status response
		if ts, ok := backend.(ReadTimeoutSetter); ok {
			ts.SetReadTimeout(3 * time.Second)
		}

		// Send invalidate + status request
		var cmdBuf []byte
		cmdBuf = append(cmdBuf, make([]byte, 200)...) // invalidate buffer
		cmdBuf = append(cmdBuf, 0x1B, 0x69, 0x53)     // ESC i S: status request

		if _, err := backend.Write(cmdBuf); err != nil {
			return fmt.Errorf("failed to send status request: %w", err)
		}

		// Read the 32-byte status response, retrying on EOF
		// The usblp driver may return EOF if the printer hasn't
		// prepared its response yet.
		var allData []byte
		tmpBuf := make([]byte, 64)
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			time.Sleep(100 * time.Millisecond)
			n, err := backend.Read(tmpBuf)
			if n > 0 {
				allData = append(allData, tmpBuf[:n]...)
				break
			}
			if err != nil && !errors.Is(err, io.EOF) {
				return fmt.Errorf("failed to read status: %w", err)
			}
			// EOF = no data yet, retry
		}

		if len(allData) < 32 {
			return fmt.Errorf("no response from printer (got %d bytes)", len(allData))
		}

		status, parseErr := brotherql.ParseStatusResponse(allData[:32])
		if parseErr != nil {
			return fmt.Errorf("failed to parse status response: %w", parseErr)
		}

		slog.Info("Printer Status Report",
			"ready", status.Ready, "busy", status.Busy,
			"media_type", status.MediaType, "media_width_mm", status.MediaWidth,
			"error", status.Error,
			"raw_bytes", len(allData), "raw_hex", fmt.Sprintf("%x", allData))

		statusCh <- StatusResponse{
			Status:   status,
			RawHex:   fmt.Sprintf("%x", allData),
			RawBytes: len(allData),
		}

		return nil
	}); err != nil {
		slog.Warn("Printer status unavailable", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   err.Error(),
			"message": "Printer is temporarily unavailable (may be busy or disconnected)",
		})
		return
	}

	select {
	case status := <-statusCh:
		c.JSON(http.StatusOK, status)
	case <-time.After(1 * time.Second):
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "timeout waiting for printer response",
			"message": "Printer did not respond in time",
		})
	}
}
