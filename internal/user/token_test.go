package user

import (
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestToken(t *testing.T) {
	signed, err := GenerateToken(42, "secret")
	if err != nil {
		t.Fatal(err)
	}

	id, err := ParseToken(signed, "secret")
	if err != nil || id != 42 {
		t.Fatalf("ParseToken = %d, %v; want 42, nil", id, err)
	}

	if _, err := ParseToken(signed, "wrong"); err == nil {
		t.Fatal("token accepted with wrong secret")
	}

	expired, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   strconv.Itoa(42),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
	}).SignedString([]byte("secret"))
	if _, err := ParseToken(expired, "secret"); err == nil {
		t.Fatal("expired token accepted")
	}
}
