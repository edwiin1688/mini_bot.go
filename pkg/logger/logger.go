package logger

import (
	"log/slog"
	"os"
	"regexp"
	"strings"
)

var (
	defaultLogger *slog.Logger
	sensitiveKeys = []string{
		"api_key", "apikey", "api-key",
		"token", "access_token", "access-token",
		"password", "passwd", "secret",
		"bot_token", "bot-token",
		"private_key", "private-key",
	}
	sensitivePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(api[_-]?key|token|password|secret)["']?\s*[:=]\s*["']?([a-zA-Z0-9_\-\.]{8,})["']?`),
		regexp.MustCompile(`(?i)sk-[a-zA-Z0-9]{20,}`),
		regexp.MustCompile(`(?i)bot[_-]?token["']?\s*[:=]\s*["']?\d{8,}:[a-zA-Z0-9_\-]{20,}`),
	}
)

func sanitizeValue(value string) string {
	for _, pattern := range sensitivePatterns {
		value = pattern.ReplaceAllString(value, "$1=**[REDACTED]**")
	}
	if len(value) > 50 {
		return value[:50] + "..."
	}
	return value
}

func sanitizeArgs(args ...any) []any {
	var sanitized []any
	for i := 0; i < len(args); i++ {
		key, ok := args[i].(string)
		if !ok {
			sanitized = append(sanitized, args[i])
			continue
		}
		keyLower := strings.ToLower(key)
		isSensitive := false
		for _, sk := range sensitiveKeys {
			if strings.Contains(keyLower, sk) {
				isSensitive = true
				break
			}
		}
		sanitized = append(sanitized, key)
		if i+1 < len(args) {
			if isSensitive {
				sanitized = append(sanitized, "[REDACTED]")
			} else {
				sanitized = append(sanitized, args[i+1])
			}
			i++
		}
	}
	return sanitized
}

func sanitizeMessage(msg string) string {
	for _, pattern := range sensitivePatterns {
		msg = pattern.ReplaceAllString(msg, "**[REDACTED]**")
	}
	return msg
}

func Init(debug bool) {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewTextHandler(os.Stderr, opts)
	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

func Debug(msg string, args ...any) {
	slog.Debug(sanitizeMessage(msg), sanitizeArgs(args...)...)
}

func Info(msg string, args ...any) {
	slog.Info(sanitizeMessage(msg), sanitizeArgs(args...)...)
}

func Warn(msg string, args ...any) {
	slog.Warn(sanitizeMessage(msg), sanitizeArgs(args...)...)
}

func Error(msg string, args ...any) {
	slog.Error(sanitizeMessage(msg), sanitizeArgs(args...)...)
}
