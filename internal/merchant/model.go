package merchant

// Merchant = satu baris tabel merchants (≈ Eloquent model).
// Sengaja tanpa tag json: data keluar ke client hanya lewat MerchantResource,
// jadi kolom baru di tabel tidak otomatis ikut terkirim.
type Merchant struct {
	ID          int64
	Name        string
	Phone       string
	Address     string
	Description string
}
