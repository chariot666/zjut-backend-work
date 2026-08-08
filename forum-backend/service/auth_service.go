package service

import (
	"errors"

	"forum-backend/model"
	"forum-backend/repository"
	"forum-backend/utils"

	"golang.org/x/crypto/bcrypt"
)

func Register(req model.RegisterRequest) (*model.User, error) {
	user, err := repository.FindUserByUsername(req.Username)
	if err == nil && user != nil {
		return nil, errors.New("用户名已存在")
	}

	password, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("密码加密失败")
	}

	newUser := &model.User{
		Username: req.Username,
		Password: string(password),
		Name:     req.Name,
		Role:     req.Role,
	}
	if err := repository.CreateUser(newUser); err != nil {
		return nil, err
	}
	return newUser, nil
}

func Login(req model.LoginRequest) (*model.User, string, error) {
	user, err := repository.FindUserByUsername(req.Username)
	if err != nil {
		return nil, "", errors.New("用户名或密码错误")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, "", errors.New("用户名或密码错误")
	}

	token, err := utils.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, "", errors.New("token生成失败")
	}
	return user, token, nil
}
