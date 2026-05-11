package core

import (
	"fmt"
	"os"
	"strings"
)

// ANSI color codes
const (
	ColorBold       = "\033[1m"
	ColorRed        = "\033[31m"
	ColorBoldRed    = "\033[1;31m"
	ColorYellow     = "\033[33m"
	ColorBoldYellow = "\033[1;33m"
	ColorGreen      = "\033[32m"
	ColorBoldGreen  = "\033[1;32m"
	ColorBlue       = "\033[34m"
	ColorBoldBlue   = "\033[1;34m"
	ColorCyan       = "\033[36m"
	ColorBoldCyan   = "\033[1;36m"
	ColorGrayBold   = "\033[90m"
	ColorReset      = "\033[0m"
	Bold            = "\033[1m"
)

// Logger provides structured logging with colors
type Logger struct {
	logFile *os.File
}

// NewLogger creates a new Logger instance
func NewLogger(logPath string) (*Logger, error) {
	dir := ""
	if idx := lastSlash(logPath); idx > 0 {
		dir = logPath[:idx]
	}

	if dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
	}

	var logFile *os.File
	var err error

	if logPath != "" {
		logFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
	}

	return &Logger{logFile: logFile}, nil
}

// Close closes the log file
func (l *Logger) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	l.log("info", msg, ColorReset)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	l.log("warn", msg, ColorReset)
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	l.log("error", msg, ColorBoldRed)
}

// Command logs a command message
func (l *Logger) Command(msg string) {
	l.log("cmd", msg, ColorBlue)
}

func (l *Logger) log(level, msg, color string) {
	if msg == "" {
		return
	}

	prefix := ""
	switch level {
	case "error":
		prefix = ColorBoldRed + " [ERROR]" + ColorReset
	case "warn":
		prefix = ColorBoldYellow + " [WARN]" + ColorReset
	case "info":
		prefix = ColorBoldBlue + " [INFO]" + ColorReset
	case "cmd":
		prefix = ColorBoldBlue + "  [CMD]" + ColorReset
	}

	prefixBase := ColorGrayBold + "[Bellum-Linux-Installer]" + ColorReset + ":" + prefix
	indentBase := "[Bellum-Linux-Installer]:[ERROR]"
	indent := make([]byte, len(indentBase))
	for i := range indent {
		indent[i] = ' '
	}

	msg = ColorizeSubstr(msg, "WINEPREFIX", ColorBold, color)
	msg = ColorizeSubstr(msg, "WINEPREFIX:", ColorBold, color)
	msg = ColorizeSubstr(msg, "[OK]", ColorGreen, color)

	// Split into lines and wrap if necessary
	lines := splitLines(msg, 125)

	for i, line := range lines {
		var indentStr string
		if i == 0 {
			indentStr = prefixBase + color + "  "
		} else {
			indentStr = string(indent) + color + "  "
		}

		output := indentStr + line + ColorReset

		fmt.Println(output)

		if l.logFile != nil {
			fmt.Fprintln(l.logFile, output)
		}
	}
}

func lastSlash(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}

// Colorize applies a color code to a string
func Colorize(text, color string) string {
	return color + text + ColorReset
}

// ColorizeSubstr applies a color to a specific substring within a larger string
func ColorizeSubstr(s, substring, color, restoreColor string) string {
	if substring == "" {
		return s
	}

	var builder strings.Builder
	builder.Grow(len(s) + len(color)*10)

	remaining := s
	for {
		idx := strings.Index(remaining, substring)
		if idx == -1 {
			builder.WriteString(remaining)
			break
		}

		builder.WriteString(remaining[:idx])
		builder.WriteString(color)
		builder.WriteString(substring)
		builder.WriteString(ColorReset)
		builder.WriteString(restoreColor)
		remaining = remaining[idx+len(substring):]
	}

	return builder.String()
}

func splitLines(s string, maxLen int) []string {
	if len(s) <= maxLen {
		return []string{s}
	}

	var lines []string
	start := 0
	for start < len(s) {
		end := start + maxLen
		if end >= len(s) {
			lines = append(lines, s[start:])
			break
		}
		// Try to break at space
		spaceIdx := -1
		for i := end - 1; i > start; i-- {
			if s[i] == ' ' || s[i] == '\t' {
				spaceIdx = i
				break
			}
		}
		if spaceIdx == -1 {
			spaceIdx = end - 3
			lines = append(lines, s[start:spaceIdx]+"...")
			start = spaceIdx + 3
		} else {
			lines = append(lines, s[start:spaceIdx])
			start = spaceIdx + 1
		}
	}
	return lines
}
