package model

type User struct {
	ID       string    `json:"-"`
	Login    string `json:"login"`
	Password string `json:"password"`
}
