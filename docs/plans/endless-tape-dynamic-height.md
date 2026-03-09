# Plan: Endless Tape Dynamic & Custom Height

## Problem

1. **Rotated + endless tape bug**: When `orientation: "rotated"` and tape is endless (`DotsPrintableHeight == 0`), canvas dimensions break:
   - `canvasWidth = DotsPrintableHeight = 0` (broken!)
   - `canvasHeight = DotsPrintableWidth = 696` (locked to fixed value)
   - Result: rotated text on endless tape ignores content size

2. **No custom height**: User cannot specify tape length manually. Use case: printing on non-adhesive rolls/cardboard where exact dimensions matter.

3. **PNG endpoints hardcoded**: `print_png` and `print_png_raw` use hardcoded 300px fallback for endless tape instead of content-driven sizing.

## Scope

### Backend Changes

#### B1: Add `custom_height_mm` to PrintRequest

File: `api/print.go`

```go
type PrintRequest struct {
    // ... existing fields ...
    CustomHeightMM float64 `json:"custom_height_mm"` // 0 = auto (content-driven)
}
```

Add a helper to convert mm to dots (300 DPI):

```go
func mmToDots(mm float64) int {
    return int(mm * 300.0 / 25.4)
}
```

#### B2: Fix renderTextLabel canvas logic

File: `api/print.go`, function `renderTextLabel`

Current broken logic (lines 60-75):
```go
canvasWidth := label.DotsPrintableWidth
canvasHeight := label.DotsPrintableHeight
if isRotated {
    canvasWidth = label.DotsPrintableHeight   // 0 for endless!
    canvasHeight = label.DotsPrintableWidth   // locked!
}
if canvasHeight > 0 {
    imageHeight = canvasHeight
} else {
    imageHeight = textBoundsHeight + (2 * padding)
}
```

New logic:
```go
canvasWidth := label.DotsPrintableWidth
canvasHeight := label.DotsPrintableHeight

// Custom height override (only for endless tape)
if req.CustomHeightMM > 0 && !label.IsDieCut {
    canvasHeight = mmToDots(req.CustomHeightMM)
}

if isRotated {
    // For endless tape: canvasWidth stays as DotsPrintableWidth (print head width)
    // canvasHeight becomes dynamic or custom
    // We render on a "rotated canvas" then rotate back
    canvasWidth, canvasHeight = canvasHeight, canvasWidth
}

// Determine image height
var imageHeight int
if canvasHeight > 0 {
    imageHeight = canvasHeight
} else {
    imageHeight = textBoundsHeight + (2 * padding)
}
```

Key insight: When rotated + endless + auto height:
- `canvasHeight` is 0, so after swap: `canvasWidth=0, canvasHeight=DotsPrintableWidth`
- `canvasWidth=0` means we need the dynamic dimension for width too
- After swap, `imageHeight = canvasHeight = DotsPrintableWidth` (correct: this becomes the print head dimension)
- But `canvasWidth` (the tape direction) must be dynamic

Revised approach - separate print-head dimension from tape dimension:

```go
// printHeadWidth is always fixed by the tape
printHeadDots := label.DotsPrintableWidth

// tapeLengthDots is either custom, label-defined, or content-driven
var tapeLengthDots int
if req.CustomHeightMM > 0 && !label.IsDieCut {
    tapeLengthDots = mmToDots(req.CustomHeightMM)
} else {
    tapeLengthDots = label.DotsPrintableHeight // 0 for endless
}

var canvasWidth, canvasHeight int
if isRotated {
    // Canvas is rotated: width = tape direction, height = print head
    if tapeLengthDots > 0 {
        canvasWidth = tapeLengthDots
    } else {
        canvasWidth = textBoundsWidth + (2 * padding) // dynamic
    }
    canvasHeight = printHeadDots
} else {
    // Normal: width = print head, height = tape direction
    canvasWidth = printHeadDots
    if tapeLengthDots > 0 {
        canvasHeight = tapeLengthDots
    } else {
        canvasHeight = textBoundsHeight + (2 * padding) // dynamic
    }
}

img := brotherql.CreateBlankImage(canvasWidth, canvasHeight)
```

After rendering text + alignment, rotate if needed:
```go
if isRotated {
    img = brotherql.RotateImage(img, 90)
    // Result: width=printHeadDots, height=tapeLengthDots (correct for printer)
}
```

#### B3: Fix processSVG for rotated + endless

File: `api/print_svg.go`, function `processSVG`

Apply same printHeadDots/tapeLengthDots pattern. SVG already handles endless height=0 correctly for normal orientation, but needs the same rotated fix.

#### B4: Fix PNG endpoints

Files: `api/print_png.go`, `api/print_png_raw.go`

Replace hardcoded `300px` fallback:
- If `custom_height_mm > 0`: use `mmToDots(custom_height_mm)`
- Else: scale proportionally to fit `DotsPrintableWidth`, derive height from aspect ratio

```go
wantW := label.DotsPrintableWidth
wantH := label.DotsPrintableHeight
if wantH == 0 {
    if req.CustomHeightMM > 0 {
        wantH = mmToDots(req.CustomHeightMM)
    } else {
        // Scale proportionally based on source image aspect ratio
        srcBounds := img.Bounds()
        aspect := float64(srcBounds.Dy()) / float64(srcBounds.Dx())
        wantH = int(float64(wantW) * aspect)
    }
}
```

#### B5: Fix QR endpoint

File: `api/print_qr.go`

Add `custom_height_mm` support. When set, QR size fits within the custom height.

#### B6: Preview endpoint

File: `api/preview.go`

Preview uses `renderTextLabel` and `processSVG` so it inherits fixes automatically. Ensure `CustomHeightMM` is passed through from the preview request.

Add field to preview request struct if separate, or reuse PrintRequest.

### Frontend Changes

#### F1: Add height control for endless tape

File: `frontend/src/App.tsx`

New state:
```typescript
const [customHeightMM, setCustomHeightMM] = useState<number>(0);
const [heightMode, setHeightMode] = useState<"auto" | "manual">("auto");
```

UI element (shown only when selected label is endless tape, i.e. `labelHeight === 0`):

```
[Height Mode]
  ( ) Auto - fit to content
  ( ) Manual - specify length

[Height (mm)] ____  (shown only when manual, number input, min=10, max=2000)
```

#### F2: Pass custom_height_mm to API calls

Files: `frontend/src/App.tsx`, `frontend/src/hooks/usePreview.ts`

Add `custom_height_mm` to:
- Print request body
- Preview request body

Value: `heightMode === "manual" ? customHeightMM : 0`

#### F3: Update LabelPreview component

File: `frontend/src/components/LabelPreview.tsx`

When endless tape + manual height:
- Show preview container with correct aspect ratio using custom height
- Convert mm to preview pixels: `customHeightMM * PREVIEW_SCALE_FACTOR`

When endless tape + auto height:
- Use backend preview dimensions (current behavior, but now correct for rotated)

#### F4: Persist height settings

File: `frontend/src/utils/localStorageUtils.ts`

Add `customHeightMM` and `heightMode` to saved settings.

## Implementation Order

1. **B1** - Add `custom_height_mm` field + `mmToDots` helper
2. **B2** - Fix `renderTextLabel` rotated+endless bug (core fix)
3. **B3** - Fix `processSVG` rotated+endless
4. **B4** - Fix PNG endpoints hardcoded height
5. **B5** - Fix QR endpoint
6. **B6** - Verify preview works with fixes
7. **F1** - Add height controls to UI
8. **F2** - Wire API calls
9. **F3** - Update preview component
10. **F4** - Persist settings

## Testing

- Text + 62mm endless + normal orientation + auto height -> height fits text
- Text + 62mm endless + rotated orientation + auto height -> tape length fits text width
- Text + 62mm endless + manual 100mm height -> exactly 100mm printed
- Text + 62mm endless + rotated + manual 150mm -> 150mm tape, text rotated
- PNG + 62mm endless + auto -> proportional scaling
- PNG + 62mm endless + manual 80mm -> fits within 80mm
- SVG + rotated + endless -> correct dimensions
- Die-cut labels -> custom_height_mm ignored, fixed size always used
- Preview matches print output in all cases above
- Use `printer: "file"` for all tests (debug output to `debug_output/`)

## Notes

- `mmToDots` formula: `mm * 300 / 25.4` (300 DPI printer resolution)
- Die-cut labels always ignore `custom_height_mm` (physical label size is fixed)
- Protocol sends actual image height in `buildCommandStream` regardless of label type, so no protocol changes needed
- FeedMargin (35 dots for endless) is applied by the protocol layer, not the rendering layer
