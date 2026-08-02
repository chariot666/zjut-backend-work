package controller

import (
	"forum-backend/database"
	"forum-backend/model"
	"forum-backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 创建评论
func CreateComment(c *gin.Context) {
	var comment model.Comment
	if err := c.ShouldBindJSON(&comment); err != nil {
		utils.Error(
			c,
			http.StatusBadRequest,
			"参数错误",
		)
		return
	}
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(
			c,
			http.StatusUnauthorized,
			"未登录",
		)
		return
	}
	comment.UserID = userID.(uint)
	result := database.DB.Create(&comment)
	if result.Error != nil {
		utils.Error(
			c,
			http.StatusInternalServerError,
			"评论失败",
		)
		return
	}
	utils.Success(
		c,
		gin.H{
			"comment": comment,
		},
	)
}

// 获取帖子评论
func GetCommentList(c *gin.Context) {
	var comments []model.Comment
	postID := c.Param("post_id")
	result := database.DB.
		Where(
			"post_id = ?",
			postID,
		).
		Find(&comments)
	if result.Error != nil {
		utils.Error(
			c,
			http.StatusInternalServerError,
			"查询失败",
		)
		return
	}
	utils.Success(
		c,
		gin.H{
			"comments": comments,
		},
	)
}

// 删除评论
func DeleteComment(c *gin.Context) {
	id := c.Param("id")
	var comment model.Comment
	result := database.DB.First(
		&comment,
		id,
	)
	if result.Error != nil {
		utils.Error(
			c,
			http.StatusNotFound,
			"评论不存在",
		)
		return
	}
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(
			c,
			http.StatusUnauthorized,
			"未登录",
		)
		return
	}
	// 判断是否本人评论
	if comment.UserID != userID.(uint) {
		utils.Error(
			c,
			http.StatusForbidden,
			"无权限删除",
		)
		return
	}
	database.DB.Delete(&comment)
	utils.Success(
		c,
		gin.H{
			"message": "删除成功",
		},
	)
}
