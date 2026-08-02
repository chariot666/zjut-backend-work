package controller

import (
	"forum-backend/database"
	"forum-backend/model"
	"forum-backend/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 点赞
func LikePost(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(
			c,
			http.StatusUnauthorized,
			"未登录",
		)
		return
	}
	postIDStr := c.Param("id")
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		utils.Error(
			c,
			http.StatusBadRequest,
			"帖子ID错误",
		)
		return
	}
	var like model.Like
	// 查询是否已经点赞
	result := database.DB.
		Where(
			"user_id=? AND post_id=?",
			userID,
			postID,
		).
		First(&like)
	if result.Error == nil {
		utils.Error(
			c,
			http.StatusBadRequest,
			"已经点赞",
		)
		return
	}
	// 创建点赞记录
	newLike := model.Like{
		UserID: userID.(uint),
		PostID: uint(postID),
	}
	result = database.DB.Create(&newLike)
	if result.Error != nil {
		utils.Error(
			c,
			http.StatusInternalServerError,
			"点赞失败",
		)
		return
	}
	// 帖子点赞数+1
	database.DB.Model(&model.Post{}).
		Where(
			"id=?",
			postID,
		).
		UpdateColumn(
			"like_count",
			gorm.Expr(
				"like_count + ?",
				1,
			),
		)
	utils.Success(
		c,
		gin.H{
			"message": "点赞成功",
		},
	)
}

// 取消点赞
func UnlikePost(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(
			c,
			http.StatusUnauthorized,
			"未登录",
		)
		return
	}
	postID := c.Param("id")
	var like model.Like
	result := database.DB.
		Where(
			"user_id=? AND post_id=?",
			userID,
			postID,
		).
		First(&like)
	if result.Error != nil {
		utils.Error(
			c,
			http.StatusBadRequest,
			"未点赞",
		)
		return
	}
	database.DB.Delete(&like)
	database.DB.Model(&model.Post{}).
		Where(
			"id=?",
			postID,
		).
		UpdateColumn(
			"like_count",
			gorm.Expr(
				"like_count - ?",
				1,
			),
		)
	utils.Success(
		c,
		gin.H{
			"message": "取消点赞成功",
		},
	)
}
