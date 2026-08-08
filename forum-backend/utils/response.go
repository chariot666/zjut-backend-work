package utils

import "github.com/gin-gonic/gin"

func Success(c *gin.Context, data interface{}) {
	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "success",
		"data": data,
	})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(201, gin.H{
		"code": 0,
		"msg":  "success",
		"data": data,
	})
}

func Error(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{
		"code": code,
		"msg":  message,
		"data": nil,
	})
}
