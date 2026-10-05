package transaction

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/fadilAndrian/go-learn/internal/apilog"
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

func (h *Handler) Create(c fiber.Ctx) error {
	merchantID, err := h.merchantID(c)
	if err != nil {
		return err
	}

	var request CreateRequest
	if err := c.Bind().Body(&request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	start := time.Now()
	t, err := h.service.Create(c.Context(), merchantID, request)
	if errors.Is(err, ErrInvalidAmount) {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if errors.Is(err, ErrReferenceInUse) {
		return fiber.NewError(fiber.StatusConflict, err.Error())
	}
	if err != nil {
		return err
	}

	response, _ := json.Marshal(t)
	h.logs.Record(c.Context(), apilog.Log{
		TransactionID: t.ID,
		Type:          apilog.InboundCreate,
		Method:        c.Method(),
		URL:           c.OriginalURL(),
		RequestBody:   json.RawMessage(c.Body()),
		ResponseBody:  response,
		StatusCode:    fiber.StatusCreated,
		DurationMs:    int(time.Since(start).Milliseconds()),
	})

	return c.Status(fiber.StatusCreated).JSON(t)
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

func parseID(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) }
