package merchant

import (
	"errors"
	"strings"
)

// Request = bentuk input JSON + aturan validasinya (≈ FormRequest di Laravel).

type StoreRequest struct {
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	Description string `json:"description"`
}

func (r StoreRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	return nil
}

// UpdateRequest: field kosong = tidak diubah.
type UpdateRequest struct {
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	Description string `json:"description"`
}
