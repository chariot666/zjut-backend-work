package controller

import (
	"encoding/json"
	"forum-backend/database"
	"forum-backend/model"
	"forum-backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreatePost(c *gin.Context) {
	var post model.Post
	if err := c.ShouldBindJSON(&post); err != nil {
		c.JSON(400, gin.H{
			"message": "参数错误",
		})
		return
	}
	userID, _ := c.Get("user_id")
	post.UserID = userID.(uint)
	database.DB.Create(&post)
	utils.Success(c, gin.H{
		"post": post,
	})
}
func GetPostList(c *gin.Context) {
	var posts []model.Post
	result := database.DB.
		Order("created_at desc").
		Find(&posts)
	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "获取帖子失败",
		})
		return
	}
	utils.Success(c, gin.H{
		"posts": posts,
	})
}
func GetPostDetail(c *gin.Context) {
	id := c.Param("id")
	// Redis key
	key := "post:" + id
	// 查询Redis
	val, err := database.RDB.Get(
		database.Ctx,
		key,
	).Result()
	if err == nil {
		var post model.Post
		json.Unmarshal(
			[]byte(val),
			&post,
		)
		c.JSON(200, gin.H{
			"source": "redis",
			"post":   post,
		})
		return
	}
	// Redis没有，查询MySQL
	var post model.Post
	result := database.DB.First(
		&post,
		id,
	)
	if result.Error != nil {
		utils.Error(
			c,
			404,
			"没有帖子",
		)
		return
	}
	// 浏览量+1
	database.DB.Model(&model.Post{}).
		Where("id=?", id).
		UpdateColumn(
			"view_count",
			gorm.Expr("view_count + ?", 1),
		)
	post.ViewCount++
	// 写入Redis
	data, _ := json.Marshal(post)
	database.RDB.Set(
		database.Ctx,
		key,
		data,
		time.Minute*10,
	)
	utils.Success(c, gin.H{
		"source": "mysql",
		"post":   post,
	})
}
func DeletePost(c *gin.Context) {
	// 获取登录用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "未登录",
		})
		return
	}
	// 获取帖子ID
	id := c.Param("id")
	var post model.Post
	// 查询帖子
	result := database.DB.First(&post, id)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "帖子不存在",
		})
		return
	}
	// 判断是不是自己的帖子
	if post.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{
			"message": "无权删除",
		})
		return
	}
	// 删除
	database.DB.Delete(&post)
	database.RDB.Del(
		database.Ctx,
		"post:"+id,
	)
	utils.Success(c, gin.H{
		"message": "删除成功",
	})
}
func UpdatePost(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "未登录",
		})
		return
	}
	id := c.Param("id")
	var post model.Post
	// 查询帖子
	result := database.DB.First(&post, id)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "帖子不存在",
		})
		return
	}
	// 判断作者
	if post.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{
			"message": "无权修改",
		})
		return
	}
	// 接收修改内容
	var updateData struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		utils.Error(
			c,
			http.StatusBadRequest,
			"参数错误",
		)
		return
	}
	// 更新
	post.Title = updateData.Title
	post.Content = updateData.Content
	database.DB.Save(&post)
	database.RDB.Del(
		database.Ctx,
		"post:"+id,
	)
	utils.Success(c, gin.H{
		"post": post,
	})
}
