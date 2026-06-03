package handlers

import (
	"fmt"
	"log"
	"mm/service/internal/models"
	"mm/service/internal/models/inputs"
	"mm/service/internal/models/views"
	"mm/service/pkg/initializer"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetUserProfile(c *gin.Context, h *ImageUploadHandler) {
	var userProfile views.UserProfile
	userInterface, exists := c.Get("currentUser")
	if !exists {
		c.JSON(401, gin.H{"error": "user not found in context"})
		return
	}
	user, _ := userInterface.(models.User)

	log.Printf("%v", user.ID)
	result := initializer.DB.Table("user_details ud").
		Select("u.id, u.username, ud.xp, ud.avatar, ud.bio, u.email, u.created_at").
		Where("? = u.id", user.ID).
		Joins("JOIN users u ON u.id = ud.user_id").
		Scan(&userProfile)
	if result.Error != nil {
		log.Printf("%v", result)
		c.JSON(500, gin.H{"error": "user error"})
		return
	}

	userProfile.Avatar = GenerateUrl(userProfile.Avatar, c, h)
	c.JSON(200, gin.H{
		"profile": userProfile,
	})
}

func GetFriendsListById(c *gin.Context) {
	var id = c.Param("id")
	var friendsList []models.Friend

	initializer.DB.Where("user_id=?", id).
		Or("friend_user_id = ?", id).
		Find(&friendsList)

	c.JSON(200, gin.H{
		"friends": friendsList,
	})
}

func GetFriendsList(c *gin.Context) {
	userInterface, exists := c.Get("currentUser")
	if !exists {
		c.JSON(401, gin.H{"error": "user not found in context"})
		return
	}

	user, ok := userInterface.(models.User)
	if !ok {
		c.JSON(500, gin.H{"error": "invalid user type"})
		return
	}

	var friendsList []models.Friend

	initializer.DB.Where("user_id=?", user.ID).
		Or("friend_user_id = ?", user.ID).
		Find(&friendsList)

	c.JSON(200, gin.H{
		"friends": friendsList,
	})
}

func DeleteFriend(c *gin.Context) {
	userInterface, exists := c.Get("currentUser")
	if !exists {
		c.JSON(401, gin.H{"error": "user not found in context"})
		return
	}

	user, ok := userInterface.(models.User)
	if !ok {
		c.JSON(500, gin.H{"error": "invalid user type"})
		return
	}

	var friend models.Friend

	if err := c.ShouldBindJSON(&friend); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if friend.UserID != user.ID && friend.FriendUserID != user.ID {
		c.JSON(401, gin.H{})

	}

	initializer.DB.Delete(&models.Friend{}, "user_id = ? and friend_user_id = ?", friend.UserID, friend.FriendUserID)

	c.JSON(200, gin.H{})
}
func CreateFriend(c *gin.Context) {
	var friend inputs.FriendInput
	if err := c.ShouldBindJSON(&friend); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"invalid format": err.Error()})
		return
	}
	if friend.Status == "" {
		friend.Status = "pending"
	}
	result := initializer.DB.Create(friend)
	if result.Error != nil {
		//TODO: Parse error so use cant see details
		log.Printf("failed to create record %+v: %v\n", friend, result.Error)
		c.JSON(400, gin.H{"err": fmt.Sprintf("failed to create record %+v: %v", friend, result.Error)})
	} else {
		c.JSON(201, gin.H{"result": result})
	}
}

func EditFriend(c *gin.Context) {
	var friend inputs.FriendInput
	if err := c.ShouldBindJSON(&friend); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"invalid format": err.Error()})
		return
	}

	initializer.DB.Model(models.Friend{}).Where("user_id = ? and friend_user_id = ?", friend.UserID, friend.FriendUserID).Update("status", friend.Status)
	c.JSON(200, gin.H{"updated": friend.Status})
}

func GetNotifications(c *gin.Context) {
	var notifications []models.Notification
	userInterface, _ := c.Get("currentUser")

	var user models.User
	user, _ = userInterface.(models.User)

	initializer.DB.Where("user_id=?", user.ID).Find(&notifications)

	c.JSON(200, gin.H{
		"notifications": notifications,
	})
}

func GetNotificationsByID(c *gin.Context) {
	var notifications []models.Notification
	var id = c.Param("id")

	initializer.DB.Where("user_id=?", id).Find(&notifications)

	c.JSON(200, gin.H{
		"notifications": notifications,
	})
}

// CreateNotifications Post
// @Summary Create and store notifications
// @Description Ingest X Number of Notifications and insert them to notifications db
// @Produce json
// @Success 200 {object} map[string]string
// @Router /generate/CreateNotifications [post]
func CreateNotifications(c *gin.Context) {
	//Notification Inputs
	var notInputs []inputs.NotificationInput
	if err := c.ShouldBindJSON(&notInputs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"invalid format": err.Error()})
		return
	}

	var count = 0
	for _, v := range notInputs {
		result := initializer.DB.Create(v)
		if result.Error != nil {
			log.Printf("failed to create record %+v: %v\n", v, result.Error)
			continue
		}
		count++
	}

	c.JSON(200, gin.H{
		"notifications_sent":    len(notInputs),
		"notifications_created": count,
	})
}
