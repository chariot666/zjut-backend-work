package service

import (
	"errors"

	"forum-backend/model"
	"forum-backend/repository"
	"forum-backend/utils"

	"golang.org/x/crypto/bcrypt"
)

func Register(req model.RegisterRequest) error {

	_, err := repository.FindUserByUsername(req.Username)

	if err == nil {
		return errors.New("用户名已存在")
	}

	password, _ := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)

	user := model.User{
		Username: req.Username,
		Password: string(password),
		Name:     req.Name,
		Role:     req.Role,
	}

	return repository.CreateUser(&user)

}
func Login(req model.LoginRequest) (*model.User, string, error) {

	user, err := repository.FindUserByUsername(req.Username)

	if err != nil {
		return nil, "", errors.New("用户不存在")
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(req.Password),
	)

	if err != nil {
		return nil, "", errors.New("密码错误")
	}

	token, err := utils.GenerateToken(
		user.ID,
		user.Username,
		user.Role,
	)

	return user, token, err
}
