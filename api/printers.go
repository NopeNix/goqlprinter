package api

import (
	"log/slog"
	"net/http"

	"goqlprinter/services"
	"github.com/gin-gonic/gin"
)

// GetPrinters godoc
// @Summary List printers
// @Description Returns available printers
// @Tags printers
// @Produce json
// @Success 200 {array} services.FoundPrinter
// @Router /printers [get]
func GetPrinters(c *gin.Context) {
	printers, err := services.FindPrinters()
	if err != nil {
		slog.Error("Error finding printers", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find printers"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"printers": printers})
}
