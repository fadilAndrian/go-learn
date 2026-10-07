package merchant

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, m *Merchant) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO merchants (name, phone, address, description)
		VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, ''))
		RETURNING id`,
		m.Name, m.Phone, m.Address, m.Description,
	).Scan(&m.ID)
}

func (r *Repository) FindByID(ctx context.Context, id int64) (*Merchant, error) {
	m := &Merchant{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, COALESCE(phone, ''), COALESCE(address, ''), COALESCE(description, '')
		FROM merchants WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&m.ID, &m.Name, &m.Phone, &m.Address, &m.Description)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return m, err
}

func (r *Repository) Update(ctx context.Context, m *Merchant) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE merchants
		SET name = $1,
			phone = NULLIF($2, ''),
			address = NULLIF($3, ''),
			description = NULLIF($4, ''),
			updated_at = now()
		WHERE id = $5 AND deleted_at IS NULL`,
		m.Name, m.Phone, m.Address, m.Description, m.ID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE merchants SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}
