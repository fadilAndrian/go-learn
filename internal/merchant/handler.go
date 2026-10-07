package merchant

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

// Handler untuk dashboard admin. Semua route-nya dipasang di belakang admin.Auth
// (cookie sesi admin), jadi di sini tidak ada cek login lagi.
//
// Pola setiap method: ambil input → Validate → panggil service → balas lewat Resource.
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Store: POST /admin/merchants
func (h *Handler) Store(c fiber.Ctx) error {
	var request StoreRequest
	if err := c.Bind().Body(&request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := request.Validate(); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	m, err := h.service.Create(c.Context(), request)
	if err != nil {
		return toHTTP(err)
	}
	return c.Status(fiber.StatusCreated).JSON(NewMerchantResource(m))
}

// Show: GET /admin/merchants/:id
func (h *Handler) Show(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return err
	}

	m, err := h.service.Get(c.Context(), id)
	if err != nil {
		return toHTTP(err)
	}
	return c.JSON(NewMerchantResource(m))
}

// Update: PUT /admin/merchants/:id
func (h *Handler) Update(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return err
	}
	var request UpdateRequest
	if err := c.Bind().Body(&request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	m, err := h.service.Update(c.Context(), id, request)
	if err != nil {
		return toHTTP(err)
	}
	return c.JSON(NewMerchantResource(m))
}

// Destroy: DELETE /admin/merchants/:id
func (h *Handler) Destroy(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return err
	}

	if err := h.service.Delete(c.Context(), id); err != nil {
		return toHTTP(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func paramID(c fiber.Ctx) (int64, error) {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	return id, nil
}

// toHTTP: satu-satunya tempat error modul merchant diterjemahkan ke status HTTP.
// Error yang tidak dikenal diteruskan apa adanya → errorHandler membalas 500 dan mencatat log.
func toHTTP(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	default:
		return err
	}
}
