package logger

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

type Logger interface {
	Info(action, message, requestID string, details map[string]interface{})
	Debug(action, message, requestID string, details map[string]interface{})
	Error(action, message, requestID string, details map[string]interface{}, err error)
}

type jsonLogger struct {
	service  string
	hostname string
	mu       sync.Mutex
}

func New(service string) Logger {
	hostname, _ := os.Hostname()
	return &jsonLogger{
		service:  service,
		hostname: hostname,
	}
}

func (l *jsonLogger) Info(action, message, requestID string, details map[string]interface{}) {
	l.log("INFO", action, message, requestID, details, nil)
}

func (l *jsonLogger) Debug(action, message, requestID string, details map[string]interface{}) {
	l.log("DEBUG", action, message, requestID, details, nil)
}

func (l *jsonLogger) Error(action, message, requestID string, details map[string]interface{}, err error) {
	l.log("ERROR", action, message, requestID, details, err)
}

func (l *jsonLogger) log(level, action, message, requestID string, details map[string]interface{}, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level,
		Service:   l.service,
		Hostname:  l.hostname,
		RequestID: requestID,
		Action:    action,
		Message:   message,
		Details:   details,
	}

	if err != nil {
		entry.Error = &ErrorInfo{
			Msg:   err.Error(),
			Stack: err.Error(),
		}
	}

	json.NewEncoder(os.Stdout).Encode(entry)
}
