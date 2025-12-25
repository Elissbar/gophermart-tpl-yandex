package model

type User struct {
	ID       string `json:"-"`
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Order struct {
	ID         string  `json:"-"`
	OrderNum   string  `json:"order,omitempty"`
	Number     string  `json:"number,omitempty"`
	Status     string  `json:"status"`
	Accrual    float64 `json:"accrual"`
	UploadedAt string  `json:"uploaded_at"`
}

type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type Withdraw struct {
	Order       string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}
