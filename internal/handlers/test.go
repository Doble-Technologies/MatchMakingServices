package handlers

import (
	"log"
	"mm/service/internal/jobs"

	"github.com/gin-gonic/gin"
)

func TestPatchNotes(c *gin.Context) {
	err := jobs.ScrapeRiot()
	if err != nil {
		log.Printf("%s", err)
		c.JSON(500, gin.H{"result": err})

	}
	c.JSON(200, gin.H{"result": "AB"})
}
