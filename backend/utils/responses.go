package utils

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// SuccessResponse структура для успешных ответов
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// ErrorResponse структура для ошибок
type ErrorResponse struct {
	Success bool        `json:"success"`
	Error   string      `json:"error"`
	Message string      `json:"message,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

// Success создает успешный JSON ответ
func Success(c *fiber.Ctx, status int, data interface{}, meta ...interface{}) error {
	response := SuccessResponse{
		Success: true,
		Data:    data,
	}

	if len(meta) > 0 {
		response.Meta = meta[0]
	}

	return c.Status(status).JSON(response)
}

// Error создает JSON ответ с ошибкой
func Error(c *fiber.Ctx, status int, err error, details ...interface{}) error {
	response := ErrorResponse{
		Success: false,
		Error:   http.StatusText(status),
		Message: err.Error(),
	}

	if len(details) > 0 {
		response.Details = details[0]
	}

	return c.Status(status).JSON(response)
}

// PaginatedResponse структура для пагинированных ответов
type PaginatedResponse struct {
	Success  bool        `json:"success"`
	Data     interface{} `json:"data"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

// Paginate создает пагинированный JSON ответ
func Paginate(c *fiber.Ctx, data interface{}, total int64, page int, pageSize int) error {
	return c.JSON(PaginatedResponse{
		Success:  true,
		Data:     data,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// ValidationError создает JSON ответ для ошибок валидации
func ValidationError(c *fiber.Ctx, errors map[string]string) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorResponse{
		Success: false,
		Error:   "Validation Error",
		Details: errors,
	})
}

// Created отправляет ответ 201 Created
func Created(c *fiber.Ctx, data interface{}) error {
	return Success(c, fiber.StatusCreated, data)
}

// NoContent отправляет ответ 204 No Content
func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// NotFound отправляет ответ 404 Not Found
func NotFound(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusNotFound, fiber.NewError(fiber.StatusNotFound, message))
}

// BadRequest отправляет ответ 400 Bad Request
func BadRequest(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusBadRequest, fiber.NewError(fiber.StatusBadRequest, message))
}

// Unauthorized отправляет ответ 401 Unauthorized
func Unauthorized(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusUnauthorized, fiber.NewError(fiber.StatusUnauthorized, message))
}

// Forbidden отправляет ответ 403 Forbidden
func Forbidden(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusForbidden, fiber.NewError(fiber.StatusForbidden, message))
}

// InternalServerError отправляет ответ 500 Internal Server Error
func InternalServerError(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusInternalServerError, fiber.NewError(fiber.StatusInternalServerError, message))
}
