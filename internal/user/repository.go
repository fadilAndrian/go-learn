package user

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByID(ctx context.Context, id int64) (*User, error) {
	user := &User{}

	err := r.db.QueryRow(ctx,
		`SELECT id, name, email, password FROM users WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Password)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}

	err := r.db.QueryRow(ctx,
		`SELECT id, name, email, password FROM users WHERE email = $1 AND deleted_at IS NULL`, email,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Password)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *Repository) Create(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (merchant_id, name, email, password)
		VALUES (NULLIF($1, 0), $2, $3, $4)
		RETURNING id
	`
	err := r.db.QueryRow(ctx, query, user.MerchantID, user.Name, user.Email, user.Password).Scan(&user.ID)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrEmailInUse
	}

	return err
}

func (r *Repository) Update(ctx context.Context, user *User) error {
	query := `
		UPDATE users
		SET name = $1,
			email = $2,
			password = $3,
			updated_at = now()
		WHERE id = $4 AND deleted_at IS NULL
	`

	_, err := r.db.Exec(ctx, query, user.Name, user.Email, user.Password, user.ID)

	return err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	query := `
		UPDATE users
		SET deleted_at = now()
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err := r.db.Exec(ctx, query, id)

	return err
}
