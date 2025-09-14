package logger

import (
	"os"
	"strings"

	"github.com/fatih/color"
)

// Logger represents a structured logger
type Logger struct {
	level   LogLevel
	format  LogFormat
	console bool
	file    *os.File
}

// LogLevel represents the logging level
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// LogFormat represents the log output format
type LogFormat int

const (
	JSONFormat LogFormat = iota
	TextFormat
)

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level   string
	File    string
	Format  string
	Console bool
}

// New creates a new logger instance
func New(config LoggingConfig) (*Logger, error) {
	logger := &Logger{
		level:   parseLogLevel(config.Level),
		format:  parseLogFormat(config.Format),
		console: config.Console,
	}

	// Open log file if specified
	if config.File != "" {
		file, err := os.OpenFile(config.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		logger.file = file
	}

	return logger, nil
}

// parseLogLevel parses string log level to LogLevel
func parseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	case "fatal":
		return FatalLevel
	default:
		return InfoLevel
	}
}

// parseLogFormat parses string log format to LogFormat
func parseLogFormat(format string) LogFormat {
	switch strings.ToLower(format) {
	case "json":
		return JSONFormat
	case "text", "console":
		return TextFormat
	default:
		return JSONFormat
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	if l.level <= DebugLevel {
		l.log(DebugLevel, "🔍", msg, keysAndValues...)
	}
}

// Info logs an info message
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	if l.level <= InfoLevel {
		l.log(InfoLevel, "ℹ️", msg, keysAndValues...)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	if l.level <= WarnLevel {
		l.log(WarnLevel, "⚠️", msg, keysAndValues...)
	}
}

// Error logs an error message
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	if l.level <= ErrorLevel {
		l.log(ErrorLevel, "❌", msg, keysAndValues...)
	}
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, keysAndValues ...interface{}) {
	l.log(FatalLevel, "💀", msg, keysAndValues...)
	os.Exit(1)
}

// log handles the actual logging
func (l *Logger) log(level LogLevel, icon, msg string, keysAndValues ...interface{}) {
	if l.console {
		l.logConsole(level, icon, msg, keysAndValues...)
	}

	if l.file != nil {
		l.logFile(level, msg, keysAndValues...)
	}
}

// logConsole logs to console with colors
func (l *Logger) logConsole(level LogLevel, icon, msg string, keysAndValues ...interface{}) {
	var colorFunc func(a ...interface{}) string

	switch level {
	case DebugLevel:
		colorFunc = color.New(color.FgCyan).SprintFunc()
	case InfoLevel:
		colorFunc = color.New(color.FgGreen).SprintFunc()
	case WarnLevel:
		colorFunc = color.New(color.FgYellow).SprintFunc()
	case ErrorLevel:
		colorFunc = color.New(color.FgRed).SprintFunc()
	case FatalLevel:
		colorFunc = color.New(color.FgRed, color.Bold).SprintFunc()
	}

	output := colorFunc(icon + " " + msg)

	// Add key-value pairs
	if len(keysAndValues) > 0 {
		output += " " + formatKeyValues(keysAndValues...)
	}

	os.Stdout.WriteString(output + "\n")
}

// logFile logs to file
func (l *Logger) logFile(level LogLevel, msg string, keysAndValues ...interface{}) {
	// Implementation depends on format (JSON vs text)
	// For now, simple text format
	levelStr := levelToString(level)
	output := levelStr + " " + msg

	if len(keysAndValues) > 0 {
		output += " " + formatKeyValues(keysAndValues...)
	}

	l.file.WriteString(output + "\n")
}

// formatKeyValues formats key-value pairs
func formatKeyValues(keysAndValues ...interface{}) string {
	var parts []string
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key := keysAndValues[i]
			value := keysAndValues[i+1]
			parts = append(parts, key.(string)+"="+stringify(value))
		}
	}
	return strings.Join(parts, " ")
}

// stringify converts a value to string
func stringify(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case error:
		return val.Error()
	default:
		return ""
	}
}

// levelToString converts LogLevel to string
func levelToString(level LogLevel) string {
	switch level {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "INFO"
	}
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	if l.file != nil {
		return l.file.Sync()
	}
	return nil
}

// Close closes the logger and any open file handles
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
