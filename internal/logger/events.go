package logger

import (
	"OMEGA3-IOT/internal/eventbus"
	"encoding/json"
	"time"
)

// LogLevel represents the severity level of a log entry
type LogLevel string

const (
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelPanic   LogLevel = "panic"
)

// LogEventType represents different types of log events
type LogEventType string

const (
	// Device Events
	LogEventDeviceActionReceived LogEventType = "device.action.received"
	LogEventDeviceActionResult   LogEventType = "device.action.result"
	LogEventDeviceStatusChange   LogEventType = "device.status.change"
	LogEventDevicePropertyUpdate LogEventType = "device.property.update"
	LogEventDeviceLogUpload      LogEventType = "device.log.upload"
	LogEventDeviceError          LogEventType = "device.error"

	// User Events
	LogEventUserLogin        LogEventType = "user.login"
	LogEventUserLogout       LogEventType = "user.logout"
	LogEventUserDeviceShare  LogEventType = "user.device.share"
	LogEventUserDeviceUnshare LogEventType = "user.device.unshare"
	LogEventUserDeviceBind   LogEventType = "user.device.bind"
	LogEventUserPasswordChange LogEventType = "user.password.change"

	// System Events
	LogEventSystemError LogEventType = "system.error"
)

// DeviceLogEvent represents a log entry from a device
type DeviceLogEvent struct {
	eventbus.BaseEvent
	DeviceUUID  string                 `json:"device_uuid"`
	Level       LogLevel               `json:"level"`
	Message     string                 `json:"message"`
	EventType   LogEventType           `json:"event_type"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ActionName  string                 `json:"action_name,omitempty"`
	ActionData  json.RawMessage        `json:"action_data,omitempty"`
	Result      string                 `json:"result,omitempty"`
	ErrorCode   string                 `json:"error_code,omitempty"`
	ErrorDetail string                 `json:"error_detail,omitempty"`
}

// UserLogEvent represents a log entry from user operations
type UserLogEvent struct {
	eventbus.BaseEvent
	UserUUID   string                 `json:"user_uuid"`
	Level      LogLevel               `json:"level"`
	Message    string                 `json:"message"`
	EventType  LogEventType           `json:"event_type"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
}

// SystemLogEvent represents system-level log entries
type SystemLogEvent struct {
	eventbus.BaseEvent
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	EventType LogEventType           `json:"event_type"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewDeviceLogEvent creates a new device log event
func NewDeviceLogEvent(deviceUUID string, level LogLevel, message string, eventType LogEventType) DeviceLogEvent {
	return DeviceLogEvent{
		BaseEvent: eventbus.BaseEvent{
			Type:      eventbus.EventType(eventType),
			Timestamp: time.Now().Unix(),
			Source:    deviceUUID,
		},
		DeviceUUID: deviceUUID,
		Level:      level,
		Message:    message,
		EventType:  eventType,
		Metadata:   make(map[string]interface{}),
	}
}

// NewUserLogEvent creates a new user log event
func NewUserLogEvent(userUUID string, level LogLevel, message string, eventType LogEventType) UserLogEvent {
	return UserLogEvent{
		BaseEvent: eventbus.BaseEvent{
			Type:      eventbus.EventType(eventType),
			Timestamp: time.Now().Unix(),
			Source:    userUUID,
		},
		UserUUID:  userUUID,
		Level:     level,
		Message:   message,
		EventType: eventType,
		Metadata:  make(map[string]interface{}),
	}
}

// NewSystemLogEvent creates a new system log event
func NewSystemLogEvent(level LogLevel, message string, eventType LogEventType) SystemLogEvent {
	return SystemLogEvent{
		BaseEvent: eventbus.BaseEvent{
			Type:      eventbus.EventType(eventType),
			Timestamp: time.Now().Unix(),
			Source:    "system",
		},
		Level:     level,
		Message:   message,
		EventType: eventType,
		Metadata:  make(map[string]interface{}),
	}
}

// LogQuery represents a query for log entries
type LogQuery struct {
	StartTime int64       `json:"start_time"`
	EndTime   int64       `json:"end_time"`
	Level     *LogLevel   `json:"level,omitempty"`
	EventType *LogEventType `json:"event_type,omitempty"`
	Limit     int         `json:"limit"`
	Offset    int         `json:"offset"`
}

// DeviceLogQuery represents a query for device logs
type DeviceLogQuery struct {
	LogQuery
	DeviceUUID string `json:"device_uuid"`
}

// UserLogQuery represents a query for user logs
type UserLogQuery struct {
	LogQuery
	UserUUID string `json:"user_uuid,omitempty"`
}

// LogEntry represents a single log entry returned from queries
type LogEntry struct {
	Timestamp int64                  `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	EventType LogEventType           `json:"event_type"`
	Source    string                 `json:"source"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// LogQueryResponse represents the response for log queries
type LogQueryResponse struct {
	Total   int        `json:"total"`
	Entries []LogEntry `json:"entries"`
}
