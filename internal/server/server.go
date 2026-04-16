package server

import (
	"log"
	_ "mm/service/docs"
	"mm/service/internal/handlers"
	"mm/service/internal/middleware"
	"mm/service/pkg/initializer"

	"github.com/gin-gonic/gin"
)

func init() {
	initializer.LoadEnvs()
	initializer.ConnectDB()

}

func setupRoutes(r *gin.Engine) {
	r.GET("/", handlers.HealthCheck)
	r.GET("/queue", middleware.CheckAuth, handlers.MatchmakingWs)

	r.POST("/auth/signup", handlers.CreateUser)
	r.POST("/auth/login", handlers.Login)
	r.POST("/auth/refresh", middleware.CheckAuth, handlers.AuthRefresh)

	r.GET("/user/profile", middleware.CheckAuth, handlers.GetUserProfile)
	r.GET("/user/notifications", middleware.CheckAuth, handlers.GetNotifications)
	r.GET("/user/notificationsbyid/:id", middleware.CheckAuth, handlers.GetNotificationsByID)

	r.POST("/generate/notifications", middleware.CheckAuth, handlers.CreateNotifications)

	//TODO: Finish Swagger Setup
	//r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
}

func Start(addr string) {
	r := gin.Default()
	setupRoutes(r)
	log.Fatal(r.Run(addr))
}
