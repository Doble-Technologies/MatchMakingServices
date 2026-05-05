package server

import (
	"context"
	"log"
	_ "mm/service/docs"
	"mm/service/internal/app"
	"mm/service/internal/handlers"
	"mm/service/internal/jobs"
	"mm/service/internal/middleware"
	"mm/service/pkg/initializer"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

func init() {
	initializer.LoadEnvs()
	initializer.ConnectDB()

}

// TODO: Get Notification based on player id not jks player id
// Todo: Finish setting up Swagger
func setupRoutes(r *gin.Engine, app *app.App) {

	//Setup Redis Handler
	mmHandler := handlers.NewMMHandler(app.Redis)
	ver1 := r.Group("/api")
	{
		ver1.GET("/", handlers.HealthCheck)
		ver1.GET("/queue", middleware.CheckAuth, mmHandler.MatchmakingWs)

		ver1.POST("/auth/signup", handlers.CreateUser)
		ver1.POST("/auth/login", handlers.Login)
		ver1.POST("/auth/refresh", middleware.CheckAuth, handlers.AuthRefresh)

		ver1.GET("/user/profile", middleware.CheckAuth, handlers.GetUserProfile)

		ver1.GET("/user/notifications", middleware.CheckAuth, handlers.GetNotifications)
		ver1.GET("/user/notificationsbyid/:id", middleware.CheckAuth, handlers.GetNotificationsByID)
		ver1.GET("/friends/:id", middleware.CheckAuth, handlers.GetFriendsListById)
		ver1.GET("/friends/", middleware.CheckAuth, handlers.GetFriendsList)

		ver1.POST("/friends/delete/", middleware.CheckAuth, handlers.DeleteFriend)
		ver1.PUT("/friends/status/", middleware.CheckAuth, handlers.EditFriend)

		ver1.POST("/generate/friend/", middleware.CheckAuth, handlers.CreateFriend)
		ver1.POST("/generate/notifications", middleware.CheckAuth, handlers.CreateNotifications)
	}

	//TODO: Finish Swagger Setup
	//r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
}

func Start(addr string) {
	r := gin.Default()
	rdb := initializer.RedisClient()

	goApp := &app.App{
		Redis: rdb,
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	jm := jobs.NewJobManager(ctx, rdb)
	jm.RegisterJob(jobs.AlphaMatchJob{}) // Job every 2 minutes

	go jm.StartScheduler()

	setupRoutes(r, goApp)

	go func() {
		<-ctx.Done()
		log.Println("Shutting down gracefully...")
		cancel()
	}()
	log.Fatal(r.Run(addr))
}
