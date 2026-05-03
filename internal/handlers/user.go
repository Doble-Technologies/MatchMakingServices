package handlers

import (
	"log"
	"mm/service/internal/models"
	"mm/service/pkg/initializer"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetUserProfile(c *gin.Context) {

	user, _ := c.Get("currentUser")
	
	c.JSON(200, gin.H{
		"user": user,
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

	var id = c.Param("id")

	//user, _ := c.Get("currentUser")

	c.JSON(200, gin.H{
		"user2": id,
	})
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
	var notInputs []models.NotificationInput
	log.Println("Inside")
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
