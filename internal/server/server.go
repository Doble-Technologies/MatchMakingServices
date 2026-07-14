package server

import (
	"context"
	"log"
	_ "mm/service/docs"
	"mm/service/internal/app"
	"mm/service/internal/handlers"
	"mm/service/internal/handlers/sse"
	"mm/service/internal/handlers/view"
	"mm/service/internal/jobs"
	"mm/service/internal/middleware"
	"mm/service/pkg/initializer"
	"mm/service/pkg/s3"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	initializer.LoadEnvs()
	initializer.ConnectDB()
}

// TODO: Get Notification based on player id not jks player id
// Todo: Finish setting up Swagger
func setupRoutes(r *gin.Engine, app *app.App) {

	ch := make(chan string)
	//Setup Redis Handler
	mmHandler := handlers.NewMMHandler(app.Redis)
	s3Client, err := s3.NewGarageClient(
		os.Getenv("GARAGE_ENDPOINT"), // e.g. "http://localhost:3900"
		os.Getenv("GARAGE_ACCESS_KEY"),
		os.Getenv("GARAGE_SECRET_KEY"),
		"garage", // any non-empty string works as the region
	)
	if err != nil {
		log.Fatal(err)
	}

	uploadHandler := handlers.NewImageUploadHandler(s3Client)

	limiter := middleware.NewIPRateLimiter(10, 20, 2*time.Minute)

	r.Use(middleware.RateLimiter(limiter))
	ver1 := r.Group("/api")
	{
		ver1.GET("/", handlers.HealthCheck)
		ver1.GET("/queue", middleware.CheckAuth, mmHandler.MatchmakingWs)

		ver1.POST("/auth/signup", handlers.CreateUser)
		ver1.POST("/auth/login", handlers.Login)
		ver1.POST("/auth/refresh", middleware.CheckAuth, handlers.AuthRefresh)

		ver1.GET("/user/profile", middleware.CheckAuth, func(c *gin.Context) {
			handlers.GetUserProfile(c, uploadHandler)
		})

		ver1.GET("/user/profile/:username", func(c *gin.Context) {
			handlers.GetUserProfileByUser(c, uploadHandler)
		})

		ver1.GET("/user/notifications", middleware.CheckAuth, handlers.GetNotifications)
		ver1.GET("/user/notificationsbyid/:id", middleware.CheckAuth, handlers.GetNotificationsByID)

		//Todo: Fix This
		ver1.GET("/friends/:id", middleware.CheckAuth, handlers.GetFriendsListById)
		ver1.GET("/friends/", middleware.CheckAuth, handlers.GetFriendsList)

		//todo: protect this
		ver1.POST("/friends/delete/", middleware.CheckAuth, handlers.DeleteFriend)
		ver1.PUT("/friends/status/", middleware.CheckAuth, handlers.EditFriend)

		ver1.POST("/generate/friend/", middleware.CheckAuth, handlers.CreateFriend)
		ver1.POST("/generate/notifications", middleware.CheckAuth, handlers.CreateNotifications)

		//TODO: Finish SSE Events
		ver1.POST("/event-stream", func(c *gin.Context) {
			sse.HandleEventStreamPost(c, ch)
		})
		ver1.GET("/event-stream", func(c *gin.Context) {
			sse.HandleEventStreamGet(c, ch)
		})

		//Below are public views

		ver1.GET("/view/friendslist/:id", view.GetFriendsListById)
		ver1.GET("/view/latest/users", func(c *gin.Context) {
			view.GetLatestUsers(c, uploadHandler)
		})

		ver1.POST("/users/avatar/upload", middleware.CheckAuth, uploadHandler.UploadImage)
		ver1.GET("/users/avatar/fetch", uploadHandler.GetImageURL)

	}
	//
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
	jm.RegisterJob(jobs.PatchNotesJob{})
	go jm.StartScheduler()

	setupRoutes(r, goApp)

	go func() {
		<-ctx.Done()
		log.Println("Shutting down gracefully..")
		cancel()
	}()
	log.Fatal(r.Run(addr))
}
