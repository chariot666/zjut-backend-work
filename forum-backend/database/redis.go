package database

import (
	"context"
	"fmt"
	"forum-backend/config"
	"log"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client
var Ctx = context.Background()

func InitRedis() {
	redisConfig := config.AppConfig.Redis
	RDB = redis.NewClient(
		&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", redisConfig.Host, redisConfig.Port),
			Password: redisConfig.Password,
			DB:       redisConfig.DB,
		},
	)
	_, err := RDB.Ping(Ctx).Result()
	if err != nil {
		log.Printf("Redis连接失败，服务将不使用缓存运行: %v", err)
		RDB = nil
		return
	}
	fmt.Println("Redis连接成功")
}
