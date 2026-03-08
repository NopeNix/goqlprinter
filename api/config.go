package api

import (
	"net/http"

	"goqlprinter/config"
	"goqlprinter/services"
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
func GetConfig(c *gin.Context) {
	defaultPrinter := services.GetActiveDefaultPrinter()
	c.JSON(http.StatusOK, ConfigResponse{Config: config.Cfg, DefaultPrinter: defaultPrinter})
}
