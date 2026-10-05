package transaction

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
)

var (
	ErrNotFound       = errors.New("transaction not found")
	ErrReferenceInUse = errors.New("reference number already in use")
	ErrInvalidAmount  = errors.New("amount must be a number greater than 0")
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, merchantID int64, req CreateRequest) (*Transaction, error) {
	if v, err := strconv.ParseFloat(req.Amount, 64); err != nil || v <= 0 {
		return nil, ErrInvalidAmount
	}

	return s.repository.Create(ctx, merchantID, fmt.Sprintf("TRX-%d", time.Now().UnixNano()), req)
}

func (s *Service) Get(ctx context.Context, merchantID, id int64) (*Transaction, error) {
	return s.repository.FindByID(ctx, merchantID, id)
}

func (s *Service) List(ctx context.Context, merchantID int64, status string, limit, offset int) ([]Transaction, error) {
	return s.repository.List(ctx, merchantID, status, limit, offset)
}
