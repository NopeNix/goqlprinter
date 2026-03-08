package api

import (
	"goqlprinter/internal/config"
	"goqlprinter/internal/services"
)

// Handlers holds all dependencies for the API layer.
type Handlers struct {
	Printers *services.PrinterService
	Fonts    *services.FontService
	Config   *config.Config
}

// NewHandlers creates a new Handlers with all dependencies injected.
func NewHandlers(ps *services.PrinterService, fs *services.FontService, cfg *config.Config) *Handlers {
	return &Handlers{Printers: ps, Fonts: fs, Config: cfg}
}
