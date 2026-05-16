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

// GetLatestUsers Public facing needs to be more secure than ones requiring login
func GetLatestUsers(c *gin.Context) {
	var userView []views.UserView

	result := initializer.DB.Table("user_details ud").
		Select("ud.user_id, u.username, ud.xp, ud.avatar, ud.bio").
		Joins("JOIN users u ON u.id = ud.user_id").
		Order("ud.created_at DESC").
		Limit(5).
		Scan(&userView)

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
			"users":   []models.UserDetail{},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": userView,
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

	result := initializer.DB.Table("users u").
		Select("u.id as user_id, u.username as username").
		Joins("JOIN friends f ON u.id = CASE WHEN f.user_id = ? THEN f.friend_user_id ELSE f.user_id END", id).
		Where("? IN (f.user_id, f.friend_user_id)", id).
		Where("status = 'accepted'").
		Scan(&friendList)
	log.Printf("%v", friendList)
	if result.Error != nil {
		log.Printf("Database error: %v\n", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch users from database",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"friends_list": friendList,
	})
}
