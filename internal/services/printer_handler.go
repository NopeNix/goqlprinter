package services

import "goqlprinter/brotherql"

// PrinterHandler defines the function signature for printer operations.
// Uses Backend interface instead of concrete USBBackend type.
type PrinterHandler func(backend brotherql.Backend, model string) error
