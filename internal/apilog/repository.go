package apilog

import (
	"context"
	"encoding/json"
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

func (r *Repository) Create(ctx context.Context, l Log) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO api_logs (transaction_id, type, method, url, request_body, response_body, status_code, duration_ms)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		l.TransactionID, l.Type, l.Method, l.URL, nullJSON(l.RequestBody), nullJSON(l.ResponseBody), l.StatusCode, l.DurationMs)
	return err
}

// ListByTransaction hanya mengembalikan log transaksi milik merchantID.
func (r *Repository) ListByTransaction(ctx context.Context, merchantID, transactionID int64) ([]Log, error) {
	rows, err := r.db.Query(ctx,
		`SELECT l.id, l.transaction_id, l.type, l.method, l.url,
		        l.request_body, l.response_body, COALESCE(l.status_code, 0), COALESCE(l.duration_ms, 0), l.created_at
		 FROM api_logs l JOIN transactions t ON t.id = l.transaction_id
		 WHERE l.transaction_id = $1 AND t.merchant_id = $2
		 ORDER BY l.id`, transactionID, merchantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := []Log{}
	for rows.Next() {
		var l Log
		if err := rows.Scan(&l.ID, &l.TransactionID, &l.Type, &l.Method, &l.URL,
			&l.RequestBody, &l.ResponseBody, &l.StatusCode, &l.DurationMs, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// FindByID hanya mengembalikan log milik merchantID (lewat transaksinya).
func (r *Repository) FindByID(ctx context.Context, merchantID, id int64) (*Log, error) {
	var l Log
	err := r.db.QueryRow(ctx,
		`SELECT l.id, l.transaction_id, l.type, l.method, l.url,
		        l.request_body, l.response_body, COALESCE(l.status_code, 0), COALESCE(l.duration_ms, 0), l.created_at
		 FROM api_logs l JOIN transactions t ON t.id = l.transaction_id
		 WHERE l.id = $1 AND t.merchant_id = $2`, id, merchantID,
	).Scan(&l.ID, &l.TransactionID, &l.Type, &l.Method, &l.URL,
		&l.RequestBody, &l.ResponseBody, &l.StatusCode, &l.DurationMs, &l.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func nullJSON(b json.RawMessage) any {
	if len(b) == 0 {
		return nil
	}
	return []byte(b)
}
