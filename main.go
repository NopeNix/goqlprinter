// @title Label Printer API
// @version 1.0
// @description API for controlling Brother QL label printers
// @BasePath /api
package main

import (
	"embed"

	"goqlprinter/cmd"
	_ "goqlprinter/docs" // Swagger docs
)

//go:embed all:frontend/dist
var embeddedFiles embed.FS

func main() {
	cmd.EmbeddedFiles = embeddedFiles
	cmd.Execute()
}
