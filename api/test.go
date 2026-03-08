package api

import (
	"goqlprinter/brotherql"
	"goqlprinter/internal/services"
	"bytes"
	"encoding/binary"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TestRequest defines the structure for test endpoint requests
// @description Request body for test printer commands
type TestRequest struct {
	Printer string `json:"printer" binding:"required"`
}

func (h *Handlers) executeTestCommand(c *gin.Context, command []byte) {
	var req TestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Use ConnectToPrinter to handle USB connection with the Backend interface
	err := services.ConnectToPrinter(h.Printers, req.Printer, "", func(backend brotherql.Backend, model string) error {
		// Write the test command to the printer
		_, err := backend.Write(command)
		if err != nil {
			return fmt.Errorf("failed to write to printer: %w", err)
		}

		// Attempt to read the 32-byte status response from the printer.
		readBuffer := make([]byte, 32)
		n, readErr := backend.Read(readBuffer)

		response := gin.H{
			"status":        "success",
			"bytes_written": len(command),
		}

		if readErr != nil {
			// Even on error (like a timeout), we might have read some data.
			response["read_error"] = readErr.Error()
		}

		if n > 0 {
			response["bytes_read"] = n
			response["response_hex"] = fmt.Sprintf("%x", readBuffer[:n])
		} else {
			response["bytes_read"] = 0
			response["response_hex"] = ""
		}

		c.JSON(http.StatusOK, response)
		return nil
	})

	if err != nil {
		// Determine appropriate HTTP status code based on error type
		statusCode := http.StatusInternalServerError
		errorMsg := err.Error()

		switch errorMsg {
		case "printer resolution error: no printer specified and no default printer is configured or connected",
			"unsupported printer format",
			"invalid printer UID format":
			statusCode = http.StatusBadRequest
		}

		c.JSON(statusCode, gin.H{"error": errorMsg})
		return
	}
}

// TestInvalidate sends 200 null bytes to clear the printer's buffer.
// TestInvalidate godoc
// @Summary Send buffer invalidate command
// @Description Sends 200 null bytes to clear printer buffer
// @Tags test
// @Accept json
// @Produce json
// @Param request body TestRequest true "Test request parameters"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /test/invalidate [post]
func (h *Handlers) TestInvalidate(c *gin.Context) {
	command := make([]byte, 200)
	h.executeTestCommand(c, command)
}

// TestInitialize sends an initialize command to the printer.
// TestInitialize godoc
// @Summary Send initialize command
// @Description Sends ESC @ initialize command to printer
// @Tags test
// @Accept json
// @Produce json
// @Param request body TestRequest true "Test request parameters"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /test/initialize [post]
func (h *Handlers) TestInitialize(c *gin.Context) {
	command := []byte{0x1b, 0x40}
	h.executeTestCommand(c, command)
}

// TestFeed sends a print-and-feed command to the printer.
// TestFeed godoc
// @Summary Send feed command
// @Description Sends 0x1A feed command to printer
// @Tags test
// @Accept json
// @Produce json
// @Param request body TestRequest true "Test request parameters"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /test/feed [post]
func (h *Handlers) TestFeed(c *gin.Context) {
	command := []byte{0x1a}
	h.executeTestCommand(c, command)
}

// TestSetMediaAndFeed sends a sequence to set media type and then feed.
// This helps diagnose if the printer needs media context before acting.
// TestSetMediaAndFeed godoc
// @Summary Test media setting and feed
// @Description Sends sequence to set media type and feed
// @Tags test
// @Accept json
// @Produce json
// @Param request body TestRequest true "Test request parameters"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /test/set_media_and_feed [post]
func (h *Handlers) TestSetMediaAndFeed(c *gin.Context) {
	var buf bytes.Buffer
	buf.Write(bytes.Repeat([]byte{0x00}, 200)) // Invalidate
	buf.Write([]byte{0x1b, 0x40})              // Initialize
	buf.Write([]byte{0x1b, 0x69, 0x61, 0x01})  // Select ESC/P mode

	// Set Media & Quality for 62mm x 29mm die-cut tape, which is installed.
	mediaCmd := []byte{0x1b, 0x69, 0x7a} // ESC i z
	payload := make([]byte, 10)
	var validFlags byte = 0x80 | 0x40 | 0x08 | 0x04 | 0x02 // 0xCE
	payload[0] = validFlags
	payload[1] = 0x0B                                        // Media Type: Die-cut
	payload[2] = 62                                          // Media Width (mm)
	payload[3] = 29                                          // Media Height (mm)
	binary.LittleEndian.PutUint32(payload[4:8], uint32(271)) // raster lines (271 for 62x29)
	buf.Write(mediaCmd)
	buf.Write(payload)

	// Finally, the feed command
	buf.Write([]byte{0x1a}) // Feed

	h.executeTestCommand(c, buf.Bytes())
}
