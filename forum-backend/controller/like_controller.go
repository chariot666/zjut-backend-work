package controller

import (
	"net/http"

	"forum-backend/service"
	"forum-backend/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func LikePost(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "未认证")
		return
	}

	postID, ok := parseIDParam(c, "post_id")
	if !ok {
		return
	}

	isLiked, err := service.ToggleLike(userID.(uint), postID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "记录不存在" || err == gorm.ErrRecordNotFound {
			status = http.StatusNotFound
		}
		utils.Error(c, status, "点赞失败")
		return
	}

	utils.Success(c, gin.H{
		"post_id":  postID,
		"is_liked": isLiked,
	})
}

func UnlikePost(c *gin.Context) {
	LikePost(c)
}

func GetLikeStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "未认证")
		return
	}

	var req struct {
		PostIDs []uint `json:"post_ids" binding:"required,min=1,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	status, err := service.GetLikeStatus(userID.(uint), req.PostIDs)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "查询失败")
		return
	}

	utils.Success(c, gin.H{"status": status})
}
