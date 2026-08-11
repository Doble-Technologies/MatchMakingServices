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

// GetUserProfile GETs a user's profile based on the current user context
//
//	@Summary		Get User Profile by Current User Context
//	@Description	Retrieves and returns the profile details of the currently logged-in user.
//	@Tags			users
//	@Produce		json
//	@Success		200	{object}	views.UserProfile	"Returns the user profile"
//	@Failure		401	{object}	gin.H				"User not found in context"
//	@Failure		500	{object}	gin.H				"Internal server error"
//	@Router			/user/profile [get]
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

// GetUserProfileByUser GETs a user's profile by username
//
//	@Summary		Get User Profile by Username
//	@Description	Retrieves and returns the profile details of a user by their username.
//	@Tags			users
//	@Produce		json
//	@Param			username	path		string				true	"Username"
//	@Success		200			{object}	views.UserProfile	"Returns the user profile"
//	@Failure		404			{object}	gin.H				"User not found"
//	@Failure		500			{object}	gin.H				"Internal server error"
//	@Router			/user/profile/{username} [get]
func GetUserProfileByUser(c *gin.Context, h *ImageUploadHandler) {
	var userProfile views.UserProfile
	username := c.Param("username")
	result := initializer.DB.Table("user_details ud").
		Select("ud.user_id, u.username, ud.xp, ud.avatar, ud.bio, u.email, u.created_at").
		Where("? = u.username", username).
		Joins("JOIN users u ON u.id = ud.user_id").
		Scan(&userProfile)
	if result.Error != nil {
		log.Printf("%v", result)
		c.JSON(500, gin.H{"error": "user error"})
		return
	}
	if userProfile.UserID == 0 {
		c.JSON(404, gin.H{"error": "user not found"})
		return
	}
	userProfile.Avatar = GenerateUrl(userProfile.Avatar, c, h)
	c.JSON(200, gin.H{
		"profile": userProfile,
	})
}

// GetFriendsListById GETs a user's friends list by user ID
//
//	@Summary		Get Friends List by User ID
//	@Description	Retrieves and returns the list of friends for a given user ID.
//	@Tags			users
//	@Produce		json
//	@Param			id	path		string							true	"User ID"
//	@Success		200	{object}	gin.H{friends:[]models.Friend}	"Returns the list of friends"
//	@Failure		500	{object}	gin.H							"Internal server error"
//	@Router			/user/friends/{id} [get]
func GetFriendsListById(c *gin.Context) {
	id := c.Param("id")
	var friendsList []models.Friend

	initializer.DB.Where("user_id=?", id).
		Or("friend_user_id = ?", id).
		Find(&friendsList)

	c.JSON(200, gin.H{
		"friends": friendsList,
	})
}

// GetFriendsList GETs a user's friends list by current user context
//
//	@Summary		Get Friends List by Current User Context
//	@Description	Retrieves and returns the list of friends for the currently logged-in user.
//	@Tags			users
//	@Produce		json
//	@Success		200	{object}	gin.H{friends:[]models.Friend}	"Returns the list of friends"
//	@Failure		401	{object}	gin.H							"User not found in context"
//	@Failure		500	{object}	gin.H							"Invalid user type"
//	@Router			/user/friends [get]
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

// DeleteFriend DELETEs a friend from the user's friend list
//
//	@Summary		Delete Friend
//	@Description	Deletes a friend relationship from the current user's friend list.
//	@Tags			users
//	@Produce		json
//	@Param			body	body		inputs.FriendInput	true	"Friend details to delete"
//	@Success		200		{object}	gin.H				"Returns an empty response on success"
//	@Failure		401		{object}	gin.H				"Unauthorized operation"
//	@Failure		400		{object}	gin.H				"Invalid request format"
//	@Router			/user/friends [delete]
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

// CreateFriend POSTs a new friend to the user's friend list
//
//	@Summary		Create Friend
//	@Description	Creates a new friend relationship in the current user's friend list.
//	@Tags			users
//	@Produce		json
//	@Param			body	body		inputs.FriendInput	true	"Friend details"
//	@Success		201		{object}	gin.H				"Returns creation result on success"
//	@Failure		400		{object}	gin.H				"Invalid request format"
//	@Router			/user/friends [post]
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
		log.Printf("failed to create record %+v: %v\n", friend, result.Error)
		c.JSON(400, gin.H{"err": fmt.Sprintf("failed to create record %+v: %v", friend, result.Error)})
	} else {
		c.JSON(201, gin.H{"result": result})
	}
}

// EditFriend PUTs an updated status for a friend in the user's friend list
//
//	@Summary		Edit Friend Status
//	@Description	Updates the status of a friend relationship in the current user's friend list.
//	@Tags			users
//	@Produce		json
//	@Param			body	body		inputs.FriendInput	true	"Friend details with new status"
//	@Success		200		{object}	gin.H				"Returns updated status on success"
//	@Failure		400		{object}	gin.H				"Invalid request format"
//	@Router			/user/friends [put]
func EditFriend(c *gin.Context) {
	var friend inputs.FriendInput
	if err := c.ShouldBindJSON(&friend); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"invalid format": err.Error()})
		return
	}

	initializer.DB.Model(models.Friend{}).Where("user_id = ? and friend_user_id = ?", friend.UserID, friend.FriendUserID).Update("status", friend.Status)
	c.JSON(200, gin.H{"updated": friend.Status})
}

// GetNotifications GETs notifications for the current user
//
//	@Summary		Get Notifications by Current User
//	@Description	Retrieves and returns the list of notifications for the currently logged-in user.
//	@Tags			users
//	@Produce		json
//	@Success		200	{object}	gin.H{notifications:[]models.Notification}	"Returns the list of notifications"
//	@Router			/user/notifications [get]
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

// GetNotificationsByID GETs notifications for a specific user by ID
//
//	@Summary		Get Notifications by User ID
//	@Description	Retrieves and returns the list of notifications for a specified user by their ID.
//	@Tags			users
//	@Produce		json
//	@Param			id	path		string										true	"User ID"
//	@Success		200	{object}	gin.H{notifications:[]models.Notification}	"Returns the list of notifications"
//	@Failure		400	{object}	gin.H										"Missing user ID"
//	@Router			/user/notifications/{id} [get]
func GetNotificationsByID(c *gin.Context) {
	var notifications []models.Notification
	id := c.Param("id")

	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"err": "missing id"})
		return
	}

	initializer.DB.Where("user_id=?", id).Find(&notifications)

	c.JSON(200, gin.H{
		"notifications": notifications,
	})
}

// CreateNotifications POSTs multiple notifications to the database
//
//	@Summary		Create Notifications
//	@Description	Ingests and inserts multiple notification records into the notifications database.
//	@Tags			users
//	@Produce		json
//	@Param			body	body		[]inputs.NotificationInput								true	"List of Notification Inputs"
//	@Success		200		{object}	gin.H{notifications_sent:int,notifications_created:int}	"Returns the count of sent and created notifications"
//	@Router			/user/notifications [post]
func CreateNotifications(c *gin.Context) {
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
