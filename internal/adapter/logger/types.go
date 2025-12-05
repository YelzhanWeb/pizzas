// internal/adapter/logger/types.go
package logger

type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Service   string                 `json:"service"`
	Hostname  string                 `json:"hostname"`
	RequestID string                 `json:"request_id"`
	Action    string                 `json:"action"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Error     *ErrorInfo             `json:"error,omitempty"`
}

type ErrorInfo struct {
	Msg   string `json:"msg"`
	Stack string `json:"stack"`
}
