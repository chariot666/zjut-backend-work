package controller

import (
	"forum-backend/database"
	"forum-backend/model"
	"forum-backend/utils"
	"net/http"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func Register(c *gin.Context){

	c.JSON(http.StatusOK,gin.H{
		"message":"注册成功",
	})

}



func Login(c *gin.Context) {

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := c.ShouldBindJSON(&req)

	if err != nil {

		c.JSON(400, gin.H{
			"message":"参数错误",
		})

		return
	}


	var user model.User


	result := database.DB.Where(
		"username = ?",
		req.Username,
	).First(&user)


	if result.Error != nil {

		c.JSON(400, gin.H{
			"message":"用户不存在",
		})

		return
	}



	err = bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(req.Password),
	)


	if err != nil {

		c.JSON(400, gin.H{
			"message":"密码错误",
		})

		return
	}



	token, err := utils.GenerateToken(
		user.ID,
		user.Username,
	)


	if err != nil {

		c.JSON(500, gin.H{
			"message":"token生成失败",
		})

		return
	}



	c.JSON(200, gin.H{

		"message":"登录成功",

		"token":token,

	})

}