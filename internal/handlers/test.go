package handlers

import (
	"mm/service/internal/jobs"

	"github.com/gin-gonic/gin"
)

func TestPatchNotes(c *gin.Context) {
	err := jobs.ScrapeRiot()
	if err != nil {
		return
	}
	c.JSON(200, gin.H{"result": "AB"})
}
