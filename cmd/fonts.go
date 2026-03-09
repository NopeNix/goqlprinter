package cmd

import (
	"fmt"

	isvc "goqlprinter/internal/services"

	"github.com/spf13/cobra"
)

var fontsCmd = &cobra.Command{
	Use:   "fonts",
	Short: "List available fonts",
	RunE: func(cmd *cobra.Command, args []string) error {
		fs := isvc.NewFontService(Cfg.App.FontDirs)
		fonts, err := fs.ListFonts()
		if err != nil {
			return err
		}
		if len(fonts) == 0 {
			fmt.Println("No fonts found.")
			return nil
		}
		for _, f := range fonts {
			fmt.Println(f)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(fontsCmd)
}
