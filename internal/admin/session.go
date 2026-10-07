package admin

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Sesi admin dashboard: setelah login, browser menyimpan cookie HttpOnly berisi
// JWT bertanda tangan. JavaScript dashboard tidak bisa membaca cookie ini, jadi
// token tidak bisa dicuri lewat XSS seperti token di localStorage.
//
// SEMENTARA: login memakai satu ADMIN_KEY bersama dan sesi tidak bisa dicabut
// sebelum kedaluwarsa. Nanti diganti akun admin per orang (users + roles) dan
// sesi di database. Middleware Auth dan route merchant tidak perlu berubah.

const (
	CookieName = "admin_session"
	cookiePath = "/admin" // cookie hanya terkirim ke /admin/*, tidak ke API lain
	sessionTTL = 8 * time.Hour

	// audience membedakan token sesi admin dari token Bearer user,
	// walaupun keduanya ditandatangani JWT_SECRET yang sama.
	audience = "admin-dashboard"
)

type Config struct {
	Key          string // ADMIN_KEY: kunci login dashboard
	Secret       string // JWT_SECRET: penanda tangan cookie sesi
	SecureCookie bool   // false hanya untuk lokal tanpa HTTPS
}

// validKey membandingkan dengan waktu konstan supaya panjang/isi kunci
// tidak bisa ditebak dari lama respons.
func (cfg Config) validKey(key string) bool {
	a, b := sha256.Sum256([]byte(key)), sha256.Sum256([]byte(cfg.Key))
	return key != "" && subtle.ConstantTimeCompare(a[:], b[:]) == 1
}

func newSessionToken(secret string, now time.Time) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   "admin",
		Audience:  jwt.ClaimStrings{audience},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(sessionTTL)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func parseSessionToken(token, secret string) error {
	if token == "" {
		return errors.New("no session")
	}
	_, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{}, func(*jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired(), jwt.WithAudience(audience))
	return err
}
