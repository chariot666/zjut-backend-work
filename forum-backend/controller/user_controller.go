package controller

import (
	"net/http"
	"regexp"
	"strings"

	"forum-backend/model"
	"forum-backend/service"
	"forum-backend/utils"

	"github.com/gin-gonic/gin"
)

func Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if !regexp.MustCompile(`^[0-9]+$`).MatchString(req.Username) {
		utils.Error(c, http.StatusBadRequest, "用户名只能由数字组成")
		return
	}

	user, err := service.Register(req)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "已存在") {
			status = http.StatusConflict
		}
		utils.Error(c, status, err.Error())
		return
	}

	utils.Created(c, user.ToResponse())
}

func Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	user, token, err := service.Login(req)
	if err != nil {
		status := http.StatusUnauthorized
		if strings.Contains(err.Error(), "token") {
			status = http.StatusInternalServerError
		}
		utils.Error(c, status, err.Error())
		return
	}

	utils.Success(c, gin.H{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   utils.TokenExpireSeconds,
		"user":         user.ToResponse(),
	})
}
