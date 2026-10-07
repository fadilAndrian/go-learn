package admin

import (
	"time"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	cfg Config
}

func NewHandler(cfg Config) *Handler {
	return &Handler{cfg: cfg}
}

// Login: POST /admin/login {"key": "..."} → set cookie sesi, balas 204.
func (h *Handler) Login(c fiber.Ctx) error {
	var request LoginRequest
	if err := c.Bind().Body(&request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if !h.cfg.validKey(request.Key) {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid admin key")
	}

	token, err := newSessionToken(h.cfg.Secret, time.Now())
	if err != nil {
		return err
	}
	c.Cookie(h.cookie(token, time.Now().Add(sessionTTL)))
	return c.SendStatus(fiber.StatusNoContent)
}

// Logout: POST /admin/logout → hapus cookie sesi di browser.
func (h *Handler) Logout(c fiber.Ctx) error {
	c.Cookie(h.cookie("", time.Unix(0, 0)))
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) cookie(value string, expires time.Time) *fiber.Cookie {
	return &fiber.Cookie{
		Name:     CookieName,
		Value:    value,
		Path:     cookiePath,
		Expires:  expires,
		HTTPOnly: true,
		Secure:   h.cfg.SecureCookie,
		// Strict: browser tidak mengirim cookie dari situs lain → mencegah CSRF.
		SameSite: fiber.CookieSameSiteStrictMode,
	}
}
