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

func setupRoutes(r *gin.Engine, app *app.App) {

	//Setup Redis Handler
	mmHandler := handlers.NewMMHandler(app.Redis)

	r.GET("/", handlers.HealthCheck)
	r.GET("/queue", middleware.CheckAuth, mmHandler.MatchmakingWs)

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
	rdb := initializer.RedisClient()

	goApp := &app.App{
		Redis: rdb,
	}
	//Todo: Rewrite the contex handling, so that the goroutine is properly closed out
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
