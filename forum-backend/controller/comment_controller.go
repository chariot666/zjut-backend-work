package controller

import (
	"errors"
	"net/http"

	"forum-backend/service"
	"forum-backend/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateComment(c *gin.Context) {
	postID, ok := parseIDParam(c, "post_id")
	if !ok {
		return
	}

	var req struct {
		Content string `json:"content" binding:"required,min=1,max=1000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "未认证")
		return
	}

	comment, err := service.CreateComment(userID.(uint), postID, req.Content)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			status = http.StatusNotFound
		case err.Error() == "禁止访问":
			status = http.StatusForbidden
		}
		utils.Error(c, status, err.Error())
		return
	}

	utils.Created(c, comment)
}

func GetCommentList(c *gin.Context) {
	postID, ok := parseIDParam(c, "post_id")
	if !ok {
		return
	}

	comments, err := service.GetCommentsByPostID(postID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		utils.Error(c, status, "查询失败")
		return
	}

	utils.Success(c, gin.H{"comments": comments})
}

func DeleteComment(c *gin.Context) {
	commentID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "未认证")
		return
	}

	if err := service.DeleteComment(commentID, userID.(uint)); err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			status = http.StatusNotFound
		case err.Error() == "禁止访问":
			status = http.StatusForbidden
		}
		utils.Error(c, status, err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "删除成功"})
}
