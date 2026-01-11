// Package debug provides structured logging for gh-subissue.
// Enable debug output by setting GH_DEBUG=1 or GH_DEBUG=api.
package debug

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Logger provides structured debug logging in logfmt style.
type Logger struct {
	out     io.Writer
	enabled bool
}

// instance is the global logger instance.
var instance *Logger

// Init initializes the global debug logger based on GH_DEBUG env var.
func Init() {
	enabled := false
	if val := os.Getenv("GH_DEBUG"); val != "" {
		enabled = true
	}
	instance = &Logger{
		out:     os.Stderr,
		enabled: enabled,
	}
}

// IsEnabled returns true if debug logging is enabled.
func IsEnabled() bool {
	if instance == nil {
		return false
	}
	return instance.enabled
}

// Log writes a structured log message in logfmt format.
// Example: ts=2024-01-10T15:04:05Z level=debug fn=CreateIssue owner=foo repo=bar
func Log(fn string, fields ...interface{}) {
	if instance == nil || !instance.enabled {
		return
	}

	var b strings.Builder
	b.WriteString("ts=")
	b.WriteString(time.Now().UTC().Format(time.RFC3339))
	b.WriteString(" level=debug fn=")
	b.WriteString(fn)

	for i := 0; i < len(fields)-1; i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			continue
		}
		b.WriteString(" ")
		b.WriteString(key)
		b.WriteString("=")
		b.WriteString(formatValue(fields[i+1]))
	}

	b.WriteString("\n")
	fmt.Fprint(instance.out, b.String())
}

// Error logs an error with context.
func Error(fn string, err error, fields ...interface{}) {
	if instance == nil || !instance.enabled {
		return
	}

	allFields := append([]interface{}{"error", err.Error()}, fields...)
	Log(fn, allFields...)
}

// formatValue formats a value for logfmt output.
func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		if strings.ContainsAny(val, " \t\n\"") {
			return fmt.Sprintf("%q", val)
		}
		return val
	case error:
		s := val.Error()
		if strings.ContainsAny(s, " \t\n\"") {
			return fmt.Sprintf("%q", s)
		}
		return s
	default:
		return fmt.Sprintf("%v", v)
	}
}
