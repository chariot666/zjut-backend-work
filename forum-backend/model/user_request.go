package model

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname"`
}
type LoginRequest struct {

	Username string `json:"username" binding:"required"`

	Password string `json:"password" binding:"required"`

}