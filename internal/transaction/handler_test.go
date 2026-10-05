package transaction

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fadilAndrian/go-learn/internal/gateway"
	"github.com/gofiber/fiber/v3"
)

func TestStatusFromSNAP(t *testing.T) {
	for code, want := range map[string]string{"00": StatusPaid, "06": StatusFailed, "05": StatusFailed, "03": StatusPending, "": StatusPending} {
		if got := statusFromSNAP(code); got != want {
			t.Fatalf("%q: got %s, want %s", code, got, want)
		}
	}
}

// signature dicek sebelum akses service/DB, jadi handler tanpa dependency cukup.
func TestNotifyRejectsBadSignature(t *testing.T) {
	app := fiber.New()
	app.Post("/n", (&Handler{}).Notify("secret"))

	body := `{"originalPartnerReferenceNo":"TRX-1","latestTransactionStatus":"00"}`
	for _, sig := range []string{"", "bad", gateway.Sign("other", []byte(body))} {
		req := httptest.NewRequest("POST", "/n", strings.NewReader(body))
		req.Header.Set("X-SIGNATURE", sig)
		resp, err := app.Test(req)
		if err != nil || resp.StatusCode != 401 {
			t.Fatalf("sig %q: status %v err %v, want 401", sig, resp.StatusCode, err)
		}
	}
}
