package merchant

import (
	"context"
	"strings"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, request StoreRequest) (*Merchant, error) {
	m := &Merchant{
		Name:        strings.TrimSpace(request.Name),
		Phone:       request.Phone,
		Address:     request.Address,
		Description: request.Description,
	}
	if err := s.repository.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Update mengganti hanya field request yang tidak kosong.
func (s *Service) Update(ctx context.Context, id int64, request UpdateRequest) (*Merchant, error) {
	m, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if name := strings.TrimSpace(request.Name); name != "" {
		m.Name = name
	}
	if request.Phone != "" {
		m.Phone = request.Phone
	}
	if request.Address != "" {
		m.Address = request.Address
	}
	if request.Description != "" {
		m.Description = request.Description
	}
	if err := s.repository.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Service) Get(ctx context.Context, id int64) (*Merchant, error) {
	return s.repository.FindByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repository.Delete(ctx, id)
}
