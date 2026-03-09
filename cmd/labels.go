package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"goqlprinter/brotherql"

	"github.com/spf13/cobra"
)

var labelsCmd = &cobra.Command{
	Use:   "labels",
	Short: "List available label sizes",
	Run: func(cmd *cobra.Command, args []string) {
		labels := brotherql.ListLabels()
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ID\tWIDTH(mm)\tHEIGHT(mm)\tTYPE")
		for _, l := range labels {
			labelType := "endless"
			if l.IsDieCut {
				labelType = "die-cut"
			}
			_, _ = fmt.Fprintf(w, "%s\t%d\t%d\t%s\n", l.ID, l.TapeSizeWidth, l.TapeSizeHeight, labelType)
		}
		_ = w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(labelsCmd)
}
