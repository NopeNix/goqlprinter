# Data Flow

## Print Pipeline

```mermaid
flowchart TD
    A[POST /api/print] --> B[PrintLabel handler]
    B --> C[Acquire PrinterLock]
    C --> D[Resolve label size + font]
    D --> E[renderTextLabel]
    E --> F[MeasureText + DrawText]
    F --> G[ConnectToPrinter]
    G --> H[BrotherQL.Print]
    H --> I[Reset: invalidate + ESC @]
    I --> J[Flip image horizontally]
    J --> K[Build command stream]
    K --> L[Rasterize: threshold 250]
    L --> M[Stream rows + compress]
    M --> N[Print cmd 0x1A]
    N --> O[RequestStatus]
```

## Key Structures

```yaml
PrintRequest:
  Text: string
  LabelSize: string      # e.g. "62"
  FontFamily: string
  FontSize: float64
  Printer: string         # name/UID/"file"
  Model: string
  Orientation: string     # portrait/landscape
  Alignment: string       # center/start/end
  SVGData: string         # optional SVG content
  Scale: float64

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

PrinterStatus:              # 32-byte response
  Ready/Busy/Error: bool
  MediaType/Width/Length: int
```

## Debug Mode

`printer: "file"` → saves PNG to `debug_output/` instead of printing
