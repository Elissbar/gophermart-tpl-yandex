package model

type User struct {
	ID       string `json:"-"`
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Order struct {
	Number     string `json:"number"`
	Status     string `json:"status"`
	Accrual    string `json:"accrual"`
	UploadedAt string `json:"uploaded_at"`
}

type Balance struct {
	Current   string `json:"current"`
	Withdrawn string `json:"withdrawn"`
}
