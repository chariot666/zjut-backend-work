package middleware

import (
	"forum-backend/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			utils.Error(c, http.StatusUnauthorized, "未认证")
			c.Abort()
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(
			tokenString,
			func(token *jwt.Token) (interface{}, error) {
				// 检查签名算法
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return utils.SecretKey, nil
			},
		)
		if err != nil || !token.Valid {
			utils.Error(c, http.StatusUnauthorized, "token无效")
			c.Abort()
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			utils.Error(c, http.StatusUnauthorized, "token无效")
			c.Abort()
			return
		}
		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			utils.Error(c, http.StatusUnauthorized, "token无效")
			c.Abort()
			return
		}
		userID := uint(userIDFloat)
		role, _ := claims["role"].(string)
		c.Set("user_id", userID)
		c.Set("role", role)
		c.Next()
	}
}

func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != "admin" {
			utils.Error(c, http.StatusForbidden, "禁止访问")
			c.Abort()
			return
		}
		c.Next()
	}
}
