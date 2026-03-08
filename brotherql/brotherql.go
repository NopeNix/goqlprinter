package brotherql

import (
	"goqlprinter/logger"
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/draw"
	"image/png"
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

// BrotherQL is the main struct for interacting with a printer.
type BrotherQL struct {
	backend Backend
	model   string // e.g. "QL-800"
}

// NewBrotherQL creates a new BrotherQL instance.
func NewBrotherQL(model string, backend Backend) *BrotherQL {
	return &BrotherQL{
		model:   model,
		backend: backend,
	}
}

// flipImageHorizontally kääntää kuvan peilikuvaksi, kuten protokolla vaatii.
func flipImageHorizontally(src image.Image) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Lähdepikseli otetaan vastakkaiselta puolelta
			srcX := bounds.Max.X - (x - bounds.Min.X) - 1
			dst.Set(x, y, src.At(srcX, y))
		}
	}
	return dst
}

// Print sends an image to the printer following the Brother protocol exactly.
func (p *BrotherQL) Print(img image.Image, label LabelSize) error {
	logger.Debug("=== Starting Print Function ===")
	logger.Debug("Printer model: %s", p.model)
	logger.Debug("Label: %s (width=%dmm, height=%dmm)", label.Name, label.TapeSizeWidth, label.TapeSizeHeight)

	model, err := GetModel(p.model)
	if err != nil {
		// Even though GetModel returns defaults, log the error
		logger.Warning("Model warning: %v", err)
	}

	var buf bytes.Buffer
	bounds := img.Bounds()
	height := bounds.Max.Y

	logger.Debug("Image dimensions: %dx%d pixels", bounds.Dx(), bounds.Dy())

	// --- Phase 0: Reset printer and drain stale responses ---
	// Send invalidate + initialize first as a separate write to clear any
	// error state from previous failed prints. Then read and discard any
	// stale status response the printer may have queued.
	var resetBuf bytes.Buffer
	resetBuf.Write(bytes.Repeat([]byte{0x00}, model.InvalidateBytes))
	resetBuf.Write([]byte{0x1B, 0x40}) // ESC @ = Initialize
	if _, err := p.backend.Write(resetBuf.Bytes()); err != nil {
		return fmt.Errorf("failed to send reset: %w", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Read and discard any stale response (non-fatal if nothing to read).
	// If the backend supports configurable timeout, use a short one for draining.
	if lb, ok := p.backend.(*LinuxBackend); ok {
		lb.readTimeout = 300 * time.Millisecond
	}
	discardBuf := make([]byte, 64)
	n, _ := p.backend.Read(discardBuf)
	if n > 0 {
		logger.Debug("Discarded %d stale bytes from printer after reset", n)
	}
	// Restore default timeout for subsequent reads
	if lb, ok := p.backend.(*LinuxBackend); ok {
		lb.readTimeout = 3 * time.Second
	}

	// --- KORJAUS KUVAN VENYMISEEN JA MUSTAAN RAITAAN ---
	// Ongelma: Tulostin odottaa dataa koko tulostuspään leveydeltä (rasterileveys),
	// mutta API:sta tuleva kuva on vain tulostettavan alueen levyinen.
	// Tämä aiheuttaa kuvan venymisen. Aiempi korjausyritys loi mustan taustan,
	// joka tulostui raitana.
	//
	// Ratkaisu:
	// 1. Luodaan uusi, tyhjä VALKOINEN kuva (`CreateBlankImage`), joka on täsmälleen
	//    tulostimen fyysisen rasterileveyden kokoinen.
	// 2. Piirretään alkuperäinen kuva tämän uuden, leveämmän kuvan keskelle.
	//    Tämä lisää oikean määrän valkoista tyhjää tilaa reunoille.
	// 3. Jatketaan tämän uuden, oikean levyisen kuvan käsittelyä.

	rasterWidthPixels := model.RasterWidthBytes * 8
	logger.Debug("Raster width: %d bytes (%d pixels)", model.RasterWidthBytes, rasterWidthPixels)

	fullWidthImg := CreateBlankImage(rasterWidthPixels, height) // Luo valkoisen pohjan

	// Lasketaan keskitystä varten tarvittava siirtymä
	offsetX := (rasterWidthPixels - bounds.Dx()) / 2
	offset := image.Point{X: offsetX, Y: 0}
	drawRect := bounds.Add(offset)

	// Piirretään alkuperäinen kuva valkoisen pohjan päälle keskitetysti
	draw.Draw(fullWidthImg, drawRect, img, bounds.Min, draw.Src)

	// Protokolla vaatii kuvan kääntämisen peilikuvaksi.
	flippedImg := flipImageHorizontally(fullWidthImg) // Käännetään uusi, täysleveä kuva
	// Rasteroidaan uusi, oikean levyinen kuva. Annetaan rasteroijalle leveydeksi
	// myös tämä uusi, täysi leveys pikseleinä.
	rasterData := p.rasterize(flippedImg, model.RasterWidthBytes, rasterWidthPixels)
	logger.Debug("Rasterized %d rows of data", len(rasterData))

	// --- COMMAND STREAM BUILDING FOLLOWING PROTOCOL ---
	// Note: Invalidate + Initialize already sent in Phase 0 above.

	// 3. Switch Mode (to Raster) - ONLY IF MODEL SUPPORTS
	if model.SupportsSwitchMode {
		buf.Write([]byte{0x1B, 0x69, 0x61, 0x01})
	}

	// 4. Set Media and Quality
	buf.Write([]byte{0x1B, 0x69, 0x7A}) // Opcode: ESC i z
	payload := make([]byte, 10)

	// Flags byte built dynamically per protocol.
	// Base flags for media type, auto-cut etc.
	var flags byte = 0x80 | (1 << 3) | (1 << 2) | (1 << 1)

	// Set quality flag for models that need it (like QL-570)
	if model.NeedsQualitySetting {
		flags |= (1 << 6) // Set standard quality (300x300dpi) flag
	}

	payload[0] = flags

	// For die-cut labels, DotsPrintableHeight is > 0. This is more reliable
	// than a separate IsDieCut field.
	isDieCut := label.DotsPrintableHeight > 0

	if isDieCut {
		payload[1] = 0x0B // Die-cut
		payload[3] = byte(label.TapeSizeHeight)
		logger.Debug("Media type: Die-cut (0x0B), width=%dmm, height=%dmm", label.TapeSizeWidth, label.TapeSizeHeight)
	} else {
		payload[1] = 0x0A // Continuous
		payload[3] = 0    // CRITICAL: Length must be 0 for continuous tape
		logger.Debug("Media type: Continuous (0x0A), width=%dmm", label.TapeSizeWidth)
	}
	payload[2] = byte(label.TapeSizeWidth)

	binary.LittleEndian.PutUint32(payload[4:8], uint32(height)) // Image height in pixels
	logger.Debug("Print height: %d pixels", height)
	buf.Write(payload)

	// 5. Set Margins - CRITICAL FIX
	// Use label-specific value, not hardcoded zero
	buf.Write([]byte{0x1B, 0x69, 0x64}) // Opcode: ESC i d
	marginPayload := make([]byte, 2)
	binary.LittleEndian.PutUint16(marginPayload, uint16(label.FeedMargin))
	buf.Write(marginPayload)
	logger.Debug("Feed margin: %d dots", label.FeedMargin)

	// 6. Set Auto-Cut and Expanded Mode
	// Send commands as the working Python reference does:
	// First enable auto cut (ESC i M 0x40)
	buf.Write([]byte{0x1B, 0x69, 0x4D, 0x40})
	// Then set expanded mode (ESC i K 0x08) - Important: Don't set 0x40 (high res) here!
	buf.Write([]byte{0x1B, 0x69, 0x4B, 0x08})

	// 7. Set Compression - ONLY IF MODEL SUPPORTS
	useCompression := model.SupportsCompression
	if useCompression {
		buf.Write([]byte{0x4D, 0x02}) // PackBits on
	} else {
		buf.Write([]byte{0x4D, 0x00}) // Compression off
	}

	// 8. Raster Data Transfer
	for _, row := range rasterData {
		var dataToSend []byte
		if useCompression {
			dataToSend = packBits(row)
		} else {
			dataToSend = row
		}

		length := len(dataToSend)
		if length > 255 {
			return fmt.Errorf("raster row is too long (%d bytes) for a single-byte length", length)
		}

		// 'g' command for data transfer
		buf.Write([]byte{0x67, 0x00})
		buf.WriteByte(byte(length)) // Length as single byte
		buf.Write(dataToSend)
	}

	// 9. Print
	buf.WriteByte(0x1A)

	// Send entire command stream to printer
	totalBytes := buf.Len()
	logger.Debug("Sending %d bytes to printer", totalBytes)
	_, err = p.backend.Write(buf.Bytes())
	if err != nil {
		logger.Error("Failed to write to printer: %v", err)
		return err
	}
	logger.Debug("Print command sent successfully")

	// Request status from printer
	status, err := p.RequestStatus()
	if err != nil {
		logger.Warning("Failed to read printer status: %v", err)
	} else {
		logger.Debug("Printer Ready: %v, Busy: %v, Error: %s", status.Ready, status.Busy, status.Error)
		logger.Debug("Media: %s, Width: %dmm", status.MediaType, status.MediaWidth)
		if status.Error != "" {
			logger.Error("PRINTER ERROR: %s", status.Error)
			return fmt.Errorf("printer reported error: %s", status.Error)
		}
		logger.Debug("No printer errors reported")
	}

	return nil
}

// RequestStatus sends a status request to the printer and reads the response
func (p *BrotherQL) RequestStatus() (PrinterStatus, error) {
	// Give printer time to process the print command
	time.Sleep(200 * time.Millisecond)

	// Send status request command
	statusCmd := []byte{0x1B, 0x69, 0x53} // ESC i S
	_, err := p.backend.Write(statusCmd)
	if err != nil {
		return PrinterStatus{}, fmt.Errorf("failed to write status request: %w", err)
	}

	// Read status response with retries - accumulate data in case it arrives in chunks
	var allData []byte
	buf := make([]byte, 64)
	maxRetries := 5

	for i := 0; i < maxRetries; i++ {
		time.Sleep(100 * time.Millisecond)
		n, readErr := p.backend.Read(buf)

		if n > 0 {
			allData = append(allData, buf[:n]...)
			if len(allData) >= 32 {
				logger.Debug("Status read successful on attempt %d/%d: got %d bytes", i+1, maxRetries, len(allData))
				break
			}
		}

		if readErr != nil {
			logger.Debug("Status read attempt %d/%d failed: %v", i+1, maxRetries, readErr)
			continue
		}

		logger.Debug("Status read attempt %d/%d: got %d bytes total, expected 32", i+1, maxRetries, len(allData))
	}

	if len(allData) < 32 {
		return PrinterStatus{}, fmt.Errorf("incomplete status response after %d retries: got %d bytes, expected 32", maxRetries, len(allData))
	}

	return ParseStatusResponse(allData[:32])
}

// packBits implements the PackBits compression algorithm.
func packBits(data []byte) []byte {
	var compressed bytes.Buffer
	i := 0
	max := len(data)
	for i < max {
		// Try to find a run of repeated bytes
		runLen := 1
		// Find run, but not longer than 128 bytes
		for i+runLen < max && data[i] == data[i+runLen] && runLen < 128 {
			runLen++
		}

		if runLen > 1 {
			// This is a run of repeated bytes. Encode it.
			// Control byte is -(runLen - 1)
			compressed.WriteByte(byte(1 - runLen))
			compressed.WriteByte(data[i])
			i += runLen
		} else {
			// This is a run of literal (non-repeated) bytes.
			// Find where the literal run ends.
			literalEnd := i
			for literalEnd < max {
				// Stop if a run of 2 or more identical bytes is found.
				if literalEnd+1 < max && data[literalEnd] == data[literalEnd+1] {
					break
				}
				literalEnd++
				// Stop if the literal run reaches 128 bytes.
				if literalEnd-i == 128 {
					break
				}
			}
			literalLen := literalEnd - i
			// Control byte is (literalLen - 1)
			compressed.WriteByte(byte(literalLen - 1))
			compressed.Write(data[i:literalEnd])
			i = literalEnd
		}
	}
	return compressed.Bytes()
}

// rasterize converts an image to the printer's raster format.
func (p *BrotherQL) rasterize(img image.Image, bytesPerRow int, printableWidth int) [][]byte {
	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	var rasterData [][]byte

	// Varmistetaan, että käsittelemme Gray-kuvaa.
	grayImg, ok := img.(*image.Gray)
	if !ok {
		// Jos kuva ei ole harmaasävy, muunnetaan se. Tämä tekee funktiosta vankemman.
		newGray := image.NewGray(bounds)
		draw.Draw(newGray, bounds, img, bounds.Min, draw.Src)
		grayImg = newGray
	}

	for y := 0; y < height; y++ {
		rowData := make([]byte, bytesPerRow)

		loopWidth := printableWidth
		if width < printableWidth {
			loopWidth = width
		}

		for x := 0; x < loopWidth; x++ {
			// --- TÄMÄ ON KRIITTINEN KORJAUS ---
			// Emme tarkista alfakanavaa, vaan harmaasävyarvoa.
			// Protokollan mukaan musta = 1, valkoinen = 0.
			// image.Gray-mallissa musta = 0, valkoinen = 255.
			// Joten jos pikseli EI ole täysin valkoinen (arvo < 250), se tulostetaan mustana.
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
	defer f.Close()

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
		Errors: []string{}, // Always initialize to empty slice (never nil) for frontend
	}

	// Parse model name from model code byte
	if modelName, ok := statusModelCodes[data[4]]; ok {
		status.ModelName = modelName
	} else {
		status.ModelName = fmt.Sprintf("Unknown (0x%02x)", data[4])
	}

	// Collect all errors (not just the first one)
	errorByte1 := data[8]
	errorByte2 := data[9]

	for i := 0; i < 8; i++ {
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
		// Determine Ready/Busy state based on phase type
		phaseType := data[19]
		if phaseType == 0x00 {
			status.Ready = true
			status.Busy = false
		} else if phaseType == 0x01 {
			status.Ready = false
			status.Busy = true
		} else {
			status.Ready = true
			status.Busy = false
		}
	}

	// Parse media info
	status.MediaWidth = int(data[10])
	status.MediaLength = int(data[17])

	if mediaType, ok := statusMediaTypes[data[11]]; ok {
		status.MediaType = mediaType
	}

	// Parse status type and phase type
	if st, ok := statusTypes[data[18]]; ok {
		status.StatusType = st
	}
	if pt, ok := statusPhaseTypes[data[19]]; ok {
		status.PhaseType = pt
	}

	logger.Debug("Parsed status - Model: %s, Ready: %v, Busy: %v, Errors: %v, MediaType: %s, MediaWidth: %d, MediaLength: %d",
		status.ModelName, status.Ready, status.Busy, status.Errors, status.MediaType, status.MediaWidth, status.MediaLength)

	return status, nil
}
