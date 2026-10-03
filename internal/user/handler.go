package user

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(c fiber.Ctx) error {
	var request CreateUserRequest
	if err := c.Bind().Body(&request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if request.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name are required")
	}

	if request.Email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email are required")
	}

	if request.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "password are required")
	}

	user, err := h.service.RegisterUser(c.Context(), request)
	if errors.Is(err, ErrEmailInUse) {
		return fiber.NewError(fiber.StatusConflict, err.Error())
	}
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(user)
}

func (h *Handler) Login(c fiber.Ctx) error {
	var request LoginRequest
	if err := c.Bind().Body(&request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	user, token, err := h.service.Login(c.Context(), request.Email, request.Password)
	if errors.Is(err, ErrInvalidCredentials) {
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"token": token, "user": user})
}

func (h *Handler) Me(c fiber.Ctx) error {
	id, _ := c.Locals(userIDKey).(int64)

	user, err := h.service.GetUserByID(c.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		return fiber.NewError(fiber.StatusUnauthorized, "user not found")
	}
	if err != nil {
		return err
	}

	return c.JSON(user)
}
