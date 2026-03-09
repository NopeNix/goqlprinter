package brotherql

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"testing"
	"time"
)

// --- Mock Backend ---

type mockBackend struct {
	// writes tracks each Write call separately for phase analysis
	writes      [][]byte
	writtenData []byte   // flattened view of all writes
	readPhases  [][]byte // each entry is returned by one Read call in order
	readPhase   int
	writeErr    error
	readErr     error
}

func (m *mockBackend) Write(data []byte) (int, error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	m.writes = append(m.writes, cp)
	m.writtenData = append(m.writtenData, data...)
	return len(data), nil
}

func (m *mockBackend) Read(data []byte) (int, error) {
	if m.readErr != nil {
		return 0, m.readErr
	}
	if m.readPhase >= len(m.readPhases) {
		return 0, nil
	}
	n := copy(data, m.readPhases[m.readPhase])
	m.readPhase++
	return n, nil
}

func (m *mockBackend) Close() error { return nil }

// --- packBits tests ---

func TestPackBits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []byte
		check func(t *testing.T, result []byte)
	}{
		{
			name:  "empty input",
			input: []byte{},
			check: func(t *testing.T, result []byte) {
				if len(result) != 0 {
					t.Errorf("packBits(empty) = %d bytes, want 0", len(result))
				}
			},
		},
		{
			name:  "single byte",
			input: []byte{0x42},
			check: func(t *testing.T, result []byte) {
				// Single byte → literal run of length 1: control=0, data=0x42
				if len(result) != 2 {
					t.Fatalf("packBits(single) = %d bytes, want 2", len(result))
				}
				if result[0] != 0x00 {
					t.Errorf("control byte = 0x%02x, want 0x00", result[0])
				}
				if result[1] != 0x42 {
					t.Errorf("data byte = 0x%02x, want 0x42", result[1])
				}
			},
		},
		{
			name:  "all zeros (run-length)",
			input: bytes.Repeat([]byte{0x00}, 10),
			check: func(t *testing.T, result []byte) {
				// 10-byte run: control = 1 - 10 = -9 = 0xF7, data = 0x00
				if len(result) != 2 {
					t.Fatalf("packBits(10 zeros) = %d bytes, want 2", len(result))
				}
				want := byte(256 - 9) // 1-10 = -9, as unsigned byte = 0xF7
				if result[0] != want {
					t.Errorf("control byte = 0x%02x, want 0x%02x", result[0], want)
				}
				if result[1] != 0x00 {
					t.Errorf("data byte = 0x%02x, want 0x00", result[1])
				}
			},
		},
		{
			name:  "all different bytes (literal)",
			input: []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			check: func(t *testing.T, result []byte) {
				// Literal run of 5: control = 4, then 5 data bytes
				if len(result) != 6 {
					t.Fatalf("packBits(5 different) = %d bytes, want 6", len(result))
				}
				if result[0] != 4 {
					t.Errorf("control byte = %d, want 4", result[0])
				}
				for i, b := range []byte{0x01, 0x02, 0x03, 0x04, 0x05} {
					if result[i+1] != b {
						t.Errorf("data[%d] = 0x%02x, want 0x%02x", i, result[i+1], b)
					}
				}
			},
		},
		{
			name:  "mixed runs and literals",
			input: []byte{0x01, 0x02, 0x03, 0xAA, 0xAA, 0xAA, 0xAA, 0x05},
			check: func(t *testing.T, result []byte) {
				// Should produce: literal(3: 01 02 03) + run(4: AA) + literal(1: 05)
				// Decompressing result should give original
				decompressed := decompressPackBits(result)
				if !bytes.Equal(decompressed, []byte{0x01, 0x02, 0x03, 0xAA, 0xAA, 0xAA, 0xAA, 0x05}) {
					t.Errorf("round-trip failed: got %v", decompressed)
				}
			},
		},
		{
			name:  "round-trip preserves data",
			input: []byte{0xFF, 0xFF, 0xFF, 0x00, 0x01, 0x02, 0x03, 0x03, 0x03},
			check: func(t *testing.T, result []byte) {
				decompressed := decompressPackBits(result)
				if !bytes.Equal(decompressed, []byte{0xFF, 0xFF, 0xFF, 0x00, 0x01, 0x02, 0x03, 0x03, 0x03}) {
					t.Errorf("round-trip failed: got %v", decompressed)
				}
			},
		},
		{
			name:  "compression is smaller for repetitive data",
			input: bytes.Repeat([]byte{0xAB}, 128),
			check: func(t *testing.T, result []byte) {
				if len(result) >= 128 {
					t.Errorf("compressed size %d should be < 128 for 128 identical bytes", len(result))
				}
				decompressed := decompressPackBits(result)
				if !bytes.Equal(decompressed, bytes.Repeat([]byte{0xAB}, 128)) {
					t.Errorf("round-trip failed for 128 identical bytes")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := packBits(tc.input)
			tc.check(t, result)
		})
	}
}

// decompressPackBits is a test helper that reverses PackBits compression.
func decompressPackBits(data []byte) []byte {
	var result []byte
	i := 0
	for i < len(data) {
		control := int(int8(data[i]))
		i++
		if control >= 0 {
			// Literal run: control+1 bytes follow
			n := control + 1
			result = append(result, data[i:i+n]...)
			i += n
		} else {
			// Repeat run: 1-control copies of next byte
			n := 1 - control
			val := data[i]
			i++
			for range n {
				result = append(result, val)
			}
		}
	}
	return result
}

// --- ParseStatusResponse tests ---

func TestParseStatusResponse(t *testing.T) {
	t.Parallel()

	// Build a valid 32-byte status response template
	makeStatus := func() []byte {
		data := make([]byte, 32)
		data[0] = 0x80  // header byte 0
		data[2] = 0x42  // header byte 2
		data[4] = 0x38  // model code for QL-800
		data[8] = 0x00  // error info 1 (no errors)
		data[9] = 0x00  // error info 2 (no errors)
		data[10] = 62   // media width mm
		data[11] = 0x0A // continuous tape
		data[17] = 0    // media length
		data[18] = 0x00 // status type: reply to status request
		data[19] = 0x00 // phase: waiting to receive
		return data
	}

	tests := []struct {
		name      string
		data      []byte
		wantErr   bool
		checkFunc func(t *testing.T, s PrinterStatus)
	}{
		{
			name: "valid ready response",
			data: makeStatus(),
			checkFunc: func(t *testing.T, s PrinterStatus) {
				if !s.Ready {
					t.Error("expected Ready=true")
				}
				if s.Busy {
					t.Error("expected Busy=false")
				}
				if s.Error != "" {
					t.Errorf("expected no error, got %q", s.Error)
				}
				if len(s.Errors) != 0 {
					t.Errorf("expected 0 errors, got %d", len(s.Errors))
				}
				if s.ModelName != "QL-800" {
					t.Errorf("ModelName = %q, want QL-800", s.ModelName)
				}
				if s.MediaWidth != 62 {
					t.Errorf("MediaWidth = %d, want 62", s.MediaWidth)
				}
				if s.MediaType != "Continuous length tape" {
					t.Errorf("MediaType = %q, want Continuous length tape", s.MediaType)
				}
			},
		},
		{
			name: "busy state (printing)",
			data: func() []byte {
				d := makeStatus()
				d[19] = 0x01 // phase: printing
				return d
			}(),
			checkFunc: func(t *testing.T, s PrinterStatus) {
				if s.Ready {
					t.Error("expected Ready=false during printing")
				}
				if !s.Busy {
					t.Error("expected Busy=true during printing")
				}
				if s.PhaseType != "Printing state" {
					t.Errorf("PhaseType = %q, want Printing state", s.PhaseType)
				}
			},
		},
		{
			name: "error bits set",
			data: func() []byte {
				d := makeStatus()
				d[8] = 0x01 // bit 0 of error info 1: "No media when printing"
				return d
			}(),
			checkFunc: func(t *testing.T, s PrinterStatus) {
				if s.Ready {
					t.Error("expected Ready=false with errors")
				}
				if len(s.Errors) == 0 {
					t.Fatal("expected at least 1 error")
				}
				if s.Errors[0] != "No media when printing" {
					t.Errorf("Errors[0] = %q, want 'No media when printing'", s.Errors[0])
				}
				if s.Error != "No media when printing" {
					t.Errorf("Error = %q, want 'No media when printing'", s.Error)
				}
			},
		},
		{
			name: "multiple error bits",
			data: func() []byte {
				d := makeStatus()
				d[8] = 0x03 // bits 0 and 1: "No media when printing" + "End of media"
				return d
			}(),
			checkFunc: func(t *testing.T, s PrinterStatus) {
				if len(s.Errors) < 2 {
					t.Fatalf("expected >= 2 errors, got %d", len(s.Errors))
				}
			},
		},
		{
			name: "error info 2 bits",
			data: func() []byte {
				d := makeStatus()
				d[9] = 0x04 // bit 2 of error info 2: "Communication error"
				return d
			}(),
			checkFunc: func(t *testing.T, s PrinterStatus) {
				found := false
				for _, e := range s.Errors {
					if e == "Communication error" {
						found = true
					}
				}
				if !found {
					t.Errorf("expected 'Communication error' in errors, got %v", s.Errors)
				}
			},
		},
		{
			name: "die-cut media type",
			data: func() []byte {
				d := makeStatus()
				d[11] = 0x0B // die-cut
				return d
			}(),
			checkFunc: func(t *testing.T, s PrinterStatus) {
				if s.MediaType != "Die-cut labels" {
					t.Errorf("MediaType = %q, want Die-cut labels", s.MediaType)
				}
			},
		},
		{
			name: "unknown model code",
			data: func() []byte {
				d := makeStatus()
				d[4] = 0xFF // unknown model
				return d
			}(),
			checkFunc: func(t *testing.T, s PrinterStatus) {
				if s.ModelName == "" {
					t.Error("ModelName should not be empty for unknown model")
				}
				// Should contain "Unknown"
				if s.ModelName != "Unknown (0xff)" {
					t.Errorf("ModelName = %q, want Unknown (0xff)", s.ModelName)
				}
			},
		},
		{
			name:    "too short data",
			data:    make([]byte, 10),
			wantErr: true,
		},
		{
			name: "invalid header byte 0",
			data: func() []byte {
				d := makeStatus()
				d[0] = 0x00 // wrong header
				return d
			}(),
			wantErr: true,
		},
		{
			name: "invalid header byte 2",
			data: func() []byte {
				d := makeStatus()
				d[2] = 0x00 // wrong header
				return d
			}(),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, err := ParseStatusResponse(tc.data)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.checkFunc != nil {
				tc.checkFunc(t, status)
			}
		})
	}
}

// --- flipImageHorizontally tests ---

func TestFlipImageHorizontally(t *testing.T) {
	t.Parallel()

	// Create a 4x2 image with known colors:
	// Row 0: black, dark-gray, light-gray, white
	// Row 1: red, green, blue, yellow
	src := image.NewGray(image.Rect(0, 0, 4, 2))
	grayValues := []uint8{0, 64, 192, 255, 100, 150, 200, 50}
	for i, v := range grayValues {
		x := i % 4
		y := i / 4
		src.SetGray(x, y, color.Gray{Y: v})
	}

	flipped := flipImageHorizontally(src)
	bounds := flipped.Bounds()

	if bounds.Dx() != 4 || bounds.Dy() != 2 {
		t.Fatalf("flipped dimensions = %dx%d, want 4x2", bounds.Dx(), bounds.Dy())
	}

	// After horizontal flip, row 0 should be reversed: 255, 192, 64, 0
	for x := range 4 {
		flippedX := 3 - x
		got := flipped.GrayAt(flippedX, 0).Y
		want := grayValues[x]
		if got != want {
			t.Errorf("pixel (%d,0) after flip: got %d, want %d", flippedX, got, want)
		}
	}
}

func TestFlipImageHorizontally_SinglePixel(t *testing.T) {
	t.Parallel()

	src := image.NewGray(image.Rect(0, 0, 1, 1))
	src.SetGray(0, 0, color.Gray{Y: 42})

	flipped := flipImageHorizontally(src)
	got := flipped.GrayAt(0, 0).Y
	if got != 42 {
		t.Errorf("single pixel flip changed value: got %d, want 42", got)
	}
}

// --- Rasterize tests ---

func TestRasterize_WhiteImageProducesZeroBytes(t *testing.T) {
	t.Parallel()

	bql := &BrotherQL{model: "QL-800"}
	img := CreateBlankImage(8, 2) // 8 pixels wide = 1 byte per row, all white

	rows := bql.rasterize(img, 1, 8)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	for i, row := range rows {
		for j, b := range row {
			if b != 0 {
				t.Errorf("row[%d][%d] = 0x%02x, want 0x00 for white image", i, j, b)
			}
		}
	}
}

func TestRasterize_BlackImageProducesAllOnes(t *testing.T) {
	t.Parallel()

	bql := &BrotherQL{model: "QL-800"}
	img := image.NewGray(image.Rect(0, 0, 8, 1))
	for x := range 8 {
		img.SetGray(x, 0, color.Gray{Y: 0}) // black
	}

	rows := bql.rasterize(img, 1, 8)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0][0] != 0xFF {
		t.Errorf("row[0] = 0x%02x, want 0xFF for all-black row", rows[0][0])
	}
}

func TestRasterize_NonGrayImageConverted(t *testing.T) {
	t.Parallel()

	bql := &BrotherQL{model: "QL-800"}
	// Create an RGBA image (not Gray) with a black pixel
	img := image.NewRGBA(image.Rect(0, 0, 8, 1))
	for x := range 8 {
		img.Set(x, 0, color.White)
	}
	img.Set(0, 0, color.Black) // first pixel black

	rows := bql.rasterize(img, 1, 8)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	// Pixel 0 → bit 7 should be set
	if rows[0][0]&0x80 == 0 {
		t.Error("expected bit 7 set for black pixel at position 0")
	}
}

func TestRasterize_RowPaddedToModelWidth(t *testing.T) {
	t.Parallel()

	bql := &BrotherQL{model: "QL-800"}
	// Image is 16 pixels (2 bytes) but model requires 90 bytes per row
	img := image.NewGray(image.Rect(0, 0, 16, 1))

	rows := bql.rasterize(img, 90, 16)
	if len(rows[0]) != 90 {
		t.Errorf("row length = %d, want 90", len(rows[0]))
	}
}

// makeReadPhasesForPrint returns read phases for a successful Print() call:
// phase 0: drain read after reset (returns some junk bytes)
// phases 1-5: RequestStatus retries (phase 1 returns the full status response)
func makeReadPhasesForPrint(statusResp []byte) [][]byte {
	return [][]byte{
		make([]byte, 8), // drain: some stale bytes
		statusResp,      // status response (32 bytes)
	}
}

// makeValidStatusResponse builds a 32-byte valid status response.
func makeValidStatusResponse() []byte {
	data := make([]byte, 32)
	data[0] = 0x80
	data[2] = 0x42
	data[4] = 0x38  // QL-800
	data[19] = 0x00 // waiting
	return data
}

// --- Print integration test with mock backend ---

func TestPrint_MockBackend_FullProtocol(t *testing.T) {
	t.Parallel()

	mock := &mockBackend{
		readPhases: makeReadPhasesForPrint(makeValidStatusResponse()),
	}

	bql := NewBrotherQL("QL-800", mock)
	img := CreateBlankImage(696, 100)
	label, err := GetLabel("62")
	if err != nil {
		t.Fatalf("GetLabel failed: %v", err)
	}

	err = bql.Print(img, label)
	if err != nil {
		t.Fatalf("Print returned error: %v", err)
	}

	// Verify data was written
	if len(mock.writtenData) == 0 {
		t.Fatal("no data written to backend")
	}

	// Verify the written data starts with invalidate bytes (400 null bytes for QL-800)
	nullCount := 0
	for _, b := range mock.writtenData {
		if b == 0x00 {
			nullCount++
		} else {
			break
		}
	}
	if nullCount < 400 {
		t.Errorf("expected at least 400 null bytes at start, got %d", nullCount)
	}

	// Verify ESC @ (initialize) follows the null bytes
	if mock.writtenData[400] != 0x1B || mock.writtenData[401] != 0x40 {
		t.Errorf("expected ESC @ at offset 400, got 0x%02x 0x%02x", mock.writtenData[400], mock.writtenData[401])
	}

	// Print() makes 3 Write calls:
	//   writes[0]: reset (invalidate + ESC @)
	//   writes[1]: command stream (ESC i z, raster rows, 0x1A)
	//   writes[2]: status request (ESC i S)
	if len(mock.writes) < 3 {
		t.Fatalf("expected at least 3 Write calls, got %d", len(mock.writes))
	}

	// Verify command stream (writes[1]) ends with 0x1A (print and feed)
	cmdStream := mock.writes[1]
	if cmdStream[len(cmdStream)-1] != 0x1A {
		t.Errorf("command stream last byte = 0x%02x, want 0x1A (print and feed)", cmdStream[len(cmdStream)-1])
	}

	// Verify command stream contains ESC i z (media settings)
	escIZ := []byte{0x1B, 0x69, 0x7A}
	if !bytes.Contains(cmdStream, escIZ) {
		t.Error("ESC i z (media settings) not found in command stream")
	}

	// Verify command stream contains raster data prefix 'g' (0x67)
	rasterPrefix := []byte{0x67, 0x00}
	if !bytes.Contains(cmdStream, rasterPrefix) {
		t.Error("raster data prefix 'g' (0x67 0x00) not found in command stream")
	}

	// Verify status request is a separate write (writes[2]) with ESC i S
	statusCmd := []byte{0x1B, 0x69, 0x53}
	if !bytes.Equal(mock.writes[2], statusCmd) {
		t.Errorf("status write = %x, want ESC i S (%x)", mock.writes[2], statusCmd)
	}
}

func TestPrint_MockBackend_DieCutLabel(t *testing.T) {
	t.Parallel()

	mock := &mockBackend{readPhases: makeReadPhasesForPrint(makeValidStatusResponse())}
	bql := NewBrotherQL("QL-800", mock)
	img := CreateBlankImage(696, 271)
	label, _ := GetLabel("62x29")

	err := bql.Print(img, label)
	if err != nil {
		t.Fatalf("Print with die-cut label returned error: %v", err)
	}

	// Verify media type byte 0x0B (die-cut) is in the ESC i z command
	// ESC i z = 0x1B 0x69 0x7A, followed by 10 payload bytes where payload[1] = 0x0B
	escIZ := []byte{0x1B, 0x69, 0x7A}
	idx := bytes.Index(mock.writtenData, escIZ)
	if idx < 0 {
		t.Fatal("ESC i z command not found")
	}
	payloadStart := idx + len(escIZ)
	if payloadStart+2 > len(mock.writtenData) {
		t.Fatal("not enough data after ESC i z")
	}
	mediaType := mock.writtenData[payloadStart+1]
	if mediaType != 0x0B {
		t.Errorf("media type byte = 0x%02x, want 0x0B (die-cut)", mediaType)
	}
}

func TestPrint_MockBackend_CompressionModel(t *testing.T) {
	t.Parallel()

	mock := &mockBackend{readPhases: makeReadPhasesForPrint(makeValidStatusResponse())}
	// QL-820NWB supports compression
	bql := NewBrotherQL("QL-820NWB", mock)
	img := CreateBlankImage(696, 50)
	label, _ := GetLabel("62")

	err := bql.Print(img, label)
	if err != nil {
		t.Fatalf("Print with compression model returned error: %v", err)
	}

	// Verify compression mode is enabled: 0x4D 0x02
	compressionOn := []byte{0x4D, 0x02}
	if !bytes.Contains(mock.writtenData, compressionOn) {
		t.Error("compression mode ON (0x4D 0x02) not found in written data")
	}
}

func TestPrint_MockBackend_NoCompressionModel(t *testing.T) {
	t.Parallel()

	mock := &mockBackend{readPhases: makeReadPhasesForPrint(makeValidStatusResponse())}
	// QL-800 does NOT support compression
	bql := NewBrotherQL("QL-800", mock)
	img := CreateBlankImage(696, 50)
	label, _ := GetLabel("62")

	err := bql.Print(img, label)
	if err != nil {
		t.Fatalf("Print without compression returned error: %v", err)
	}

	// Verify compression mode is OFF: 0x4D 0x00
	compressionOff := []byte{0x4D, 0x00}
	if !bytes.Contains(mock.writtenData, compressionOff) {
		t.Error("compression mode OFF (0x4D 0x00) not found in written data")
	}
}

func TestPrint_MockBackend_WriteError(t *testing.T) {
	t.Parallel()

	mock := &mockBackend{writeErr: errors.New("USB write failed")}
	bql := NewBrotherQL("QL-800", mock)
	img := CreateBlankImage(696, 100)
	label, _ := GetLabel("62")

	err := bql.Print(img, label)
	if err == nil {
		t.Fatal("expected error when write fails, got nil")
	}
}

func TestPrint_MockBackend_ErrorStatus(t *testing.T) {
	t.Parallel()

	// Phase 0: drain read returns some stale bytes (separate from status)
	// Phase 1: RequestStatus returns a status response with error bit set
	errorStatus := make([]byte, 32)
	errorStatus[0] = 0x80
	errorStatus[2] = 0x42
	errorStatus[4] = 0x38
	errorStatus[8] = 0x01 // error bit 0: "No media when printing"
	errorStatus[19] = 0x00

	mock := &mockBackend{
		readPhases: [][]byte{
			make([]byte, 4), // drain phase: some stale bytes
			errorStatus,     // status phase: error response
		},
	}
	bql := NewBrotherQL("QL-800", mock)
	img := CreateBlankImage(696, 100)
	label, _ := GetLabel("62")

	err := bql.Print(img, label)
	if err == nil {
		t.Fatal("expected error when printer reports error status, got nil")
	}
	// Verify the error message mentions the actual printer error
	if !bytes.Contains([]byte(err.Error()), []byte("No media")) {
		t.Errorf("error should mention printer error, got: %v", err)
	}
}

func TestPrint_MockBackend_SwitchModeCommand(t *testing.T) {
	t.Parallel()

	// QL-800 supports switch mode
	mock := &mockBackend{readPhases: makeReadPhasesForPrint(makeValidStatusResponse())}
	bql := NewBrotherQL("QL-800", mock)
	img := CreateBlankImage(696, 50)
	label, _ := GetLabel("62")

	err := bql.Print(img, label)
	if err != nil {
		t.Fatalf("Print returned error: %v", err)
	}

	// ESC i a 0x01 = switch to raster mode
	switchMode := []byte{0x1B, 0x69, 0x61, 0x01}
	if !bytes.Contains(mock.writtenData, switchMode) {
		t.Error("switch mode command ESC i a 0x01 not found for QL-800")
	}
}

func TestPrint_MockBackend_NoSwitchModeForOldModel(t *testing.T) {
	t.Parallel()

	mock := &mockBackend{readPhases: makeReadPhasesForPrint(makeValidStatusResponse())}
	bql := NewBrotherQL("QL-500", mock)
	img := CreateBlankImage(106, 50)
	label, _ := GetLabel("12")

	err := bql.Print(img, label)
	if err != nil {
		t.Fatalf("Print returned error: %v", err)
	}

	// QL-500 does not support switch mode.
	// The command stream is writes[1] — verify ESC i a is NOT in it.
	if len(mock.writes) < 2 {
		t.Fatalf("expected at least 2 Write calls, got %d", len(mock.writes))
	}
	cmdStream := mock.writes[1]
	escIA := []byte{0x1B, 0x69, 0x61, 0x01}
	if bytes.Contains(cmdStream, escIA) {
		t.Error("switch mode ESC i a found in command stream for QL-500 which doesn't support it")
	}

	// Verify ESC i z IS present (media settings should always be there)
	escIZ := []byte{0x1B, 0x69, 0x7A}
	if !bytes.Contains(cmdStream, escIZ) {
		t.Error("ESC i z (media settings) missing from command stream")
	}
}

// --- RequestStatus tests ---

func TestRequestStatus_Success(t *testing.T) {
	t.Parallel()

	statusResp := make([]byte, 32)
	statusResp[0] = 0x80
	statusResp[2] = 0x42
	statusResp[4] = 0x38
	statusResp[10] = 62
	statusResp[11] = 0x0A
	statusResp[19] = 0x00

	mock := &mockBackend{readPhases: [][]byte{statusResp}}
	bql := NewBrotherQL("QL-800", mock)

	status, err := bql.RequestStatus()
	if err != nil {
		t.Fatalf("RequestStatus returned error: %v", err)
	}
	if !status.Ready {
		t.Error("expected Ready=true")
	}
	if status.MediaWidth != 62 {
		t.Errorf("MediaWidth = %d, want 62", status.MediaWidth)
	}
}

func TestRequestStatus_WriteError(t *testing.T) {
	t.Parallel()

	mock := &mockBackend{writeErr: errors.New("write failed")}
	bql := NewBrotherQL("QL-800", mock)

	_, err := bql.RequestStatus()
	if err == nil {
		t.Fatal("expected error when write fails")
	}
}

func TestRequestStatus_InsufficientData(t *testing.T) {
	t.Parallel()

	// Only return 10 bytes — not enough for a valid status
	mock := &mockBackend{readPhases: [][]byte{make([]byte, 10)}}
	bql := NewBrotherQL("QL-800", mock)

	_, err := bql.RequestStatus()
	if err == nil {
		t.Fatal("expected error for insufficient status data")
	}
}

// --- Print with ReadTimeoutSetter mock ---

type mockBackendWithTimeout struct {
	mockBackend
	timeoutsSet []time.Duration
}

func (m *mockBackendWithTimeout) SetReadTimeout(d time.Duration) {
	m.timeoutsSet = append(m.timeoutsSet, d)
}

func TestPrint_MockBackend_SetsReadTimeout(t *testing.T) {
	t.Parallel()

	mock := &mockBackendWithTimeout{
		mockBackend: mockBackend{readPhases: makeReadPhasesForPrint(makeValidStatusResponse())},
	}

	bql := NewBrotherQL("QL-800", mock)
	img := CreateBlankImage(696, 50)
	label, _ := GetLabel("62")

	err := bql.Print(img, label)
	if err != nil {
		t.Fatalf("Print returned error: %v", err)
	}

	// Print should set read timeout twice: once to 300ms for drain, once to 3s after
	if len(mock.timeoutsSet) < 2 {
		t.Errorf("expected at least 2 SetReadTimeout calls, got %d", len(mock.timeoutsSet))
	}
	if len(mock.timeoutsSet) >= 1 && mock.timeoutsSet[0] != 300*time.Millisecond {
		t.Errorf("first timeout = %v, want 300ms", mock.timeoutsSet[0])
	}
	if len(mock.timeoutsSet) >= 2 && mock.timeoutsSet[1] != 3*time.Second {
		t.Errorf("second timeout = %v, want 3s", mock.timeoutsSet[1])
	}
}

// --- NewBrotherQL test ---

func TestNewBrotherQL(t *testing.T) {
	t.Parallel()

	mock := &mockBackend{}
	bql := NewBrotherQL("QL-700", mock)
	if bql.model != "QL-700" {
		t.Errorf("model = %q, want QL-700", bql.model)
	}
	if bql.backend != mock {
		t.Error("backend not set correctly")
	}
}
