package transaction

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/fadilAndrian/go-learn/internal/apilog"
	"github.com/fadilAndrian/go-learn/internal/gateway"
)

const (
	StatusPending  = "pending"
	StatusPaid     = "paid"
	StatusFailed   = "failed"
	StatusRefunded = "refunded"
)

var (
	ErrNotFound       = errors.New("transaction not found")
	ErrReferenceInUse = errors.New("reference number already in use")
	ErrInvalidAmount  = errors.New("amount must be a number greater than 0")
	ErrNotRefundable  = errors.New("only paid transactions can be refunded")
)

// DefaultTTL = batas kadaluwarsa kalau PG tidak mengirim expired_at.
const DefaultTTL = time.Hour

type Service struct {
	repository *Repository
	gateway    *gateway.Client
	logs       *apilog.Service
}

func NewService(repository *Repository, gw *gateway.Client, logs *apilog.Service) *Service {
	return &Service{repository: repository, gateway: gw, logs: logs}
}

// statusFromSNAP memetakan latestTransactionStatus SNAP: 00 sukses, 05 batal, 06 gagal, selain itu masih pending.
func statusFromSNAP(code string) string {
	switch code {
	case "00":
		return StatusPaid
	case "05", "06":
		return StatusFailed
	}
	return StatusPending
}

func (s *Service) record(ctx context.Context, id int64, typ string, c gateway.Call) {
	s.logs.Record(ctx, apilog.Log{
		TransactionID: id, Type: typ, Method: "POST", URL: c.URL,
		RequestBody: c.Req, ResponseBody: c.Res, StatusCode: c.Status, DurationMs: int(c.Dur.Milliseconds()),
	})
}

// CreateQRIS membuat transaksi lalu minta QR ke PG. PG gagal → transaksi ditandai failed, error dikembalikan.
func (s *Service) CreateQRIS(ctx context.Context, merchantID int64, req CreateRequest) (*Transaction, error) {
	req.PaymentMethod = "qris"
	t, err := s.Create(ctx, merchantID, req)
	if err != nil {
		return nil, err
	}

	res, call, err := s.gateway.Generate(strconv.FormatInt(merchantID, 10), t.ReferenceNo, t.Amount)
	s.record(ctx, t.ID, apilog.OutboundCreate, call)
	if err != nil {
		_, _ = s.repository.Update(ctx, t.ID, "", StatusFailed, "", nil, nil)
		t.Status = StatusFailed
		return t, err // t dikembalikan agar handler bisa mencatat log inbound dengan id-nya
	}

	var expiredAt *time.Time
	if e, err := time.Parse(time.RFC3339, res.ExpiredTime); err == nil {
		expiredAt = &e
	}
	if _, err := s.repository.Update(ctx, t.ID, "", "", res.ReferenceNo, expiredAt, map[string]any{"qr_content": res.QRContent}); err != nil {
		return nil, err
	}
	return s.repository.FindByID(ctx, merchantID, t.ID)
}

// Check tanya status ke PG; hanya transaksi pending yang berubah (paid/refunded tidak ditimpa).
func (s *Service) Check(ctx context.Context, merchantID, id int64) (*Transaction, error) {
	t, err := s.repository.FindByID(ctx, merchantID, id)
	if err != nil || t.ProviderRef == "" {
		return t, err // ProviderRef kosong = PG tak pernah menerima transaksi ini
	}

	res, call, err := s.gateway.Query(strconv.FormatInt(merchantID, 10), t.ProviderRef, t.ReferenceNo)
	s.record(ctx, t.ID, apilog.OutboundCheck, call)
	if err != nil {
		return nil, err
	}

	st := statusFromSNAP(res.LatestTransactionStatus)
	if st == StatusPending && t.expired(time.Now()) {
		st = StatusFailed // PG masih pending tapi QR sudah kadaluwarsa
	}
	if st != StatusPending {
		if _, err := s.repository.Update(ctx, t.ID, StatusPending, st, "", nil, nil); err != nil {
			return nil, err
		}
	}
	return s.repository.FindByID(ctx, merchantID, id)
}

// Refund refund penuh transaksi paid. partnerRefundNo deterministik → retry aman di sisi PG.
func (s *Service) Refund(ctx context.Context, merchantID, id int64, reason string) (*Transaction, error) {
	t, err := s.repository.FindByID(ctx, merchantID, id)
	if err != nil {
		return nil, err
	}
	if t.Status != StatusPaid {
		return nil, ErrNotRefundable
	}

	res, call, err := s.gateway.Refund(strconv.FormatInt(merchantID, 10), t.ProviderRef, t.ReferenceNo, "RFD-"+t.ReferenceNo, t.Amount, reason)
	s.record(ctx, t.ID, apilog.OutboundRefund, call)
	if err != nil {
		return nil, err
	}

	patch := map[string]any{"refund_no": res.RefundNo, "refunded_at": time.Now().UTC().Format(time.RFC3339)}
	if _, err := s.repository.Update(ctx, t.ID, StatusPaid, StatusRefunded, "", nil, patch); err != nil {
		return nil, err
	}
	return s.repository.FindByID(ctx, merchantID, id)
}

// Notify memproses webhook PG, return id transaksi (untuk log). Idempotent: hanya pending yang berubah.
func (s *Service) Notify(ctx context.Context, n gateway.NotifyReq) (int64, error) {
	t, err := s.repository.FindByReference(ctx, n.OriginalPartnerReferenceNo)
	if err != nil {
		return 0, err
	}
	if st := statusFromSNAP(n.LatestTransactionStatus); st != StatusPending {
		_, err = s.repository.Update(ctx, t.ID, StatusPending, st, "", nil, nil)
	}
	return t.ID, err
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
