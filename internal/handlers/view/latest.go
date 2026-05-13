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
			"users":   []models.UserDetail{},
		})
		return
	}
	//Build the user view from here
	//for _, v := range users {
	//temp:
	//	initializer.DB.Select()
	//}

	c.JSON(http.StatusOK, gin.H{
		"users": userView,
	})
}

func GetFriendsListById(c *gin.Context) {
	var friends []models.Friend
	var friendList []views.FriendDetail
	var id = c.Param("id")
	regEx := regexp.MustCompile(`^\d+$`)

	if !regEx.MatchString(id) {
		c.JSON(http.StatusBadRequest, gin.H{"Invalid ID": ""})
		return
	}

	result := initializer.DB.
		Where("(user_id = ? OR friend_user_id = ?) AND status = ?", id, id, "accepted").
		Find(&friends).Error

	if result.Error != nil {
		log.Printf("Database error: %v\n", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch users from database",
		})
		return
	}

	//var friendIds []int

	//Iterate through friends and build friend details
	//for _, v := range friends {
	//	if v.UserID != 11 {
	//		append(friendIds, 11)
	//
	//	} else if v.FriendUserID != 11 {
	//		append(friendIds, 11)
	//	}
	//}

	c.JSON(http.StatusOK, gin.H{
		"friendsList": friendList,
	})
}
