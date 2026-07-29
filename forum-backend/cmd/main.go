package main

import (
	"forum-backend/config"
	"forum-backend/controller"
	"forum-backend/database"
	"forum-backend/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {

	// 加载配置
	config.LoadConfig()

	database.InitMySQL()

	database.AutoMigrate()

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// 测试JWT
	auth := r.Group("/api/test")

	auth.Use(middleware.JWTAuth())

	auth.GET("/auth", func(c *gin.Context) {

		c.JSON(200, gin.H{
			"message": "验证成功",
		})

	})

	// 用户接口
	r.POST("/api/user/register", controller.Register)

	r.POST("/api/user/login", controller.Login)

	// 帖子公开接口
	r.GET("/api/post/list", controller.GetPostList)

	r.GET("/api/post/:id", controller.GetPostDetail)

	// 帖子登录接口
	postAuth := r.Group("/api/post")

	postAuth.Use(middleware.JWTAuth())

	postAuth.POST("/create", controller.CreatePost)

	postAuth.DELETE("/:id", controller.DeletePost)

	postAuth.PUT("/:id", controller.UpdatePost)

	r.Run(":8080")
}
