package api

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net/http"

	"goqlprinter/brotherql"
	"goqlprinter/internal/services"

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

	err := services.ConnectToPrinter(h.Printers, req.Printer, "", func(backend brotherql.Backend, model string) error {
		_, err := backend.Write(command)
		if err != nil {
			return fmt.Errorf("failed to write to printer: %w", err)
		}

		readBuffer := make([]byte, 32)
		n, readErr := backend.Read(readBuffer)

		response := gin.H{
			"status":        "success",
			"bytes_written": len(command),
		}

		if readErr != nil {
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
	buf.Write(make([]byte, 200))              // invalidate
	buf.Write([]byte{0x1b, 0x40})             // ESC @: initialize
	buf.Write([]byte{0x1b, 0x69, 0x61, 0x01}) // ESC i a: select raster mode

	// ESC i z: set media and quality (62mm x 29mm die-cut as test fixture)
	mediaCmd := []byte{0x1b, 0x69, 0x7a}
	payload := make([]byte, 10)
	var validFlags byte = 0x80 | 0x40 | 0x08 | 0x04 | 0x02 // 0xCE
	payload[0] = validFlags
	payload[1] = 0x0B                                        // media type: die-cut
	payload[2] = 62                                          // media width (mm)
	payload[3] = 29                                          // media height (mm)
	binary.LittleEndian.PutUint32(payload[4:8], uint32(271)) // raster lines for 62x29
	buf.Write(mediaCmd)
	buf.Write(payload)

	buf.Write([]byte{0x1a}) // 0x1A: print and feed

	h.executeTestCommand(c, buf.Bytes())
}
