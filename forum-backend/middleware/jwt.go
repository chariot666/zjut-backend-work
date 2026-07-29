package middleware

import (
	"net/http"
	"strings"

	"forum-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)


func JWTAuth() gin.HandlerFunc {

	return func(c *gin.Context) {


		authHeader := c.GetHeader("Authorization")


		if authHeader == "" {

			c.JSON(http.StatusUnauthorized, gin.H{
				"message":"未登录",
			})

			c.Abort()
			return
		}



		tokenString := strings.TrimPrefix(
			authHeader,
			"Bearer ",
		)



		token, err := jwt.Parse(
			tokenString,
			func(token *jwt.Token)(interface{},error){

				// 检查签名算法
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {

					return nil, jwt.ErrSignatureInvalid
				}


				return utils.SecretKey,nil
			},
		)



		if err != nil || !token.Valid {

			c.JSON(http.StatusUnauthorized,gin.H{
				"message":"token无效",
				"error":err.Error(),
			})

			c.Abort()
			return
		}
		claims := token.Claims.(jwt.MapClaims)

		userID := uint(claims["user_id"].(float64))

		c.Set("user_id", userID)


		c.Next()

	}

}