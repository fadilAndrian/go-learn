package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fadilAndrian/go-learn/internal/user"
	"github.com/gofiber/fiber/v3"
)

var testCfg = Config{Key: "rahasia-admin", Secret: "jwt-secret", SecureCookie: true}

func newApp() *fiber.App {
	app := fiber.New()
	h := NewHandler(testCfg)
	app.Post("/admin/login", h.Login)
	app.Post("/admin/logout", h.Logout)
	app.Get("/admin/ping", Auth(testCfg), func(c fiber.Ctx) error { return c.SendString("ok") })
	return app
}

func do(t *testing.T, app *fiber.App, req *http.Request) *http.Response {
	t.Helper()
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func login(t *testing.T, app *fiber.App, key string) *http.Response {
	req := httptest.NewRequest("POST", "/admin/login", strings.NewReader(`{"key":"`+key+`"}`))
	req.Header.Set("Content-Type", "application/json")
	return do(t, app, req)
}

func ping(t *testing.T, app *fiber.App, cookie string) int {
	req := httptest.NewRequest("GET", "/admin/ping", nil)
	if cookie != "" {
		req.Header.Set("Cookie", CookieName+"="+cookie)
	}
	return do(t, app, req).StatusCode
}

func TestLogin(t *testing.T) {
	app := newApp()

	for _, key := range []string{"", "salah"} {
		if resp := login(t, app, key); resp.StatusCode != 401 || len(resp.Cookies()) != 0 {
			t.Fatalf("key %q: status %d, cookies %v; want 401 tanpa cookie", key, resp.StatusCode, resp.Cookies())
		}
	}

	resp := login(t, app, testCfg.Key)
	if resp.StatusCode != 204 || len(resp.Cookies()) != 1 {
		t.Fatalf("login benar: status %d, cookies %v", resp.StatusCode, resp.Cookies())
	}
	c := resp.Cookies()[0]
	if c.Name != CookieName || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode || c.Path != "/admin" {
		t.Fatalf("atribut cookie salah: %+v", c)
	}

	if got := ping(t, app, c.Value); got != 200 {
		t.Fatalf("dengan cookie sesi: %d, want 200", got)
	}
}

func TestAuthRejects(t *testing.T) {
	app := newApp()
	userToken, _ := user.GenerateToken(1, testCfg.Secret)
	otherSecret, _ := newSessionToken("secret-lain", time.Now())

	cases := map[string]string{
		"tanpa cookie":               "",
		"token Bearer user":          userToken, // secret sama, tapi bukan sesi admin
		"ditandatangani secret lain": otherSecret,
		"sampah":                     "abc.def.ghi",
	}
	for name, cookie := range cases {
		if got := ping(t, app, cookie); got != 401 {
			t.Errorf("%s: %d, want 401", name, got)
		}
	}

	// Bearer di header juga tidak berlaku untuk route admin.
	req := httptest.NewRequest("GET", "/admin/ping", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	if got := do(t, app, req).StatusCode; got != 401 {
		t.Errorf("Authorization Bearer: %d, want 401", got)
	}
}

func TestLogoutClearsCookie(t *testing.T) {
	resp := do(t, newApp(), httptest.NewRequest("POST", "/admin/logout", nil))
	if resp.StatusCode != 204 || len(resp.Cookies()) != 1 {
		t.Fatalf("status %d, cookies %v", resp.StatusCode, resp.Cookies())
	}
	if c := resp.Cookies()[0]; c.Value != "" || c.Path != "/admin" || !c.Expires.Before(time.Now()) {
		t.Fatalf("cookie tidak terhapus: %+v", c)
	}
}
