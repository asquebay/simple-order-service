package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New создаёт и настраивает новый экземпляр slog.Logger
// уровень логирования определяется строковым параметром
func New(levelStr string) *slog.Logger {
	var level slog.Level

	// преобразуем строковый уровень из конфига в slog.Level
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		// по умолчанию используем INFO, если в конфиге указано что-то некорректное
		level = slog.LevelInfo
	}

	// создаем обработчик для локальной разработки
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true, // нужно, чтобы видеть файл и строку, откуда был вызов лога
		Level:     level,
	})

	// TODO: создать обработчик NewJSONHandler для продакшена

	// создаём логгер с нашим обработчиком
	logger := slog.New(handler)

	return logger
}
