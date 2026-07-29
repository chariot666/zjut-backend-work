package controller

import (
	"net/http"

	"forum-backend/database"
	"forum-backend/model"

	"github.com/gin-gonic/gin"
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

	c.JSON(http.StatusOK, gin.H{
		"message": "发布成功",
		"post":    post,
	})

} //

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

	c.JSON(200, gin.H{
		"posts": posts,
	})
}
func GetPostDetail(c *gin.Context) {

	var post model.Post

	id := c.Param("id")

	result := database.DB.First(&post, id)

	if result.Error != nil {

		c.JSON(http.StatusNotFound, gin.H{
			"message": "帖子不存在",
		})

		return
	}

	// 浏览量+1
	post.ViewCount++

	database.DB.Save(&post)

	c.JSON(http.StatusOK, gin.H{

		"message": "获取成功",

		"post": post,
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

	c.JSON(http.StatusOK, gin.H{

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
		Title string `json:"title"`

		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {

		c.JSON(http.StatusBadRequest, gin.H{
			"message": "参数错误",
		})

		return

	}

	// 更新

	post.Title = updateData.Title

	post.Content = updateData.Content

	database.DB.Save(&post)

	c.JSON(http.StatusOK, gin.H{

		"message": "修改成功",

		"post": post,
	})

}
