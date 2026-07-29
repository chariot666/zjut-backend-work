package database

import (
	"fmt"

	"forum-backend/model"
)


func AutoMigrate(){

	err := DB.AutoMigrate(
		&model.User{},
		&model.Post{},
	)


	if err != nil {

		panic("数据库迁移失败")

	}


	fmt.Println("数据库表创建成功")

}