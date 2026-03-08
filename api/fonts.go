package api

import (
	"net/http"

	"goqlprinter/services"
	"github.com/gin-gonic/gin"
)

// GetFonts godoc
// @Summary List available fonts
// @Description Returns list of installed fonts
// @Tags fonts
// @Produce json
// @Success 200 {array} string
// @Router /fonts [get]
func GetFonts(c *gin.Context) {
	fonts, err := services.ListFonts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"fonts": fonts})
}
