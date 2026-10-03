package user

type User struct {
	ID         int64  `json:"id"`
	MerchantID int64  `json:"merchant_id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Password   string `json:"-"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateUserRequest struct {
	Name       string `json:"name"`
	MerchantID int64  `json:"merchant_id"`
	Email      string `json:"email"`
	Password   string `json:"password"`
}

type UpdateUserRequest struct {
	Name       string `json:"name"`
	MerchantID int64  `json:"merchant_id"`
	Email      string `json:"email"`
	Password   string `json:"password"`
}
