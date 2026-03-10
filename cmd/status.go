package cmd

import (
	"fmt"
	"time"

	"goqlprinter/brotherql"
	isvc "goqlprinter/internal/services"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Query printer status",
	RunE: func(cmd *cobra.Command, args []string) error {
		ps := isvc.NewPrinterService(BackendProvider)
		ps.InitializeDefaultPrinter(Cfg.App.DefaultPrinter)

		printer := flagPrinter

		var result brotherql.PrinterStatus
		err := isvc.ConnectToPrinter(ps, printer, flagModel, func(backend brotherql.Backend, model string) error {
			type ReadTimeoutSetter interface {
				SetReadTimeout(d time.Duration)
			}

			if ts, ok := backend.(ReadTimeoutSetter); ok {
				ts.SetReadTimeout(150 * time.Millisecond)
			}
			drainBuf := make([]byte, 256)
			for {
				n, _ := backend.Read(drainBuf)
				if n == 0 {
					break
				}
			}
			if ts, ok := backend.(ReadTimeoutSetter); ok {
				ts.SetReadTimeout(3 * time.Second)
			}

			var cmdBuf []byte
			cmdBuf = append(cmdBuf, make([]byte, 200)...)
			cmdBuf = append(cmdBuf, 0x1B, 0x69, 0x53)
			if _, err := backend.Write(cmdBuf); err != nil {
				return fmt.Errorf("failed to send status request: %w", err)
			}

			var allData []byte
			tmpBuf := make([]byte, 64)
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				time.Sleep(100 * time.Millisecond)
				n, err := backend.Read(tmpBuf)
				if n > 0 {
					allData = append(allData, tmpBuf[:n]...)
					break
				}
				if err != nil && err.Error() != "EOF" {
					return fmt.Errorf("failed to read status: %w", err)
				}
			}

			if len(allData) < 32 {
				return fmt.Errorf("no response from printer (got %d bytes)", len(allData))
			}

			status, err := brotherql.ParseStatusResponse(allData[:32])
			if err != nil {
				return err
			}
			result = status
			return nil
		})
		if err != nil {
			return err
		}

		fmt.Printf("Ready:      %v\n", result.Ready)
		fmt.Printf("Busy:       %v\n", result.Busy)
		fmt.Printf("Error:      %v\n", result.Error)
		fmt.Printf("Media Type: %s\n", result.MediaType)
		fmt.Printf("Media Width: %d mm\n", result.MediaWidth)
		fmt.Printf("Media Length: %d mm\n", result.MediaLength)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
