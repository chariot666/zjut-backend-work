package model

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=1,max=32"`
	Name     string `json:"name" binding:"required,min=1,max=32"`
	Password string `json:"password" binding:"required,min=8,max=16"`
	Role     string `json:"role" binding:"required,oneof=student admin"`
}
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}
