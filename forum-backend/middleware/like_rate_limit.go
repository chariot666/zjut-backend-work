package middleware

import (
	"fmt"
	"net/http"
	"time"

	"forum-backend/database"
	"forum-backend/utils"

	"github.com/gin-gonic/gin"
)

func LikeRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if database.RDB == nil {
			c.Next()
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			utils.Error(c, http.StatusUnauthorized, "未认证")
			c.Abort()
			return
		}

		key := fmt.Sprintf("rate_limit:like:user:%v", userID)
		ok, err := database.RDB.SetNX(database.Ctx, key, 1, 3*time.Second).Result()
		if err != nil {
			c.Next()
			return
		}
		if !ok {
			c.Header("Retry-After", "3")
			utils.Error(c, http.StatusTooManyRequests, "点赞太频繁，请3秒后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}
