package utils

import (
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
)

// LoggerConfig определяет конфигурацию для логгера
type LoggerConfig struct {
	// Формат логов (текст/json)
	Format string
	// Выходной поток (os.Stdout, файл и т.д.)
	Output *os.File
	// Включить/выключить цвета для консоли
	EnableColors bool
}

// InitLogger инициализирует и возвращает логгер
func InitLogger(config ...LoggerConfig) *log.Logger {
	var cfg LoggerConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	// Установка вывода по умолчанию
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	// Создаем префикс для логов
	prefix := "[Learning Platform] "

	// Настройка формата логов
	var logger *log.Logger
	if cfg.Format == "json" {
		logger = log.New(cfg.Output, prefix, log.LstdFlags|log.LUTC)
	} else {
		if cfg.EnableColors {
			prefix = "\033[36m" + prefix + "\033[0m" // Голубой цвет
		}
		logger = log.New(cfg.Output, prefix, log.LstdFlags|log.Lshortfile|log.LUTC)
	}

	return logger
}

// LoggingMiddleware возвращает middleware для логирования запросов
func LoggingMiddleware(logger *log.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Передаем управление следующему обработчику
		err := c.Next()

		// Формируем данные для лога
		status := c.Response().StatusCode()
		method := c.Method()
		path := c.Path()
		latency := time.Since(start)
		ip := c.IP()
		userAgent := c.Get("User-Agent")

		// Цвета для разных статусов
		var statusColor, methodColor, resetColor string
		if logger.Flags()&log.Lmsgprefix == 0 {
			statusColor, methodColor, resetColor = getStatusColor(status), getMethodColor(method), "\033[0m"
		}

		logger.Printf("%s %s %s%s%s %s%d%s %s %s %s",
			ip,
			methodColor, method, resetColor,
			path,
			statusColor, status, resetColor,
			latency,
			userAgent,
			err,
		)

		return err
	}
}

func getStatusColor(status int) string {
	switch {
	case status >= 500:
		return "\033[31m" // Красный
	case status >= 400:
		return "\033[33m" // Желтый
	case status >= 300:
		return "\033[36m" // Голубой
	case status >= 200:
		return "\033[32m" // Зеленый
	default:
		return "\033[37m" // Белый
	}
}

func getMethodColor(method string) string {
	switch method {
	case "GET":
		return "\033[34m" // Синий
	case "POST":
		return "\033[33m" // Желтый
	case "PUT":
		return "\033[36m" // Голубой
	case "DELETE":
		return "\033[31m" // Красный
	case "PATCH":
		return "\033[32m" // Зеленый
	default:
		return "\033[37m" // Белый
	}
}
