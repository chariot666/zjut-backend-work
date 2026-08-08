package repository

import (
	"forum-backend/database"
	"forum-backend/model"
)

func FindUserByUsername(username string) (*model.User, error) {
	var user model.User
	err := database.DB.
		Where("username=?", username).
		First(&user).Error
	return &user, err
}
func CreateUser(user *model.User) error {
	return database.DB.Create(user).Error
}
