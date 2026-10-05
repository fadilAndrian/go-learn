package transaction

import (
	"context"
	"errors"
	"testing"
)

func TestCreateRejectsBadAmount(t *testing.T) {
	s := NewService(nil) // validasi jalan sebelum akses DB
	for _, amount := range []string{"", "abc", "0", "-5"} {
		if _, err := s.Create(context.Background(), 1, CreateRequest{Amount: amount}); !errors.Is(err, ErrInvalidAmount) {
			t.Fatalf("amount %q: got %v, want ErrInvalidAmount", amount, err)
		}
	}
}
