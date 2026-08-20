package middleware

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"forum-backend/utils"

	"github.com/gin-gonic/gin"
)

func InitLogger(logPath string) (*os.File, error) {
	if err := os.MkdirAll("logs", 0755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	log.SetOutput(io.MultiWriter(os.Stdout, file))
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	return file, nil
}

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		if query != "" {
			path = fmt.Sprintf("%s?%s", path, query)
		}

		log.Printf(
			"[request] method=%s path=%s status=%d latency=%s client_ip=%s",
			c.Request.Method,
			path,
			c.Writer.Status(),
			time.Since(start),
			c.ClientIP(),
		)
	}
}

// 记录panic，防止服务崩溃
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[panic] error=%v path=%s method=%s\n%s",
					err,
					c.Request.URL.Path,
					c.Request.Method,
					debug.Stack(),
				)
				utils.Error(c, http.StatusInternalServerError, "服务器内部错误")
				c.Abort()
			}
		}()

		c.Next()
	}
}
