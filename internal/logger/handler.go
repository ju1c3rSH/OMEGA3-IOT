package logger

import (
	"OMEGA3-IOT/internal/eventbus"
	"OMEGA3-IOT/internal/types"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// LogHandler handles HTTP requests for logging
type LogHandler struct {
	loggerService *LoggerService
}

// NewLogHandler creates a new log handler
func NewLogHandler(loggerService *LoggerService) *LogHandler {
	return &LogHandler{loggerService: loggerService}
}

// UploadDeviceLogRequest represents a device log upload request
type UploadDeviceLogRequest struct {
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

// UploadDeviceLog handles device log uploads via HTTP
func (h *LogHandler) UploadDeviceLog(c *gin.Context) {
	var req UploadDeviceLogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response := types.NewErrorResponse(http.StatusBadRequest, "Invalid request body", err.Error())
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Validate required fields
	if req.DeviceUUID == "" || req.Message == "" {
		response := types.NewErrorResponse(http.StatusBadRequest, "Missing required fields", "device_uuid and message are required")
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Set default level if not provided
	if req.Level == "" {
		req.Level = LogLevelInfo
	}

	// Set default event type if not provided
	if req.EventType == "" {
		req.EventType = LogEventDeviceLogUpload
	}

	event := DeviceLogEvent{
		BaseEvent: eventbus.BaseEvent{
			Type:      eventbus.EventType(req.EventType),
			Timestamp: time.Now().Unix(),
			Source:    req.DeviceUUID,
		},
		DeviceUUID:  req.DeviceUUID,
		Level:       req.Level,
		Message:     req.Message,
		EventType:   req.EventType,
		Metadata:    req.Metadata,
		ActionName:  req.ActionName,
		ActionData:  req.ActionData,
		Result:      req.Result,
		ErrorCode:   req.ErrorCode,
		ErrorDetail: req.ErrorDetail,
	}

	h.loggerService.EmitDeviceLog(event)

	response := types.NewSuccessResponse("Log uploaded successfully")
	c.JSON(http.StatusOK, response)
}

// QueryDeviceLogs handles device log queries
func (h *LogHandler) QueryDeviceLogs(c *gin.Context) {
	deviceUUID := c.Query("device_uuid")
	if deviceUUID == "" {
		response := types.NewErrorResponse(http.StatusBadRequest, "Missing parameter", "device_uuid is required")
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Parse query parameters
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")
	limitStr := c.Query("limit")
	offsetStr := c.Query("offset")

	startTime := time.Now().Add(-24 * time.Hour).Unix() // Default to last 24 hours
	endTime := time.Now().Unix()
	limit := 100
	offset := 0

	if startTimeStr != "" {
		if t, err := strconv.ParseInt(startTimeStr, 10, 64); err == nil {
			startTime = t
		}
	}
	if endTimeStr != "" {
		if t, err := strconv.ParseInt(endTimeStr, 10, 64); err == nil {
			endTime = t
		}
	}
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	query := DeviceLogQuery{
		LogQuery: LogQuery{
			StartTime: startTime,
			EndTime:   endTime,
			Limit:     limit,
			Offset:    offset,
		},
		DeviceUUID: deviceUUID,
	}

	result, err := h.loggerService.QueryDeviceLogs(query)
	if err != nil {
		response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to query logs", err.Error())
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := types.NewSuccessResponse(result)
	c.JSON(http.StatusOK, response)
}

// QueryUserLogs handles user log queries
func (h *LogHandler) QueryUserLogs(c *gin.Context) {
	userUUID := c.Query("user_uuid")

	// Parse query parameters
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")
	limitStr := c.Query("limit")
	offsetStr := c.Query("offset")

	startTime := time.Now().Add(-24 * time.Hour).Unix()
	endTime := time.Now().Unix()
	limit := 100
	offset := 0

	if startTimeStr != "" {
		if t, err := strconv.ParseInt(startTimeStr, 10, 64); err == nil {
			startTime = t
		}
	}
	if endTimeStr != "" {
		if t, err := strconv.ParseInt(endTimeStr, 10, 64); err == nil {
			endTime = t
		}
	}
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	query := UserLogQuery{
		LogQuery: LogQuery{
			StartTime: startTime,
			EndTime:   endTime,
			Limit:     limit,
			Offset:    offset,
		},
		UserUUID: userUUID,
	}

	result, err := h.loggerService.QueryUserLogs(query)
	if err != nil {
		response := types.NewErrorResponse(http.StatusInternalServerError, "Failed to query logs", err.Error())
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := types.NewSuccessResponse(result)
	c.JSON(http.StatusOK, response)
}
