package merchant

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// Test ini hanya mengecek input yang ditolak SEBELUM menyentuh database,
// jadi service boleh nil. Alur CRUD sampai ke database belum ada test-nya.
func TestHandlerRejectsBadInput(t *testing.T) {
	h := NewHandler(nil)
	app := fiber.New()
	app.Post("/admin/merchants", h.Store)
	app.Get("/admin/merchants/:id", h.Show)
	app.Put("/admin/merchants/:id", h.Update)
	app.Delete("/admin/merchants/:id", h.Destroy)

	cases := []struct{ method, path, body string }{
		{"POST", "/admin/merchants", `{"name":""}`},
		{"POST", "/admin/merchants", `{"name":"   "}`},
		{"POST", "/admin/merchants", `{bukan json`},
		{"GET", "/admin/merchants/abc", ""},
		{"GET", "/admin/merchants/0", ""},
		{"PUT", "/admin/merchants/-1", `{}`},
		{"DELETE", "/admin/merchants/1.5", ""},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("%s %s %s: %d, want 400", tc.method, tc.path, tc.body, resp.StatusCode)
		}
	}
}

func TestToHTTP(t *testing.T) {
	var fe *fiber.Error
	if err := toHTTP(ErrNotFound); !errors.As(err, &fe) || fe.Code != 404 {
		t.Fatalf("ErrNotFound → %v, want 404", err)
	}
}
