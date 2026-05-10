package api

import (
	"fmt"
)

// LogAppender defines the interface for different log output strategies.
type LogAppender interface {
	Append(level, message string)
}

// TerminalAppender is a strategy that writes logs to the terminal.
type TerminalAppender struct{}

func (t *TerminalAppender) Append(level, message string) {
	fmt.Printf("[%s] %s\n", level, message)
}

var (
	currentAppender LogAppender = &TerminalAppender{}
)

// SetLogAppender sets the logging strategy for the application.
func SetLogAppender(appender LogAppender) {
	currentAppender = appender
}

// Log writes a message using the current logging strategy.
func Log(level, message string) {
	if currentAppender != nil {
		currentAppender.Append(level, message)
	}
}

// LogInfo is a helper for INFO level logs.
func LogInfo(message string) {
	Log("INFO", message)
}

// LogError is a helper for ERROR level logs.
func LogError(message string) {
	Log("ERROR", message)
}
