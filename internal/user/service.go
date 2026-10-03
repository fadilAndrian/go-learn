package user

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

var (
	ErrEmailInUse         = errors.New("email already in use")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type Service struct {
	repository *Repository
	jwtSecret  string
}

func NewService(repository *Repository, jwtSecret string) *Service {
	return &Service{repository: repository, jwtSecret: jwtSecret}
}

func (s *Service) GetUserByID(ctx context.Context, id int64) (*User, error) {
	return s.repository.FindByID(ctx, id)
}

func (s *Service) Login(ctx context.Context, email string, password string) (*User, string, error) {
	user, err := s.repository.FindByEmail(ctx, email)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrInvalidCredentials
	}

	if err != nil {
		return nil, "", err
	}

	if !CheckPassword(password, user.Password) {
		return nil, "", ErrInvalidCredentials
	}

	token, err := GenerateToken(user.ID, s.jwtSecret)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *Service) RegisterUser(ctx context.Context, request CreateUserRequest) (*User, error) {
	hashedPassword, err := HashPassword(request.Password)
	if err != nil {
		return nil, err
	}

	user := &User{
		Name:       request.Name,
		MerchantID: request.MerchantID,
		Email:      request.Email,
		Password:   hashedPassword,
	}

	if err := s.repository.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateUser mengganti hanya field request yang tidak kosong.
func (s *Service) UpdateUser(ctx context.Context, id int64, request UpdateUserRequest) (*User, error) {
	user, err := s.repository.FindByID(ctx, id)

	if err != nil {
		return nil, err
	}

	if request.Name != "" {
		user.Name = request.Name
	}

	if request.Email != "" {
		user.Email = request.Email
	}

	if request.Password != "" {
		hashedPassword, err := HashPassword(request.Password)

		if err != nil {
			return nil, err
		}

		user.Password = hashedPassword
	}

	if err := s.repository.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
