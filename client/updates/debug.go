package updates

import "log/slog"

// debugLog prints a message only if debug logging is enabled
// Uses slog which respects the log level set by the main program
// Accepts a message and key-value pairs like slog.Debug
func debugLog(msg string, args ...interface{}) {
	slog.Debug(msg, args...)
}
