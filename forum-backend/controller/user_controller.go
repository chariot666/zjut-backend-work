package controller

import (
	"net/http"
	"regexp"

	"forum-backend/database"
	"forum-backend/model"
	"forum-backend/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// 注册
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

	var count int64
	database.DB.Model(&model.User{}).
		Where(
			"username = ?",
			req.Username,
		).
		Count(&count)
	if count > 0 {
		utils.Error(c, http.StatusConflict, "用户名已存在")
		return
	}

	hashPassword, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "密码加密失败")
		return
	}
	user := model.User{
		Username: req.Username,
		Name:     req.Name,
		Password: string(hashPassword),
		Role:     req.Role,
	}
	result := database.DB.Create(&user)
	if result.Error != nil {
		utils.Error(c, http.StatusInternalServerError, "注册失败")
		return
	}
	utils.Created(
		c,
		user.ToResponse(),
	)
}

// 登录
func Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	var user model.User
	result := database.DB.Where(
		"username = ?",
		req.Username,
	).
		First(&user)
	if result.Error != nil {
		utils.Error(c, http.StatusUnauthorized, "用户名或密码错误")
		return
	}
	err := bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(req.Password),
	)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	token, err := utils.GenerateToken(
		user.ID,
		user.Username,
		user.Role,
	)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "token生成失败")
		return
	}
	utils.Success(
		c,
		gin.H{
			"access_token": token,
			"token_type":   "Bearer",
			"expires_in":   utils.TokenExpireSeconds,
			"user":         user.ToResponse(),
		},
	)
}
