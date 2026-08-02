package controller

import (
	"net/http"

	"forum-backend/database"
	"forum-backend/model"
	"forum-backend/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// 注册
func Register(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	// 接收参数
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(
			c,
			http.StatusBadRequest,
			"参数错误",
		)
		return
	}
	// 检查用户名是否存在
	var count int64
	database.DB.Model(&model.User{}).
		Where(
			"username = ?",
			req.Username,
		).
		Count(&count)
	if count > 0 {
		utils.Error(
			c,
			http.StatusBadRequest,
			"用户名已存在",
		)
		return
	}
	// 密码加密
	hashPassword, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		utils.Error(
			c,
			http.StatusInternalServerError,
			"密码加密失败",
		)
		return
	}
	user := model.User{
		Username: req.Username,
		Password: string(hashPassword),
	}
	// 保存用户
	result := database.DB.Create(&user)
	if result.Error != nil {
		utils.Error(
			c,
			http.StatusInternalServerError,
			"注册失败",
		)
		return
	}
	utils.Success(
		c,
		gin.H{
			"message": "注册成功",
		},
	)
}

// 登录
func Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(
			c,
			http.StatusBadRequest,
			"参数错误",
		)
		return
	}
	var user model.User
	result := database.DB.Where(
		"username = ?",
		req.Username,
	).
		First(&user)
	if result.Error != nil {
		utils.Error(
			c,
			http.StatusBadRequest,
			"用户不存在",
		)
		return
	}
	// 验证密码
	err := bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(req.Password),
	)
	if err != nil {
		utils.Error(
			c,
			http.StatusBadRequest,
			"密码错误",
		)
		return
	}
	// 生成JWT
	token, err := utils.GenerateToken(
		user.ID,
		user.Username,
	)
	if err != nil {
		utils.Error(
			c,
			http.StatusInternalServerError,
			"token生成失败",
		)
		return
	}
	utils.Success(
		c,
		gin.H{
			"token": token,
		},
	)
}
