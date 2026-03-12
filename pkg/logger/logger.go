// pkg/logger/logger.go

package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Logger provides structured logging capabilities
type Logger struct {
	prefix string
	level  LogLevel
	output *log.Logger
}

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// NewLogger creates a new logger instance
func NewLogger() *Logger {
	return &Logger{
		prefix: "",
		level:  INFO,
		output: log.New(os.Stdout, "", 0),
	}
}

// NewLoggerWithPrefix creates a logger with a prefix
func NewLoggerWithPrefix(prefix string) *Logger {
	return &Logger{
		prefix: prefix,
		level:  INFO,
		output: log.New(os.Stdout, "", 0),
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// log is the internal logging method
func (l *Logger) log(level LogLevel, msg string, keysAndValues ...interface{}) {
	if level < l.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	prefix := ""
	if l.prefix != "" {
		prefix = fmt.Sprintf("[%s] ", l.prefix)
	}

	// Format key-value pairs
	kvPairs := ""
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			kvPairs += fmt.Sprintf(" %v=%v", keysAndValues[i], keysAndValues[i+1])
		}
	}

	logLine := fmt.Sprintf("[%s] %s%s: %s%s", timestamp, prefix, level.String(), msg, kvPairs)
	l.output.Println(logLine)

	if level == FATAL {
		os.Exit(1)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.log(DEBUG, msg, keysAndValues...)
}

// Info logs an info message
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.log(INFO, msg, keysAndValues...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	l.log(WARN, msg, keysAndValues...)
}

// Error logs an error message
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	l.log(ERROR, msg, keysAndValues...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, keysAndValues ...interface{}) {
	l.log(FATAL, msg, keysAndValues...)
}

// WithFields creates a new logger with predefined fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	// Create a shallow copy
	newLogger := &Logger{
		prefix: l.prefix,
		level:  l.level,
		output: l.output,
	}
	return newLogger
}

// SetOutput sets the output destination
func (l *Logger) SetOutput(output *log.Logger) {
	l.output = output
}
