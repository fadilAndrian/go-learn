package apilog

import (
	"context"
	"errors"
	"log"
)

var ErrNotFound = errors.New("log not found")

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

// Record menyimpan log; gagal simpan hanya di-log, tidak menggagalkan request.
// Body harus JSON valid atau nil.
func (s *Service) Record(ctx context.Context, l Log) {
	if err := s.repository.Create(ctx, l); err != nil {
		log.Printf("apilog: record failed: %v", err)
	}
}

func (s *Service) ListByTransaction(ctx context.Context, merchantID, transactionID int64) ([]Log, error) {
	return s.repository.ListByTransaction(ctx, merchantID, transactionID)
}

func (s *Service) GetByID(ctx context.Context, merchantID, id int64) (*Log, error) {
	return s.repository.FindByID(ctx, merchantID, id)
}
