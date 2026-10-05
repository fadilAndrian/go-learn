package transaction

import (
	"encoding/json"
	"time"
)

type Transaction struct {
	ID             int64           `json:"id"`
	MerchantID     int64           `json:"merchant_id"`
	ReferenceNo    string          `json:"reference_no"`
	ProviderRef    string          `json:"provider_ref"`
	Amount         string          `json:"amount"` // string: NUMERIC(15,2), hindari float
	Status         string          `json:"status"`
	PaymentMethod  string          `json:"payment_method"`
	Description    string          `json:"description"`
	AdditionalInfo json.RawMessage `json:"additional_info"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type CreateRequest struct {
	Amount         string          `json:"amount"`
	PaymentMethod  string          `json:"payment_method"`
	Description    string          `json:"description"`
	AdditionalInfo json.RawMessage `json:"additional_info"`
}
