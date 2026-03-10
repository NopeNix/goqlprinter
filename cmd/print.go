package cmd

import (
	"fmt"
	"image"
	"strings"

	"goqlprinter/brotherql"
	isvc "goqlprinter/internal/services"

	"github.com/spf13/cobra"
)

var (
	flagLabelSize    string
	flagFont         string
	flagFontSize     float64
	flagOrientation  string
	flagHAlign       string
	flagVAlign       string
	flagRotation     float64
	flagCustomHeight float64
)

var printCmd = &cobra.Command{
	Use:   "print [text...]",
	Short: "Print a text label on a Brother QL printer",
	Long:  "Renders text onto a label and sends it to the connected Brother QL printer.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runPrint,
}

func init() {
	printCmd.Flags().StringVarP(&flagLabelSize, "label-size", "l", "", "label size identifier (required)")
	printCmd.Flags().StringVar(&flagFont, "font", "Roboto", "font family name")
	printCmd.Flags().Float64Var(&flagFontSize, "font-size", 12, "font size in points")
	printCmd.Flags().StringVar(&flagOrientation, "orientation", "standard", "label orientation: standard or rotated")
	printCmd.Flags().StringVar(&flagHAlign, "halign", "start", "horizontal alignment: start, center, or end")
	printCmd.Flags().StringVar(&flagVAlign, "valign", "start", "vertical alignment: start, center, or end")
	printCmd.Flags().Float64Var(&flagRotation, "rotation", 0, "text rotation angle in degrees (0, 90, 180, 270)")
	printCmd.Flags().Float64Var(&flagCustomHeight, "custom-height", 0, "custom tape height in mm (endless tape only)")

	_ = printCmd.MarkFlagRequired("label-size")

	rootCmd.AddCommand(printCmd)
}

// mmToDots converts millimeters to dots at 300 DPI.
func mmToDots(mm float64) int {
	return int(mm * 300.0 / 25.4)
}

func runPrint(cmd *cobra.Command, args []string) error {
	text := strings.Join(args, " ")

	label, err := brotherql.GetLabel(flagLabelSize)
	if err != nil {
		return fmt.Errorf("invalid label size %q: %w", flagLabelSize, err)
	}

	fontSvc := isvc.NewFontService(Cfg.App.FontDirs)
	fontPath, err := fontSvc.GetFontPath(flagFont)
	if err != nil {
		return fmt.Errorf("font %q not found: %w", flagFont, err)
	}

	img, err := renderText(text, label, fontPath)
	if err != nil {
		return fmt.Errorf("render failed: %w", err)
	}

	printerSvc := isvc.NewPrinterService(BackendProvider)
	if Cfg.App.DefaultPrinter != "" {
		printerSvc.InitializeDefaultPrinter(Cfg.App.DefaultPrinter)
	}

	err = isvc.ConnectToPrinter(printerSvc, flagPrinter, flagModel, func(backend brotherql.Backend, model string) error {
		dev := brotherql.NewBrotherQL(model, backend)
		return dev.Print(img, label)
	})
	if err != nil {
		return fmt.Errorf("print failed: %w", err)
	}

	fmt.Println("Label printed successfully.")
	return nil
}

// renderText renders text onto a grayscale image sized for the given label,
// using the same pipeline as the API handler.
func renderText(text string, label brotherql.LabelSize, fontPath string) (*image.Gray, error) {
	padding := 10
	isRotated := flagOrientation == "rotated"

	scaledFontSize := flagFontSize * 4
	unrotatedWidth, unrotatedHeight, err := brotherql.MeasureText(text, fontPath, scaledFontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to measure text: %w", err)
	}

	// Swap dimensions when rotated 90 or 270.
	var textBoundsWidth, textBoundsHeight int
	if flagRotation == 90 || flagRotation == 270 {
		textBoundsWidth = unrotatedHeight
		textBoundsHeight = unrotatedWidth
	} else {
		textBoundsWidth = unrotatedWidth
		textBoundsHeight = unrotatedHeight
	}

	printHeadDots := label.DotsPrintableWidth

	var tapeLengthDots int
	if flagCustomHeight > 0 && !label.IsDieCut {
		tapeLengthDots = mmToDots(flagCustomHeight)
	} else {
		tapeLengthDots = label.DotsPrintableHeight
	}

	var canvasWidth, canvasHeight int
	if isRotated {
		canvasHeight = printHeadDots
		if tapeLengthDots > 0 {
			canvasWidth = tapeLengthDots
		} else {
			canvasWidth = textBoundsWidth + (2 * padding)
		}
	} else {
		canvasWidth = printHeadDots
		if tapeLengthDots > 0 {
			canvasHeight = tapeLengthDots
		} else {
			canvasHeight = textBoundsHeight + (2 * padding)
		}
	}

	img := brotherql.CreateBlankImage(canvasWidth, canvasHeight)

	var x int
	switch flagHAlign {
	case "center":
		x = (canvasWidth - textBoundsWidth) / 2
	case "end":
		x = canvasWidth - textBoundsWidth - padding
	default:
		x = padding
	}

	var y int
	switch flagVAlign {
	case "center":
		y = (canvasHeight - textBoundsHeight) / 2
	case "end":
		y = canvasHeight - textBoundsHeight - padding
	default:
		y = padding
	}

	err = brotherql.DrawText(img, text, fontPath, scaledFontSize, x, y, flagRotation, "left")
	if err != nil {
		return nil, fmt.Errorf("failed to draw text: %w", err)
	}

	if isRotated {
		img = brotherql.RotateImage(img, 90)
	}

	return img, nil
}
