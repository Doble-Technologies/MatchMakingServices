package view

import (
	"log"
	"mm/service/internal/models"
	"mm/service/pkg/initializer"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Public facing needs to be more secure than ones requiring login
func GetLatestUsers(c *gin.Context) {
	var users []models.User
	result := initializer.DB.Table("user_details").Order("created_at desc").Limit(5).Find(&users)
	//Todo: create custom view objects based on a few tables aggregated data
	if result.Error != nil {
		log.Printf("Database error: %v\n", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch users from database",
		})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "No users found",
			"users":   []models.User{},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"users": users,
	})
}
