package merchant

// MerchantResource = bentuk output JSON (≈ API Resource di Laravel).
type MerchantResource struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	Description string `json:"description"`
}

func NewMerchantResource(m *Merchant) MerchantResource {
	return MerchantResource{
		ID:          m.ID,
		Name:        m.Name,
		Phone:       m.Phone,
		Address:     m.Address,
		Description: m.Description,
	}
}
