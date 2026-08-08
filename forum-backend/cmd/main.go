package main

import (
	"fmt"
	"forum-backend/config"
	"forum-backend/controller"
	"forum-backend/database"
	"forum-backend/middleware"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	config.LoadConfig()
	logFile, err := middleware.InitLogger("logs/app.log")
	if err != nil {
		log.Printf("日志初始化失败: %v", err)
	} else {
		defer logFile.Close()
	}
	database.InitMySQL()
	database.InitRedis()
	database.AutoMigrate()
	r := gin.New()
	r.Use(middleware.RequestLogger(), middleware.Recovery())
	registerRoutes(r)
	r.Run(fmt.Sprintf(":%d", config.AppConfig.Server.Port))
}

func registerRoutes(r *gin.Engine) {
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	api := r.Group("/api/v1")

	api.POST(
		"/auth/register",
		controller.Register,
	)
	api.POST(
		"/auth/login",
		controller.Login,
	)

	auth := api.Group("")
	auth.Use(middleware.JWTAuth())
	auth.POST("/posts", controller.CreatePost)
	auth.GET("/posts", controller.GetPostList)
	auth.GET("/posts/:post_id", controller.GetPostDetail)
	auth.DELETE("/posts/:post_id", controller.DeletePost)
	auth.POST("/posts/:post_id/like", middleware.LikeRateLimit(), controller.LikePost)
	auth.POST("/posts/likes", controller.GetLikeStatus)
	auth.POST("/posts/:post_id/comment", controller.CreateComment)
	auth.POST("/agent/chat", controller.AgentChat)

	admin := api.Group("/admin")
	admin.Use(middleware.JWTAuth(), middleware.AdminAuth())
	admin.DELETE("/posts/:post_id", controller.AdminDeletePost)

}
