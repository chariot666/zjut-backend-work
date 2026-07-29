package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)


var SecretKey = []byte("forum-secret-key")


func GenerateToken(userID uint, username string)(string,error){

	claims := jwt.MapClaims{
		"user_id": userID,
		"username": username,
		"exp": time.Now().Add(
			time.Hour * 24,
		).Unix(),
	}


	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)


	return token.SignedString(SecretKey)
}