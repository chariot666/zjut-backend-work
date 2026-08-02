package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client
var Ctx = context.Background()

func InitRedis() {
	RDB = redis.NewClient(
		&redis.Options{
			Addr:     "127.0.0.1:6379",
			Password: "",
			DB:       0,
		},
	)
	_, err := RDB.Ping(Ctx).Result()
	if err != nil {
		panic("Redis连接失败")
	}
	fmt.Println("Redis连接成功")
}
