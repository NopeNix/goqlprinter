package api

import (
	"net/http"

	"goqlprinter/brotherql"
	"github.com/gin-gonic/gin"
)

// GetLabelSizes godoc
// @Summary List label sizes
// @Description Returns all available label sizes
// @Tags labels
// @Produce json
// @Success 200 {array} brotherql.LabelSize
// @Router /label-sizes [get]
func (h *Handlers) GetLabelSizes(c *gin.Context) {
	labelSizes := brotherql.ListLabels()
	c.JSON(http.StatusOK, gin.H{"label_sizes": labelSizes})
}

// GetLabelSize handles the GET /label-sizes/:id endpoint
func (h *Handlers) GetLabelSize(c *gin.Context) {
	id := c.Param("id")
	label, err := brotherql.GetLabel(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, label)
}
