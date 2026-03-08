# Tech Stack

## Dependencies

| Package | Purpose |
|---------|---------|
| `gin-gonic/gin` | HTTP framework |
| `google/gousb` | USB communication (CGO) |
| `disintegration/imaging` | Image scaling/fitting |
| `golang/freetype` | TrueType text rendering |
| `skip2/go-qrcode` | QR code generation |
| `spf13/viper` | Config management |
| `swaggo/gin-swagger` | Swagger API docs |
| `alexbrainman/printer` | Windows native printing |

## Runtime Requirements

- `rsvg-convert` (librsvg2-bin) for SVG rendering
- USB permissions for direct printer access
- libusb-dev for USB build tag
- Font files (.ttf/.otf) in configured directories

## Supported Hardware

- Brother QL series: 500/550/560/570/580N/650TD/700/710W/720NW/800/810W/820NWB/1050/1060N/1100/1110NWB
- USB Vendor ID: `0x04f9`
- 20+ label sizes: endless tape (12-104mm), die-cut, round
