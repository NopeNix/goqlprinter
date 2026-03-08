package api

import (
	"net/http"

	"goqlprinter/internal/config"
	"goqlprinter/internal/services"
	"github.com/gin-gonic/gin"
)

// ConfigResponse defines the structure for the /config endpoint response
type ConfigResponse struct {
	Config         config.Config          `json:"config"`
	DefaultPrinter *services.FoundPrinter `json:"default_printer"`
}

// GetConfig godoc
// @Summary Get server configuration
// @Description Returns current server configuration and default printer
// @Tags config
// @Produce json
// @Success 200 {object} ConfigResponse
// @Router /config [get]
func (h *Handlers) GetConfig(c *gin.Context) {
	defaultPrinter := h.Printers.GetDefaultPrinter()
	c.JSON(http.StatusOK, ConfigResponse{Config: *h.Config, DefaultPrinter: defaultPrinter})
}
