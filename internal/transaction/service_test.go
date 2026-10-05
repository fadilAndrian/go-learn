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

func TestRunCheckerStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() { NewService(nil, nil, nil).RunChecker(ctx, time.Hour); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("RunChecker did not stop")
	}
}
