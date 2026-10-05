package transaction

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

const columns = `id, merchant_id, reference_no, COALESCE(provider_ref, ''), amount::text, status,
	COALESCE(payment_method, ''), COALESCE(description, ''), additional_info, created_at, updated_at`

type scanner interface{ Scan(...any) error }

func scan(row scanner, t *Transaction) error {
	return row.Scan(&t.ID, &t.MerchantID, &t.ReferenceNo, &t.ProviderRef, &t.Amount, &t.Status,
		&t.PaymentMethod, &t.Description, &t.AdditionalInfo, &t.CreatedAt, &t.UpdatedAt)
}

func (r *Repository) Create(ctx context.Context, merchantID int64, referenceNo string, req CreateRequest) (*Transaction, error) {
	var info any
	if len(req.AdditionalInfo) > 0 {
		info = []byte(req.AdditionalInfo)
	}

	t := &Transaction{}
	err := scan(r.db.QueryRow(ctx,
		`INSERT INTO transactions (merchant_id, reference_no, amount, payment_method, description, additional_info)
		 VALUES ($1, $2, $3::numeric, NULLIF($4, ''), NULLIF($5, ''), $6)
		 RETURNING `+columns,
		merchantID, referenceNo, req.Amount, req.PaymentMethod, req.Description, info), t)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return nil, ErrReferenceInUse
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *Repository) FindByID(ctx context.Context, merchantID, id int64) (*Transaction, error) {
	t := &Transaction{}
	err := scan(r.db.QueryRow(ctx,
		`SELECT `+columns+` FROM transactions WHERE id = $1 AND merchant_id = $2`, id, merchantID), t)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

// List status kosong = semua status.
func (r *Repository) List(ctx context.Context, merchantID int64, status string, limit, offset int) ([]Transaction, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+columns+` FROM transactions
		 WHERE merchant_id = $1 AND ($2 = '' OR status = $2)
		 ORDER BY id DESC LIMIT $3 OFFSET $4`, merchantID, status, limit, offset)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []Transaction{}

	for rows.Next() {
		var t Transaction
		if err := scan(rows, &t); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}
