package logger

import (
	"OMEGA3-IOT/internal/db"
	"OMEGA3-IOT/internal/eventbus"
	"context"
	"encoding/json"
	"fmt"
	"github.com/apache/iotdb-client-go/client"
	"log"
	"time"
)

// LoggerService handles logging to IoTDB and event processing
type LoggerService struct {
	iotdbClient *db.IOTDBClient
	eventBus    *eventbus.EventBus
}

// LoggerInterface defines the interface for logging operations
type LoggerInterface interface {
	EmitDeviceLog(event DeviceLogEvent)
	EmitUserLog(event UserLogEvent)
	EmitSystemLog(event SystemLogEvent)
}

// NewLoggerService creates a new logger service
func NewLoggerService(iotdbClient *db.IOTDBClient, eventBus *eventbus.EventBus) *LoggerService {
	return &LoggerService{
		iotdbClient: iotdbClient,
		eventBus:    eventBus,
	}
}

// Start initializes event subscriptions
func (ls *LoggerService) Start() {
	// Subscribe to device log events
	eventbus.SubscribeTyped(ls.eventBus, eventbus.EventType(LogEventDeviceLogUpload), ls.handleDeviceLogEvent)
	eventbus.SubscribeTyped(ls.eventBus, eventbus.EventType(LogEventDeviceActionReceived), ls.handleDeviceLogEvent)
	eventbus.SubscribeTyped(ls.eventBus, eventbus.EventType(LogEventDeviceActionResult), ls.handleDeviceLogEvent)
	eventbus.SubscribeTyped(ls.eventBus, eventbus.EventType(LogEventDeviceStatusChange), ls.handleDeviceLogEvent)
	eventbus.SubscribeTyped(ls.eventBus, eventbus.EventType(LogEventDeviceError), ls.handleDeviceLogEvent)

	// Subscribe to user log events
	eventbus.SubscribeTyped(ls.eventBus, eventbus.EventType(LogEventUserLogin), ls.handleUserLogEvent)
	eventbus.SubscribeTyped(ls.eventBus, eventbus.EventType(LogEventUserLogout), ls.handleUserLogEvent)
	eventbus.SubscribeTyped(ls.eventBus, eventbus.EventType(LogEventUserDeviceShare), ls.handleUserLogEvent)
	eventbus.SubscribeTyped(ls.eventBus, eventbus.EventType(LogEventUserDeviceUnshare), ls.handleUserLogEvent)
	eventbus.SubscribeTyped(ls.eventBus, eventbus.EventType(LogEventUserDeviceBind), ls.handleUserLogEvent)
	eventbus.SubscribeTyped(ls.eventBus, eventbus.EventType(LogEventUserPasswordChange), ls.handleUserLogEvent)

	// Subscribe to system log events
	eventbus.SubscribeTyped(ls.eventBus, eventbus.EventType(LogEventSystemError), ls.handleSystemLogEvent)

	log.Println("[LoggerService] Started and subscribed to events")
}

// EmitDeviceLog emits a device log event
func (ls *LoggerService) EmitDeviceLog(event DeviceLogEvent) {
	ls.eventBus.Publish(context.Background(), event)
}

// EmitUserLog emits a user log event
func (ls *LoggerService) EmitUserLog(event UserLogEvent) {
	ls.eventBus.Publish(context.Background(), event)
}

// EmitSystemLog emits a system log event
func (ls *LoggerService) EmitSystemLog(event SystemLogEvent) {
	ls.eventBus.Publish(context.Background(), event)
}

// handleDeviceLogEvent processes device log events
func (ls *LoggerService) handleDeviceLogEvent(ctx context.Context, event DeviceLogEvent) error {
	return ls.writeDeviceLog(event)
}

// handleUserLogEvent processes user log events
func (ls *LoggerService) handleUserLogEvent(ctx context.Context, event UserLogEvent) error {
	return ls.writeUserLog(event)
}

// handleSystemLogEvent processes system log events
func (ls *LoggerService) handleSystemLogEvent(ctx context.Context, event SystemLogEvent) error {
	return ls.writeSystemLog(event)
}

// writeDeviceLog writes device log to IoTDB
func (ls *LoggerService) writeDeviceLog(event DeviceLogEvent) error {
	devicePath := fmt.Sprintf("root.mm1.device_data.%s.log", event.DeviceUUID)
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)

	measurements := []string{"level", "message", "event_type", "metadata"}
	dataTypes := []client.TSDataType{client.STRING, client.STRING, client.STRING, client.STRING}

	metadataJSON, _ := json.Marshal(event.Metadata)
	values := []interface{}{
		string(event.Level),
		event.Message,
		string(event.EventType),
		string(metadataJSON),
	}

	// Add optional fields if present
	if event.ActionName != "" {
		measurements = append(measurements, "action_name")
		dataTypes = append(dataTypes, client.STRING)
		values = append(values, event.ActionName)
	}
	if event.Result != "" {
		measurements = append(measurements, "result")
		dataTypes = append(dataTypes, client.STRING)
		values = append(values, event.Result)
	}
	if event.ErrorCode != "" {
		measurements = append(measurements, "error_code", "error_detail")
		dataTypes = append(dataTypes, client.STRING, client.STRING)
		values = append(values, event.ErrorCode, event.ErrorDetail)
	}

	session, err := ls.iotdbClient.SessionPool.GetSession()
	if err != nil {
		return fmt.Errorf("[LoggerService] failed to get session: %w", err)
	}
	defer ls.iotdbClient.SessionPool.PutBack(session)

	_, err = session.InsertRecord(devicePath, measurements, dataTypes, values, timestamp)
	if err != nil {
		return fmt.Errorf("[LoggerService] failed to write device log: %w", err)
	}

	return nil
}

// writeUserLog writes user log to IoTDB
func (ls *LoggerService) writeUserLog(event UserLogEvent) error {
	userPath := "root.mm1.user_data.log"
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)

	measurements := []string{"user_uuid", "level", "message", "event_type", "metadata", "ip_address", "user_agent"}
	dataTypes := []client.TSDataType{client.STRING, client.STRING, client.STRING, client.STRING, client.STRING, client.STRING, client.STRING}

	metadataJSON, _ := json.Marshal(event.Metadata)
	values := []interface{}{
		event.UserUUID,
		string(event.Level),
		event.Message,
		string(event.EventType),
		string(metadataJSON),
		event.IPAddress,
		event.UserAgent,
	}

	session, err := ls.iotdbClient.SessionPool.GetSession()
	if err != nil {
		return fmt.Errorf("[LoggerService] failed to get session: %w", err)
	}
	defer ls.iotdbClient.SessionPool.PutBack(session)

	_, err = session.InsertRecord(userPath, measurements, dataTypes, values, timestamp)
	if err != nil {
		return fmt.Errorf("[LoggerService] failed to write user log: %w", err)
	}

	return nil
}

// writeSystemLog writes system log to IoTDB
func (ls *LoggerService) writeSystemLog(event SystemLogEvent) error {
	// System logs go to user_data.log with user_uuid="system"
	userPath := "root.mm1.user_data.log"
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)

	measurements := []string{"user_uuid", "level", "message", "event_type", "metadata"}
	dataTypes := []client.TSDataType{client.STRING, client.STRING, client.STRING, client.STRING, client.STRING}

	metadataJSON, _ := json.Marshal(event.Metadata)
	values := []interface{}{
		"system",
		string(event.Level),
		event.Message,
		string(event.EventType),
		string(metadataJSON),
	}

	session, err := ls.iotdbClient.SessionPool.GetSession()
	if err != nil {
		return fmt.Errorf("[LoggerService] failed to get session: %w", err)
	}
	defer ls.iotdbClient.SessionPool.PutBack(session)

	_, err = session.InsertRecord(userPath, measurements, dataTypes, values, timestamp)
	if err != nil {
		return fmt.Errorf("[LoggerService] failed to write system log: %w", err)
	}

	return nil
}

// QueryDeviceLogs queries device logs from IoTDB
func (ls *LoggerService) QueryDeviceLogs(query DeviceLogQuery) (*LogQueryResponse, error) {
	devicePath := fmt.Sprintf("root.mm1.device_data.%s.log", query.DeviceUUID)
	return ls.queryLogs(devicePath, query.LogQuery)
}

// QueryUserLogs queries user logs from IoTDB
func (ls *LoggerService) QueryUserLogs(query UserLogQuery) (*LogQueryResponse, error) {
	userPath := "root.mm1.user_data.log"
	response, err := ls.queryLogs(userPath, query.LogQuery)
	if err != nil {
		return nil, err
	}

	// Filter by user_uuid if specified
	if query.UserUUID != "" {
		var filtered []LogEntry
		for _, entry := range response.Entries {
			if entry.Metadata != nil && entry.Metadata["user_uuid"] == query.UserUUID {
				filtered = append(filtered, entry)
			}
		}
		response.Entries = filtered
		response.Total = len(filtered)
	}

	return response, nil
}

// queryLogs is the internal implementation for querying logs
func (ls *LoggerService) queryLogs(path string, query LogQuery) (*LogQueryResponse, error) {
	session, err := ls.iotdbClient.SessionPool.GetSession()
	if err != nil {
		return nil, fmt.Errorf("[LoggerService] failed to get session: %w", err)
	}
	defer ls.iotdbClient.SessionPool.PutBack(session)

	// Build SQL query
	sql := fmt.Sprintf("SELECT * FROM %s WHERE time >= %d AND time <= %d ORDER BY time DESC LIMIT %d OFFSET %d",
		path,
		query.StartTime*1000, // Convert to milliseconds
		query.EndTime*1000,
		query.Limit,
		query.Offset)

	dataSet, err := session.ExecuteQueryStatement(sql, &ls.iotdbClient.Config.IoTDB.QueryTimeoutMs)
	if err != nil {
		return nil, fmt.Errorf("[LoggerService] failed to execute query: %w", err)
	}
	defer dataSet.Close()

	var entries []LogEntry
	for {
		hasNext, err := dataSet.Next()
		if err != nil {
			return nil, fmt.Errorf("[LoggerService] failed to iterate result: %w", err)
		}
		if !hasNext {
			break
		}

		record, err := dataSet.GetRowRecord()
		if err != nil {
			return nil, fmt.Errorf("[LoggerService] failed to get row record: %w", err)
		}

		timestamp := record.GetTimestamp() / 1000 // Convert to seconds
		entry := LogEntry{
			Timestamp: timestamp,
			Metadata:  make(map[string]interface{}),
		}

		// Parse columns
		columnCount := dataSet.GetColumnCount()
		for i := 0; i < columnCount; i++ {
			columnName := dataSet.GetColumnName(i)
			value := dataSet.GetValue(columnName)

			switch columnName {
			case "level":
				if v, ok := value.(string); ok {
					entry.Level = LogLevel(v)
				}
			case "message":
				if v, ok := value.(string); ok {
					entry.Message = v
				}
			case "event_type":
				if v, ok := value.(string); ok {
					entry.EventType = LogEventType(v)
				}
			case "metadata":
				if v, ok := value.(string); ok {
					var metadata map[string]interface{}
					json.Unmarshal([]byte(v), &metadata)
					entry.Metadata = metadata
				}
			default:
				if entry.Metadata == nil {
					entry.Metadata = make(map[string]interface{})
				}
				entry.Metadata[columnName] = value
			}
		}

		entries = append(entries, entry)
	}

	return &LogQueryResponse{
		Total:   len(entries),
		Entries: entries,
	}, nil
}

// InitializeLogSchema creates the necessary timeseries for logging
func (ls *LoggerService) InitializeLogSchema() error {
	session, err := ls.iotdbClient.SessionPool.GetSession()
	if err != nil {
		return fmt.Errorf("[LoggerService] failed to get session: %w", err)
	}
	defer ls.iotdbClient.SessionPool.PutBack(session)

	// Create user_data storage group if not exists
	status, err := session.SetStorageGroup("root.mm1.user_data")
	if checkErr := ls.iotdbClient.CheckError(status, err); checkErr != nil {
		log.Printf("[LoggerService] Storage group may already exist: %v", checkErr)
	}

	// Create log timeseries for user_data
	logPath := "root.mm1.user_data.log"
	logMeasurements := []struct {
		name     string
		dataType string
	}{
		{"user_uuid", "STRING"},
		{"level", "STRING"},
		{"message", "STRING"},
		{"event_type", "STRING"},
		{"metadata", "STRING"},
		{"ip_address", "STRING"},
		{"user_agent", "STRING"},
	}

	for _, m := range logMeasurements {
		timeseriesPath := fmt.Sprintf("%s.%s", logPath, m.name)
		sql := fmt.Sprintf("CREATE TIMESERIES %s WITH DATATYPE=%s, ENCODING=PLAIN, COMPRESSION=SNAPPY",
			timeseriesPath, m.dataType)
		status, err := session.ExecuteNonQueryStatement(sql)
		if checkErr := ls.iotdbClient.CheckError(status, err); checkErr != nil {
			log.Printf("[LoggerService] Timeseries %s may already exist: %v", m.name, checkErr)
		}
	}

	log.Println("[LoggerService] Log schema initialized")
	return nil
}
