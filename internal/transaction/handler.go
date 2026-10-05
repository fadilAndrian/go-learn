package transaction

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/fadilAndrian/go-learn/internal/apilog"
	"github.com/fadilAndrian/go-learn/internal/gateway"
	"github.com/fadilAndrian/go-learn/internal/user"
	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	service *Service
	users   *user.Service
	logs    *apilog.Service
}

func NewHandler(service *Service, users *user.Service, logs *apilog.Service) *Handler {
	return &Handler{service: service, users: users, logs: logs}
}

// merchantID mengambil merchant milik user yang login; user tanpa merchant ditolak.
func (h *Handler) merchantID(c fiber.Ctx) (int64, error) {
	u, err := h.users.GetUserByID(c.Context(), user.UserID(c))
	if err != nil {
		return 0, fiber.NewError(fiber.StatusUnauthorized, "user not found")
	}
	if u.MerchantID == 0 {
		return 0, fiber.NewError(fiber.StatusForbidden, "user has no merchant")
	}
	return u.MerchantID, nil
}

func (h *Handler) List(c fiber.Ctx) error {
	merchantID, err := h.merchantID(c)
	if err != nil {
		return err
	}

	limit := fiber.Query[int](c, "limit", 20)
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := max(fiber.Query[int](c, "offset", 0), 0)

	list, err := h.service.List(c.Context(), merchantID, c.Query("status"), limit, offset)
	if err != nil {
		return err
	}
	return c.JSON(list)
}

func (h *Handler) Get(c fiber.Ctx) error {
	merchantID, err := h.merchantID(c)
	if err != nil {
		return err
	}

	id, err := parseID(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	t, err := h.service.Get(c.Context(), merchantID, id)
	if errors.Is(err, ErrNotFound) {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	if err != nil {
		return err
	}
	return c.JSON(t)
}

func (h *Handler) Logs(c fiber.Ctx) error {
	merchantID, err := h.merchantID(c)
	if err != nil {
		return err
	}

	id, err := parseID(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	// Get dulu supaya transaksi merchant lain / tak ada → 404, bukan list kosong.
	if _, err := h.service.Get(c.Context(), merchantID, id); errors.Is(err, ErrNotFound) {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	} else if err != nil {
		return err
	}

	logs, err := h.logs.ListByTransaction(c.Context(), merchantID, id)
	if err != nil {
		return err
	}
	return c.JSON(logs)
}

func (h *Handler) LogDetail(c fiber.Ctx) error {
	merchantID, err := h.merchantID(c)
	if err != nil {
		return err
	}

	logID, err := parseID(c.Params("logId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid log id")
	}

	l, err := h.logs.GetByID(c.Context(), merchantID, logID)
	if errors.Is(err, apilog.ErrNotFound) {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	if err != nil {
		return err
	}
	return c.JSON(l)
}

// outboundErr memetakan error service outbound ke HTTP: PG bermasalah → 502.
func outboundErr(err error) error {
	switch {
	case errors.Is(err, ErrInvalidAmount):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, ErrNotRefundable):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case errors.Is(err, gateway.ErrGateway):
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	return err
}

// logInbound mencatat request client → kita (sisi inbound), berpasangan dengan log outbound ke PG.
// t boleh nil (transaksi belum terbentuk) → tidak dicatat karena api_logs butuh transaction_id.
func (h *Handler) logInbound(c fiber.Ctx, typ string, t *Transaction, err error, ok int, start time.Time) {
	if t == nil {
		return
	}
	status, resp := ok, any(t)
	if err != nil {
		status, resp = fiber.StatusInternalServerError, fiber.Map{"error": err.Error()}
		var fe *fiber.Error
		if errors.As(outboundErr(err), &fe) {
			status = fe.Code
		}
	}
	body, _ := json.Marshal(resp)
	req := json.RawMessage(c.Body())
	if !json.Valid(req) {
		req = nil
	}
	h.logs.Record(c.Context(), apilog.Log{
		TransactionID: t.ID, Type: typ, Method: c.Method(), URL: c.OriginalURL(),
		RequestBody: req, ResponseBody: body, StatusCode: status, DurationMs: int(time.Since(start).Milliseconds()),
	})
}

// CreateQRIS membuat transaksi QRIS lewat PG (outbound).
func (h *Handler) CreateQRIS(c fiber.Ctx) error {
	merchantID, err := h.merchantID(c)
	if err != nil {
		return err
	}

	var request CreateRequest
	if err := c.Bind().Body(&request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	start := time.Now()
	t, err := h.service.CreateQRIS(c.Context(), merchantID, request)
	h.logInbound(c, apilog.InboundCreate, t, err, fiber.StatusCreated, start)
	if err != nil {
		return outboundErr(err)
	}
	return c.Status(fiber.StatusCreated).JSON(t)
}

// Check menanyakan status terbaru ke PG.
func (h *Handler) Check(c fiber.Ctx) error {
	merchantID, err := h.merchantID(c)
	if err != nil {
		return err
	}
	id, err := parseID(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	start := time.Now()
	t, err := h.service.Check(c.Context(), merchantID, id)
	h.logInbound(c, apilog.InboundCheck, t, err, fiber.StatusOK, start)
	if err != nil {
		return outboundErr(err)
	}
	return c.JSON(t)
}

func (h *Handler) Refund(c fiber.Ctx) error {
	merchantID, err := h.merchantID(c)
	if err != nil {
		return err
	}
	id, err := parseID(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.Bind().Body(&body) // reason opsional

	start := time.Now()
	t, err := h.service.Refund(c.Context(), merchantID, id, body.Reason)
	h.logInbound(c, apilog.InboundRefund, t, err, fiber.StatusOK, start)
	if err != nil {
		return outboundErr(err)
	}
	return c.JSON(t)
}

// Notify = webhook payment notify dari PG (publik, diamankan X-SIGNATURE HMAC). Response format SNAP.
func (h *Handler) Notify(secret string) fiber.Handler {
	reply := func(c fiber.Ctx, status int, code, msg string) error {
		return c.Status(status).JSON(gateway.Base{ResponseCode: code, ResponseMessage: msg})
	}

	return func(c fiber.Ctx) error {
		start := time.Now()
		if !gateway.ValidSignature(secret, c.Body(), c.Get("X-SIGNATURE")) {
			return reply(c, fiber.StatusUnauthorized, "4015200", "Unauthorized. [Signature]")
		}

		var n gateway.NotifyReq
		if err := json.Unmarshal(c.Body(), &n); err != nil || n.OriginalPartnerReferenceNo == "" {
			return reply(c, fiber.StatusBadRequest, "4005200", "Invalid Mandatory Field")
		}

		id, err := h.service.Notify(c.Context(), n)
		if errors.Is(err, ErrNotFound) {
			return reply(c, fiber.StatusNotFound, "4045201", "Transaction Not Found")
		}
		if err != nil {
			return err
		}

		response, _ := json.Marshal(gateway.Base{ResponseCode: "2005200", ResponseMessage: "Successful"})
		h.logs.Record(c.Context(), apilog.Log{
			TransactionID: id, Type: apilog.Callback, Method: c.Method(), URL: c.OriginalURL(),
			RequestBody: json.RawMessage(c.Body()), ResponseBody: response,
			StatusCode: fiber.StatusOK, DurationMs: int(time.Since(start).Milliseconds()),
		})
		return c.Send(response)
	}
}

func parseID(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) }
