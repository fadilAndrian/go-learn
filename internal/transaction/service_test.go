package transaction

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCreateRejectsBadAmount(t *testing.T) {
	s := NewService(nil, nil, nil) // validasi jalan sebelum akses DB
	for _, amount := range []string{"", "abc", "0", "-5"} {
		if _, err := s.Create(context.Background(), 1, CreateRequest{Amount: amount}); !errors.Is(err, ErrInvalidAmount) {
			t.Fatalf("amount %q: got %v, want ErrInvalidAmount", amount, err)
		}
	}
}

func TestExpired(t *testing.T) {
	now := time.Now()
	past, future := now.Add(-time.Minute), now.Add(time.Minute)
	cases := []struct {
		name string
		t    Transaction
		want bool
	}{
		{"expired_at lewat", Transaction{ExpiredAt: &past, CreatedAt: now}, true},
		{"expired_at belum", Transaction{ExpiredAt: &future, CreatedAt: now.Add(-2 * time.Hour)}, false},
		{"null, lewat default 1 jam", Transaction{CreatedAt: now.Add(-2 * time.Hour)}, true},
		{"null, belum 1 jam", Transaction{CreatedAt: now.Add(-time.Minute)}, false},
	}
	for _, c := range cases {
		if got := c.t.expired(now); got != c.want {
			t.Fatalf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
