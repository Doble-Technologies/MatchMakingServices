package view

import (
	"log"
	"mm/service/internal/models"
	"mm/service/internal/models/views"
	"mm/service/pkg/initializer"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

// Public facing needs to be more secure than ones requiring login
func GetLatestUsers(c *gin.Context) {
	var users []models.UserDetail
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

func GetFriendsListById(c *gin.Context) {
	var friendList []views.FriendDetail
	var id = c.Param("id")
	regEx := regexp.MustCompile(`^\d+$`)

	if !regEx.MatchString(id) {
		c.JSON(http.StatusBadRequest, gin.H{"Invalid ID": ""})
		return
	}

	result := initializer.DB.
		Table("friends AS f").
		Select("f.*, user_details.*").
		Joins("INNER JOIN user_details ON f.user_id = user_details.user_id").
		Where("(f.user_id = ? OR f.friend_user_id = ?) AND f.status = ?", id, id, "accepted").
		Scan(&friendList).Error
	if result.Error != nil {
		log.Printf("Database error: %v\n", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch users from database",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"users": friendList,
	})
}
