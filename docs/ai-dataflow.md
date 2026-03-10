# Data Flow

## Print Pipeline

```mermaid
flowchart TD
    A[POST /api/print] --> B[PrintLabel handler]
    B --> C[Acquire PrinterLock]
    C --> D[Resolve label size + font]
    D --> E[renderTextLabel]
    E --> F[MeasureText + DrawText]
    F --> G[rotateForPrinter if rotated]
    G --> H[ConnectToPrinter]
    H --> I[BrotherQL.Print]
    I --> J[Reset: invalidate + ESC @]
    J --> K[Flip image horizontally]
    K --> L[Build command stream]
    L --> M[Rasterize: threshold 250]
    M --> N[Stream rows + compress]
    N --> O[Print cmd 0x1A]
    O --> P[RequestStatus]
```

## SSE Flow

```mermaid
flowchart LR
    A[SSEHub polls 5s] --> B{printers changed?}
    B -->|yes| C[broadcast to clients]
    B -->|no| D[skip]
    E[ForceRefresh] --> C
```

## Key Structures

```yaml
PrintRequest:
  Text: string
  LabelSize: string         # e.g. "62"
  FontFamily: string
  FontSize: float64
  Printer: string            # name/UID/"file"
  Model: string
  Orientation: string        # ""/portrait/landscape/"rotated"
  Alignment: string          # center/start/end
  SVGData: string            # optional SVG content
  Scale: float64
  ContentRotation: float64   # 90/270 for SVG/QR/PNG

PrinterModel:
  RasterWidthBytes: int   # 90 (standard) or 162 (wide)
  SupportsSwitchMode: bool
  SupportsCompression: bool
  InvalidateBytes: int    # 200-400

LabelSize:
  DotsTotalWidth: int
  DotsPrintableWidth: int
  DotsPrintableHeight: int  # 0 = endless
  TapeSizeWidth: int        # mm
  FeedMargin: int           # 35 (endless) or 0 (die-cut)

ServerConfig:
  Address: string
  Port: int
  TLS: bool
  CertFile: string
  KeyFile: string
  Token: string             # Bearer auth (hidden from JSON)

PrinterStatus:              # 32-byte response
  Ready/Busy/Error: bool
  MediaType/Width/Length: int
```

## Orientation Model

- Standard: canvas width = printHeadDots, height = tapeLengthDots
- `"rotated"`: swaps width/height, then `rotateForPrinter()` applies 90° rotation
- `ContentRotation`: separate 90°/270° rotation for SVG/QR/PNG content

## Debug Mode

`printer: "file"` → saves PNG to `debug_output/` instead of printing
