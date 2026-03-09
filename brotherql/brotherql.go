package brotherql

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log/slog"
	"os"
	"time"
)

var (
	statusErrorInformation1 = map[int]string{
		0: "No media when printing",
		1: "End of media",
		2: "Tape cutter jam",
		4: "Main unit in use",
		5: "Printer turned off",
	}
	statusErrorInformation2 = map[int]string{
		0: "Replace media error",
		1: "Expansion buffer full",
		2: "Communication error",
		4: "Cover opened while printing",
		6: "Media cannot be fed",
		7: "System error",
	}
	statusMediaTypes = map[byte]string{
		0x00: "No media",
		0x0A: "Continuous length tape",
		0x0B: "Die-cut labels",
	}
	statusTypes = map[byte]string{
		0x00: "Reply to status request",
		0x01: "Printing completed",
		0x02: "Error occurred",
		0x05: "Notification",
		0x06: "Phase change",
	}
	statusPhaseTypes = map[byte]string{
		0x00: "Waiting to receive",
		0x01: "Printing state",
	}
	statusModelCodes = map[byte]string{
		0x4f: "QL-500/QL-550",
		0x32: "QL-570",
		0x35: "QL-700",
		0x36: "QL-710W",
		0x37: "QL-720NW",
		0x38: "QL-800",
		0x39: "QL-810W",
		0x41: "QL-820NWB",
	}
)

// BrotherQL is the main struct for interacting with a Brother QL printer.
type BrotherQL struct {
	backend Backend
	model   string
}

// NewBrotherQL creates a new BrotherQL instance.
func NewBrotherQL(model string, backend Backend) *BrotherQL {
	return &BrotherQL{
		model:   model,
		backend: backend,
	}
}

// flipImageHorizontally mirrors a grayscale image left-to-right as required by the Brother protocol.
func flipImageHorizontally(src *image.Gray) *image.Gray {
	bounds := src.Bounds()
	dst := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			srcX := bounds.Max.X - (x - bounds.Min.X) - 1
			dst.SetGray(x, y, src.GrayAt(srcX, y))
		}
	}
	return dst
}

// Print sends an image to the printer following the Brother protocol exactly.
func (p *BrotherQL) Print(img image.Image, label LabelSize) error {
	slog.Debug("=== Starting Print Function ===")
	slog.Debug("Printer model", "model", p.model)
	slog.Debug("Label", "name", label.Name, "width_mm", label.TapeSizeWidth, "height_mm", label.TapeSizeHeight)

	model, err := GetModel(p.model)
	if err != nil {
		// Even though GetModel returns defaults, log the error
		slog.Warn("Model warning", "error", err)
	}

	bounds := img.Bounds()
	height := bounds.Max.Y

	slog.Debug("Image dimensions", "width", bounds.Dx(), "height", bounds.Dy())

	// Phase 0: reset the printer and drain stale bytes.
	if err := p.resetAndDrain(model); err != nil {
		return err
	}

	// The printer expects raster data for the full physical print-head width, not just
	// the printable area. Compositing the image onto a white canvas of the correct
	// raster width adds the required blank margins and prevents image stretching.
	rasterWidthPixels := model.RasterWidthBytes * 8
	slog.Debug("Raster width", "bytes", model.RasterWidthBytes, "pixels", rasterWidthPixels)

	fullWidthImg := CreateBlankImage(rasterWidthPixels, height)

	offsetX := (rasterWidthPixels - bounds.Dx()) / 2
	offset := image.Point{X: offsetX, Y: 0}
	drawRect := bounds.Add(offset)

	draw.Draw(fullWidthImg, drawRect, img, bounds.Min, draw.Src)

	// The protocol requires raster data to be mirrored horizontally.
	flippedImg := flipImageHorizontally(fullWidthImg)

	// Phase 1: build the command stream.
	cmdBuf, err := p.buildCommandStream(flippedImg, label, model, height)
	if err != nil {
		return err
	}

	// Phase 2: send commands and verify status.
	return p.sendAndVerify(cmdBuf.Bytes())
}

// resetAndDrain sends invalidate + ESC @ to clear any error state from a
// previous failed print, then drains any stale status bytes the printer queued.
func (p *BrotherQL) resetAndDrain(model PrinterModel) error {
	var resetBuf bytes.Buffer
	resetBuf.Write(bytes.Repeat([]byte{0x00}, model.InvalidateBytes))
	resetBuf.Write([]byte{0x1B, 0x40}) // ESC @: initialize
	if _, err := p.backend.Write(resetBuf.Bytes()); err != nil {
		return fmt.Errorf("failed to send reset: %w", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Use a short read timeout while draining so we don't wait unnecessarily.
	type ReadTimeoutSetter interface {
		SetReadTimeout(d time.Duration)
	}
	if ts, ok := p.backend.(ReadTimeoutSetter); ok {
		ts.SetReadTimeout(300 * time.Millisecond)
	}
	discardBuf := make([]byte, 64)
	n, _ := p.backend.Read(discardBuf)
	if n > 0 {
		slog.Debug("Discarded stale bytes from printer after reset", "n", n)
	}
	if ts, ok := p.backend.(ReadTimeoutSetter); ok {
		ts.SetReadTimeout(3 * time.Second)
	}

	return nil
}

// buildCommandStream assembles the Brother QL command buffer: mode switch,
// media/quality settings, margins, auto-cut, compression, and raster rows.
func (p *BrotherQL) buildCommandStream(flippedImg *image.Gray, label LabelSize, model PrinterModel, height int) (*bytes.Buffer, error) {
	rasterWidthPixels := model.RasterWidthBytes * 8
	rasterData := p.rasterize(flippedImg, model.RasterWidthBytes, rasterWidthPixels)
	slog.Debug("Rasterized rows of data", "rows", len(rasterData))

	var buf bytes.Buffer

	// Switch to raster mode (not supported on all models).
	if model.SupportsSwitchMode {
		buf.Write([]byte{0x1B, 0x69, 0x61, 0x01})
	}

	// ESC i z: set media and quality
	buf.Write([]byte{0x1B, 0x69, 0x7A})
	payload := make([]byte, 10)

	var flags byte = 0x80 | (1 << 3) | (1 << 2) | (1 << 1)
	if model.NeedsQualitySetting {
		flags |= (1 << 6) // standard quality (300x300 dpi)
	}

	payload[0] = flags

	// DotsPrintableHeight > 0 indicates a die-cut label.
	isDieCut := label.DotsPrintableHeight > 0

	if isDieCut {
		payload[1] = 0x0B // die-cut
		payload[3] = byte(label.TapeSizeHeight)
		slog.Debug("Media type: Die-cut (0x0B)", "width_mm", label.TapeSizeWidth, "height_mm", label.TapeSizeHeight)
	} else {
		payload[1] = 0x0A // continuous
		payload[3] = 0    // length must be 0 for continuous tape
		slog.Debug("Media type: Continuous (0x0A)", "width_mm", label.TapeSizeWidth)
	}
	payload[2] = byte(label.TapeSizeWidth)

	binary.LittleEndian.PutUint32(payload[4:8], uint32(height)) // image height in pixels
	slog.Debug("Print height", "pixels", height)
	buf.Write(payload)

	// ESC i d: set feed margin using label-specific value.
	buf.Write([]byte{0x1B, 0x69, 0x64})
	marginPayload := make([]byte, 2)
	binary.LittleEndian.PutUint16(marginPayload, uint16(label.FeedMargin))
	buf.Write(marginPayload)
	slog.Debug("Feed margin", "dots", label.FeedMargin)

	// ESC i M: enable auto-cut; ESC i K: set expanded mode (0x08, not 0x40 high-res).
	buf.Write([]byte{0x1B, 0x69, 0x4D, 0x40})
	buf.Write([]byte{0x1B, 0x69, 0x4B, 0x08})

	useCompression := model.SupportsCompression
	if useCompression {
		buf.Write([]byte{0x4D, 0x02}) // PackBits compression on
	} else {
		buf.Write([]byte{0x4D, 0x00}) // compression off
	}

	for _, row := range rasterData {
		var dataToSend []byte
		if useCompression {
			dataToSend = packBits(row)
		} else {
			dataToSend = row
		}

		length := len(dataToSend)
		if length > 255 {
			return nil, fmt.Errorf("raster row is too long (%d bytes) for a single-byte length", length)
		}

		buf.Write([]byte{0x67, 0x00}) // 'g': raster data transfer
		buf.WriteByte(byte(length))
		buf.Write(dataToSend)
	}

	buf.WriteByte(0x1A) // 0x1A: print and feed

	return &buf, nil
}

// sendAndVerify writes the command data to the printer and checks the status response.
func (p *BrotherQL) sendAndVerify(data []byte) error {
	slog.Debug("Sending bytes to printer", "total_bytes", len(data))
	_, err := p.backend.Write(data)
	if err != nil {
		slog.Error("Failed to write to printer", "error", err)
		return err
	}
	slog.Debug("Print command sent successfully")

	status, err := p.RequestStatus()
	if err != nil {
		slog.Warn("Failed to read printer status", "error", err)
	} else {
		slog.Debug("Printer status", "ready", status.Ready, "busy", status.Busy, "error", status.Error)
		slog.Debug("Media info", "type", status.MediaType, "width_mm", status.MediaWidth)
		if status.Error != "" {
			slog.Error("PRINTER ERROR", "error", status.Error)
			return fmt.Errorf("printer reported error: %s", status.Error)
		}
		slog.Debug("No printer errors reported")
	}

	return nil
}

// RequestStatus sends a status request to the printer and returns the parsed response.
func (p *BrotherQL) RequestStatus() (PrinterStatus, error) {
	time.Sleep(200 * time.Millisecond)

	statusCmd := []byte{0x1B, 0x69, 0x53} // ESC i S: status request
	_, err := p.backend.Write(statusCmd)
	if err != nil {
		return PrinterStatus{}, fmt.Errorf("failed to write status request: %w", err)
	}

	// Accumulate data across retries in case the response arrives in chunks.
	var allData []byte
	buf := make([]byte, 64)
	maxRetries := 5

	for i := range maxRetries {
		time.Sleep(100 * time.Millisecond)
		n, readErr := p.backend.Read(buf)

		if n > 0 {
			allData = append(allData, buf[:n]...)
			if len(allData) >= 32 {
				slog.Debug("Status read successful", "attempt", i+1, "max", maxRetries, "bytes", len(allData))
				break
			}
		}

		if readErr != nil {
			slog.Debug("Status read attempt failed", "attempt", i+1, "max", maxRetries, "error", readErr)
			continue
		}

		slog.Debug("Status read attempt", "attempt", i+1, "max", maxRetries, "bytes_total", len(allData), "expected", 32)
	}

	if len(allData) < 32 {
		return PrinterStatus{}, fmt.Errorf("incomplete status response after %d retries: got %d bytes, expected 32", maxRetries, len(allData))
	}

	return ParseStatusResponse(allData[:32])
}

// packBits implements the PackBits run-length compression algorithm used by the Brother protocol.
func packBits(data []byte) []byte {
	var compressed bytes.Buffer
	i := 0
	dataLen := len(data)
	for i < dataLen {
		runLen := 1
		for i+runLen < dataLen && data[i] == data[i+runLen] && runLen < 128 {
			runLen++
		}

		if runLen > 1 {
			// Repeat run: control byte is -(runLen - 1).
			compressed.WriteByte(byte(1 - runLen))
			compressed.WriteByte(data[i])
			i += runLen
		} else {
			// Literal run: find its end.
			literalEnd := i
			for literalEnd < dataLen {
				if literalEnd+1 < dataLen && data[literalEnd] == data[literalEnd+1] {
					break
				}
				literalEnd++
				if literalEnd-i == 128 {
					break
				}
			}
			literalLen := literalEnd - i
			// Control byte is (literalLen - 1).
			compressed.WriteByte(byte(literalLen - 1))
			compressed.Write(data[i:literalEnd])
			i = literalEnd
		}
	}
	return compressed.Bytes()
}

// rasterize converts an image to the printer's raster format.
// bytesPerRow is the fixed row width required by the printer model.
// printableWidth limits how many pixels are sampled per row.
func (p *BrotherQL) rasterize(img image.Image, bytesPerRow int, printableWidth int) [][]byte {
	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	var rasterData [][]byte

	grayImg, ok := img.(*image.Gray)
	if !ok {
		// Convert to grayscale if the caller passed a non-Gray image.
		newGray := image.NewGray(bounds)
		draw.Draw(newGray, bounds, img, bounds.Min, draw.Src)
		grayImg = newGray
	}

	for y := range height {
		rowData := make([]byte, bytesPerRow)

		loopWidth := min(width, printableWidth)

		for x := range loopWidth {
			// Protocol: black = bit 1, white = bit 0.
			// image.Gray: black = 0, white = 255.
			// Any pixel that is not fully white is printed as black.
			if grayImg.GrayAt(x, y).Y < 250 {
				byteIndex := x / 8
				bitIndex := 7 - (x % 8)
				rowData[byteIndex] |= 1 << bitIndex
			}
		}
		rasterData = append(rasterData, rowData)
	}
	return rasterData
}

// SaveImageToFile saves an image to a PNG file for debugging.
func SaveImageToFile(img image.Image, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			slog.Warn("failed to close file", "error", cerr)
		}
	}()

	return png.Encode(f, img)
}

// ParseStatusResponse converts raw status response bytes into a PrinterStatus struct
func ParseStatusResponse(data []byte) (PrinterStatus, error) {
	if len(data) < 32 {
		return PrinterStatus{Errors: []string{}}, fmt.Errorf("insufficient status data: got %d bytes, expected 32", len(data))
	}
	if data[0] != 0x80 || data[2] != 0x42 {
		return PrinterStatus{Errors: []string{}}, fmt.Errorf("invalid status header: %x", data[0:3])
	}

	status := PrinterStatus{
		Errors: []string{}, // always a slice, never nil, so the frontend can safely iterate
	}

	if modelName, ok := statusModelCodes[data[4]]; ok {
		status.ModelName = modelName
	} else {
		status.ModelName = fmt.Sprintf("Unknown (0x%02x)", data[4])
	}

	errorByte1 := data[8]
	errorByte2 := data[9]

	for i := range 8 {
		if (errorByte1 & (1 << i)) != 0 {
			if errMsg, ok := statusErrorInformation1[i]; ok {
				status.Errors = append(status.Errors, errMsg)
			}
		}
		if (errorByte2 & (1 << i)) != 0 {
			if errMsg, ok := statusErrorInformation2[i]; ok {
				status.Errors = append(status.Errors, errMsg)
			}
		}
	}

	if len(status.Errors) > 0 {
		status.Error = status.Errors[0]
		status.Ready = false
		status.Busy = false
	} else {
		status.Error = ""
		phaseType := data[19]
		switch phaseType {
		case 0x00:
			status.Ready = true
			status.Busy = false
		case 0x01:
			status.Ready = false
			status.Busy = true
		default:
			status.Ready = true
			status.Busy = false
		}
	}

	status.MediaWidth = int(data[10])
	status.MediaLength = int(data[17])

	if mediaType, ok := statusMediaTypes[data[11]]; ok {
		status.MediaType = mediaType
	}

	if st, ok := statusTypes[data[18]]; ok {
		status.StatusType = st
	}
	if pt, ok := statusPhaseTypes[data[19]]; ok {
		status.PhaseType = pt
	}

	slog.Debug("Parsed status",
		"model", status.ModelName, "ready", status.Ready, "busy", status.Busy,
		"errors", status.Errors, "media_type", status.MediaType,
		"media_width", status.MediaWidth, "media_length", status.MediaLength)

	return status, nil
}
