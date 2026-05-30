package handlers

import (
	"mm/service/internal/middleware"
	"mm/service/internal/models"
	"mm/service/internal/models/inputs"
	"mm/service/pkg/initializer"
	"net/http"
	"net/mail"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

func validateSingleAddress(value string) error {
	_, err := mail.ParseAddress(value)
	return err
}
func Login(c *gin.Context) {

	var loginInput inputs.LoginInput

	if err := c.ShouldBindJSON(&loginInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userFound models.User
	initializer.DB.Where("username=?", loginInput.Username).Find(&userFound)

	if userFound.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user not found"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(userFound.PasswordHash), []byte(loginInput.Password)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid password"})
		return
	}

	generateToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  userFound.ID,
		"exp": time.Now().Add(time.Hour * 1).Unix(),
	})
	//Refresh is 512
	refreshGenerateToken := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"id":     userFound.ID,
		"issuer": "rcslabs",
		"exp":    time.Now().Add(time.Hour * 720).Unix(),
	}) //30 days

	token, err := generateToken.SignedString([]byte(os.Getenv("SECRET")))
	refreshToken, refreshErr := refreshGenerateToken.SignedString([]byte(os.Getenv("REFRESH_SECRET")))

	refreshSession := models.Session{
		UserID:    userFound.ID,
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(time.Hour * 720),
		CreatedAt: time.Now(),
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to generate token"})
	} else if refreshErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to generate refresh token"})
	}
	initializer.DB.Create(&refreshSession)

	c.SetCookie("refresh-token", refreshToken, 3600, "/", "localhost", false, true)

	c.JSON(200, gin.H{
		"token": token,
	})
}

func CreateUser(c *gin.Context) {

	var authInput inputs.AuthInput

	if err := c.ShouldBindJSON(&authInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userFound models.User
	initializer.DB.Where("username=?", authInput.Username).Find(&userFound)

	if userFound.ID != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username already used"})
		return
	}

	// regEx := regexp.MustCompile(`^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[^\w\s]).{8,64}$`)

	// if !regEx.MatchString(authInput.Password) {
	// 	c.JSON(http.StatusBadRequest, gin.H{"Invalid Password": ""})
	// 	return
	// }
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(authInput.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	//re := regexp.MustCompile(`^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`)

	_, err = mail.ParseAddress(authInput.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"Invalid Email": err.Error()})
		return
	}

	user := models.User{
		Username:     authInput.Username,
		PasswordHash: string(passwordHash),
		Email:        authInput.Email,
	}

	initializer.DB.Create(&user)
	//Todo check error message
	if user.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"Invalid Email/User": ""})
		return

	}

	userData := models.UserDetail{
		UserID: user.ID,
		Xp:     0,
		Avatar: "",
		Bio:    "",
	}
	initializer.DB.Create(&userData)

	c.JSON(http.StatusOK, gin.H{"data": user})

}

func AuthRefresh(c *gin.Context) {
	middleware.CheckRefresh(c)
}

func HealthCheck(c *gin.Context) {
	c.String(http.StatusOK, "Alive and Thriving")
}
