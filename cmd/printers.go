package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	isvc "goqlprinter/internal/services"

	"github.com/spf13/cobra"
)

var printersCmd = &cobra.Command{
	Use:   "printers",
	Short: "List connected Brother QL printers",
	RunE: func(cmd *cobra.Command, args []string) error {
		ps := isvc.NewPrinterService(BackendProvider)
		printers, err := ps.FindPrinters()
		if err != nil {
			return fmt.Errorf("failed to discover printers: %w", err)
		}
		if len(printers) == 0 {
			fmt.Println("No Brother QL printers found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "MODEL\tURI")
		for _, p := range printers {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", p.Model, p.UID)
		}
		_ = w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(printersCmd)
}
