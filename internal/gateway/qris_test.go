package gateway

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func serve(t *testing.T, body string, status int) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-PARTNER-ID") != "p1" || r.Header.Get("X-TIMESTAMP") == "" {
			t.Errorf("missing SNAP headers")
		}
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, "p1")
}

func TestGenerateParsesQR(t *testing.T) {
	c := serve(t, `{"responseCode":"2004700","responseMessage":"Successful","referenceNo":"PG-1","qrContent":"0002"}`, 200)
	res, call, err := c.Generate("1", "TRX-1", "10000.00")
	if err != nil || res.ReferenceNo != "PG-1" || res.QRContent != "0002" || call.Status != 200 || call.Res == nil {
		t.Fatalf("got %+v %+v %v", res, call, err)
	}
}

func TestPGErrorBecomesErrGateway(t *testing.T) {
	c := serve(t, `{"responseCode":"4044700","responseMessage":"nope"}`, 404)
	if _, _, err := c.Generate("1", "TRX-1", "1.00"); !errors.Is(err, ErrGateway) {
		t.Fatalf("got %v, want ErrGateway", err)
	}
}

func TestSignature(t *testing.T) {
	body := []byte(`{"a":1}`)
	if !ValidSignature("s", body, Sign("s", body)) || ValidSignature("s", body, "bad") || ValidSignature("", body, Sign("", body)) {
		t.Fatal("signature check wrong")
	}
}
