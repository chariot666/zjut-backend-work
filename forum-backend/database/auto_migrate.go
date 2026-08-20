package database

import (
	"fmt"
	"forum-backend/model"
)

// 自动创建并更新表
func AutoMigrate() {
	err := DB.AutoMigrate(
		&model.User{},
		&model.Post{},
		&model.Comment{},
		&model.Like{},
		&model.AgentMessage{},
		&model.AgentDraft{},
	)
	if err != nil {
		panic("数据库迁移失败")
	}
	fmt.Println("数据库表创建成功")
}
