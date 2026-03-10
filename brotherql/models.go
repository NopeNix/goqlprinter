package brotherql

import (
	"errors"
	"fmt"
)

// PrinterModel holds model-specific protocol parameters.
type PrinterModel struct {
	Name                string
	SupportsSwitchMode  bool // whether the model supports ESC i a (switch to raster mode)
	SupportsCompression bool // whether the model supports PackBits compression (M command)
	NeedsQualitySetting bool // whether the model requires the 300x300 dpi quality flag
	RasterWidthBytes    int  // fixed raster row width in bytes
	InvalidateBytes     int  // number of null bytes sent at startup to clear the buffer
}

// GetModel returns the protocol parameters for the named printer model.
// If the model is not recognized, it returns QL-800 defaults and a non-nil error.
func GetModel(name string) (PrinterModel, error) {
	models := map[string]PrinterModel{
		// QL-5xx series: older models, no raster mode switch support.
		"QL-500": {
			Name:                "QL-500",
			SupportsSwitchMode:  false,
			SupportsCompression: false,
			NeedsQualitySetting: false,
			RasterWidthBytes:    90,
			InvalidateBytes:     200,
		},
		"QL-550": {
			Name:                "QL-550",
			SupportsSwitchMode:  false,
			SupportsCompression: false,
			RasterWidthBytes:    90,
			InvalidateBytes:     200,
		},
		"QL-560": {
			Name:                "QL-560",
			SupportsSwitchMode:  false,
			SupportsCompression: false,
			RasterWidthBytes:    90,
			InvalidateBytes:     200,
		},
		"QL-570": {
			Name:                "QL-570",
			SupportsSwitchMode:  true,
			SupportsCompression: false,
			NeedsQualitySetting: true,
			RasterWidthBytes:    90,
			InvalidateBytes:     400,
		},
		"QL-580N": {
			Name:                "QL-580N",
			SupportsSwitchMode:  true,
			SupportsCompression: true,
			NeedsQualitySetting: true,
			RasterWidthBytes:    90,
			InvalidateBytes:     400,
		},

		// QL-6xx series.
		"QL-650TD": {
			Name:                "QL-650TD",
			SupportsSwitchMode:  true,
			SupportsCompression: true,
			NeedsQualitySetting: true,
			RasterWidthBytes:    90,
			InvalidateBytes:     400,
		},

		// QL-7xx series: raster mode switch supported; most lack compression.
		"QL-700": {
			Name:                "QL-700",
			SupportsSwitchMode:  true,
			SupportsCompression: false,
			NeedsQualitySetting: true,
			RasterWidthBytes:    90,
			InvalidateBytes:     400,
		},
		"QL-710W": {
			Name:                "QL-710W",
			SupportsSwitchMode:  true,
			SupportsCompression: true,
			NeedsQualitySetting: true,
			RasterWidthBytes:    90,
			InvalidateBytes:     400,
		},
		"QL-720NW": {
			Name:                "QL-720NW",
			SupportsSwitchMode:  true,
			SupportsCompression: true,
			NeedsQualitySetting: true,
			RasterWidthBytes:    90,
			InvalidateBytes:     400,
		},

		// QL-8xx series: modern models, full raster mode support.
		"QL-800": {
			Name:                "QL-800",
			SupportsSwitchMode:  true,
			SupportsCompression: false,
			NeedsQualitySetting: true,
			RasterWidthBytes:    90,
			InvalidateBytes:     400,
		},
		"QL-810W": {
			Name:                "QL-810W",
			SupportsSwitchMode:  true,
			SupportsCompression: true,
			NeedsQualitySetting: true,
			RasterWidthBytes:    90,
			InvalidateBytes:     400,
		},
		"QL-820NWB": {
			Name:                "QL-820NWB",
			SupportsSwitchMode:  true,
			SupportsCompression: true,
			NeedsQualitySetting: true,
			RasterWidthBytes:    90,
			InvalidateBytes:     400,
		},

		// QL-1xxx series: wide-format models, 162 bytes per raster row.
		"QL-1050": {
			Name:                "QL-1050",
			SupportsSwitchMode:  true,
			SupportsCompression: true,
			NeedsQualitySetting: true,
			RasterWidthBytes:    162,
			InvalidateBytes:     400,
		},
		"QL-1060N": {
			Name:                "QL-1060N",
			SupportsSwitchMode:  true,
			SupportsCompression: true,
			NeedsQualitySetting: true,
			RasterWidthBytes:    162,
			InvalidateBytes:     400,
		},
		"QL-1100": {
			Name:                "QL-1100",
			SupportsSwitchMode:  true,
			SupportsCompression: true,
			NeedsQualitySetting: true,
			RasterWidthBytes:    162,
			InvalidateBytes:     400,
		},
		"QL-1110NWB": {
			Name:                "QL-1110NWB",
			SupportsSwitchMode:  true,
			SupportsCompression: true,
			NeedsQualitySetting: true,
			RasterWidthBytes:    162,
			InvalidateBytes:     400,
		},
	}
	if model, ok := models[name]; ok {
		return model, nil
	}
	return models["QL-800"], fmt.Errorf("printer model '%s' not explicitly defined, using defaults for QL-800", name)
}

// LabelSize describes a supported label format.
type LabelSize struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	DotsTotalWidth      int    `json:"dots_total_width"`
	DotsTotalHeight     int    `json:"dots_total_height"`
	DotsPrintableWidth  int    `json:"dots_printable_width"`
	DotsPrintableHeight int    `json:"dots_printable_height"`
	TapeSizeWidth       int    `json:"tape_size_width"`
	TapeSizeHeight      int    `json:"tape_size_height"`
	FeedMargin          int    `json:"feed_margin"` // feed margin in dots required by the protocol
	IsDieCut            bool   `json:"is_die_cut"`
	MediaCode           []byte `json:"-"`
}

// ListLabels returns all supported label sizes.
// FeedMargin values follow the reference Python driver: 0 for most die-cut labels,
// 35 (or 14) for continuous tape.
func ListLabels() []LabelSize {
	return []LabelSize{
		// Continuous tape (endless)
		{ID: "12", Name: "12mm endless", DotsTotalWidth: 142, DotsPrintableWidth: 106, TapeSizeWidth: 12, FeedMargin: 35, IsDieCut: false},
		{ID: "18", Name: "18mm endless", DotsTotalWidth: 256, DotsPrintableWidth: 234, TapeSizeWidth: 18, FeedMargin: 14, IsDieCut: false},
		{ID: "29", Name: "29mm endless", DotsTotalWidth: 342, DotsPrintableWidth: 306, TapeSizeWidth: 29, FeedMargin: 35, IsDieCut: false},
		{ID: "38", Name: "38mm endless", DotsTotalWidth: 449, DotsPrintableWidth: 413, TapeSizeWidth: 38, FeedMargin: 35, IsDieCut: false},
		{ID: "50", Name: "50mm endless", DotsTotalWidth: 590, DotsPrintableWidth: 554, TapeSizeWidth: 50, FeedMargin: 35, IsDieCut: false},
		{ID: "54", Name: "54mm endless", DotsTotalWidth: 636, DotsPrintableWidth: 590, TapeSizeWidth: 54, FeedMargin: 35, IsDieCut: false},
		{ID: "62", Name: "62mm endless", DotsTotalWidth: 732, DotsPrintableWidth: 696, TapeSizeWidth: 62, FeedMargin: 35, IsDieCut: false},
		{ID: "62red", Name: "62mm endless (black/red/white)", DotsTotalWidth: 732, DotsPrintableWidth: 696, TapeSizeWidth: 62, FeedMargin: 35, IsDieCut: false},
		{ID: "102", Name: "102mm endless", DotsTotalWidth: 1200, DotsPrintableWidth: 1164, TapeSizeWidth: 102, FeedMargin: 35, IsDieCut: false},
		{ID: "103", Name: "104mm endless", DotsTotalWidth: 1224, DotsPrintableWidth: 1200, TapeSizeWidth: 104, FeedMargin: 35, IsDieCut: false},

		// Die-cut labels (FeedMargin is 0 for most)
		{ID: "17x54", Name: "17mm x 54mm die-cut", DotsTotalWidth: 201, DotsTotalHeight: 636, DotsPrintableWidth: 165, DotsPrintableHeight: 566, TapeSizeWidth: 17, TapeSizeHeight: 54, FeedMargin: 0, IsDieCut: true},
		{ID: "17x87", Name: "17mm x 87mm die-cut", DotsTotalWidth: 201, DotsTotalHeight: 1026, DotsPrintableWidth: 165, DotsPrintableHeight: 956, TapeSizeWidth: 17, TapeSizeHeight: 87, FeedMargin: 0, IsDieCut: true},
		{ID: "23x23", Name: "23mm x 23mm die-cut", DotsTotalWidth: 272, DotsTotalHeight: 272, DotsPrintableWidth: 202, DotsPrintableHeight: 202, TapeSizeWidth: 23, TapeSizeHeight: 23, FeedMargin: 0, IsDieCut: true},
		{ID: "29x42", Name: "29mm x 42mm die-cut", DotsTotalWidth: 342, DotsTotalHeight: 495, DotsPrintableWidth: 306, DotsPrintableHeight: 425, TapeSizeWidth: 29, TapeSizeHeight: 42, FeedMargin: 0, IsDieCut: true},
		{ID: "29x90", Name: "29mm x 90mm die-cut", DotsTotalWidth: 342, DotsTotalHeight: 1061, DotsPrintableWidth: 306, DotsPrintableHeight: 991, TapeSizeWidth: 29, TapeSizeHeight: 90, FeedMargin: 0, IsDieCut: true},
		{ID: "39x90", Name: "38mm x 90mm die-cut", DotsTotalWidth: 449, DotsTotalHeight: 1061, DotsPrintableWidth: 413, TapeSizeHeight: 90, TapeSizeWidth: 38, FeedMargin: 0, IsDieCut: true},
		{ID: "39x48", Name: "39mm x 48mm die-cut", DotsTotalWidth: 461, DotsTotalHeight: 565, DotsPrintableWidth: 425, DotsPrintableHeight: 495, TapeSizeWidth: 39, TapeSizeHeight: 48, FeedMargin: 0, IsDieCut: true},
		{ID: "52x29", Name: "52mm x 29mm die-cut", DotsTotalWidth: 614, DotsTotalHeight: 341, DotsPrintableWidth: 578, DotsPrintableHeight: 271, TapeSizeWidth: 52, TapeSizeHeight: 29, FeedMargin: 0, IsDieCut: true},
		{ID: "54x29", Name: "54mm x 29mm die-cut", DotsTotalWidth: 630, DotsTotalHeight: 341, DotsPrintableWidth: 598, DotsPrintableHeight: 271, TapeSizeWidth: 54, TapeSizeHeight: 29, FeedMargin: 0, IsDieCut: true},
		{ID: "60x86", Name: "60mm x 87mm die-cut", DotsTotalWidth: 708, DotsTotalHeight: 1024, DotsPrintableWidth: 672, DotsPrintableHeight: 954, TapeSizeWidth: 60, TapeSizeHeight: 87, FeedMargin: 0, IsDieCut: true},
		{ID: "62x29", Name: "62mm x 29mm die-cut", DotsTotalWidth: 732, DotsTotalHeight: 341, DotsPrintableWidth: 696, DotsPrintableHeight: 271, TapeSizeWidth: 62, TapeSizeHeight: 29, FeedMargin: 0, IsDieCut: true},
		{ID: "62x100", Name: "62mm x 100mm die-cut", DotsTotalWidth: 732, DotsTotalHeight: 1179, DotsPrintableWidth: 696, DotsPrintableHeight: 1109, TapeSizeWidth: 62, TapeSizeHeight: 100, FeedMargin: 0, IsDieCut: true},
		{ID: "102x51", Name: "102mm x 51mm die-cut", DotsTotalWidth: 1200, DotsTotalHeight: 596, DotsPrintableWidth: 1164, DotsPrintableHeight: 526, TapeSizeWidth: 102, TapeSizeHeight: 51, FeedMargin: 0, IsDieCut: true},
		{ID: "102x152", Name: "102mm x 153mm die-cut", DotsTotalWidth: 1200, DotsTotalHeight: 1804, DotsPrintableWidth: 1164, DotsPrintableHeight: 1660, TapeSizeWidth: 102, TapeSizeHeight: 153, FeedMargin: 0, IsDieCut: true},
		{ID: "103x164", Name: "104mm x 164mm die-cut", DotsTotalWidth: 1224, DotsTotalHeight: 1941, DotsPrintableWidth: 1200, DotsPrintableHeight: 1822, TapeSizeWidth: 104, TapeSizeHeight: 164, FeedMargin: 0, IsDieCut: true},

		// Round die-cut labels
		{ID: "d12", Name: "12mm round die-cut", DotsTotalWidth: 142, DotsTotalHeight: 142, DotsPrintableWidth: 94, DotsPrintableHeight: 94, TapeSizeWidth: 12, TapeSizeHeight: 12, FeedMargin: 35, IsDieCut: true},
		{ID: "d24", Name: "24mm round die-cut", DotsTotalWidth: 284, DotsTotalHeight: 284, DotsPrintableWidth: 236, DotsPrintableHeight: 236, TapeSizeWidth: 24, TapeSizeHeight: 24, FeedMargin: 0, IsDieCut: true},
		{ID: "d58", Name: "58mm round die-cut", DotsTotalWidth: 688, DotsTotalHeight: 688, DotsPrintableWidth: 618, DotsPrintableHeight: 618, TapeSizeWidth: 58, TapeSizeHeight: 58, FeedMargin: 0, IsDieCut: true},
	}
}

// GetLabel returns the label size with the given ID.
func GetLabel(id string) (LabelSize, error) {
	for _, label := range ListLabels() {
		if label.ID == id {
			return label, nil
		}
	}
	return LabelSize{}, errors.New("label not found")
}
