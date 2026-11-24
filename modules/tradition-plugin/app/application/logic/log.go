package logic

import "log/slog"

const logComponent = "tradition-plugin"

func logInfo(message string, args ...any) {
	slog.Info(message, append([]any{"component", logComponent}, args...)...)
}

func logWarn(message string, args ...any) {
	slog.Warn(message, append([]any{"component", logComponent}, args...)...)
}

func logError(message string, args ...any) {
	slog.Error(message, append([]any{"component", logComponent}, args...)...)
}
