package repository

import (
	"OMEGA3-IOT/internal/db"
	"OMEGA3-IOT/internal/utils"
	"fmt"
	"github.com/apache/iotdb-client-go/client"
	"strings"
)

// TelemetryData represents a single telemetry data point
type TelemetryData struct {
	DeviceUUID   string
	Measurements []string
	Values       map[string]interface{}
	Timestamp    int64
}

// TelemetryQueryResult represents the result of a telemetry query
type TelemetryQueryResult struct {
	Timestamp  int64
	DeviceUUID string
	Values     map[string]interface{}
}

var tsTypeNames = map[client.TSDataType]string{
	client.BOOLEAN:   "BOOLEAN",
	client.INT32:     "INT32",
	client.INT64:     "INT64",
	client.FLOAT:     "FLOAT",
	client.DOUBLE:    "DOUBLE",
	client.TEXT:      "TEXT",
	client.TIMESTAMP: "TIMESTAMP",
	client.DATE:      "DATE",
	client.BLOB:      "BLOB",
	client.STRING:    "STRING",
}

func TSDataTypeToString(t client.TSDataType) string {
	if name, ok := tsTypeNames[t]; ok {
		return name
	}
	return fmt.Sprintf("UNKNOWN(%d)", t)
}

func StringToTSDataType(s string) (client.TSDataType, error) {
	tsTypeMap := map[string]client.TSDataType{
		"BOOLEAN":   client.BOOLEAN,
		"INT32":     client.INT32,
		"INT64":     client.INT64,
		"FLOAT":     client.FLOAT,
		"DOUBLE":    client.DOUBLE,
		"TEXT":      client.TEXT,
		"TIMESTAMP": client.TIMESTAMP,
		"DATE":      client.DATE,
		"BLOB":      client.BLOB,
		"STRING":    client.STRING,
	}
	if val, ok := tsTypeMap[s]; ok {
		return val, nil
	}
	return client.UNKNOWN, fmt.Errorf("unknown TSDataType: %s", s)
}

type TelemetryRepository interface {
	InsertTelemetry(deviceUUID string, measurements []string, values []interface{}, timestamp int64) error
	BatchInsertTelemetry(telemetryData []TelemetryData) error
	QueryTelemetry(deviceUUID string, startTime, endTime int64) (*TelemetryData, error)
	QueryLatestTelemetry(deviceUUID string) (*TelemetryData, error)
	CreateTimeseries(deviceUUID string, propertyNames []string, dataTypes []client.TSDataType) error
}

type iotdbTelemetryRepository struct {
	client *db.IOTDBClient
}

func NewTelemetryRepository(client *db.IOTDBClient) TelemetryRepository {
	return &iotdbTelemetryRepository{client: client}
}

func (r *iotdbTelemetryRepository) InsertTelemetry(deviceUUID string, measurements []string, values []interface{}, timestamp int64) error {
	devicePath := utils.ConvertHyphenIntoDash(fmt.Sprintf("root.mm1.device_data.%s", deviceUUID))
	return r.client.InsertRecordTyped(devicePath, measurements, nil, values, timestamp)
}

func (r *iotdbTelemetryRepository) BatchInsertTelemetry(telemetryData []TelemetryData) error {
	// Batch insert implementation
	for _, data := range telemetryData {
		// Convert map to slice based on measurements order
		values := make([]interface{}, len(data.Measurements))
		for i, m := range data.Measurements {
			values[i] = data.Values[m]
		}
		if err := r.InsertTelemetry(data.DeviceUUID, data.Measurements, values, data.Timestamp); err != nil {
			return err
		}
	}
	return nil
}

func (r *iotdbTelemetryRepository) QueryTelemetry(deviceUUID string, startTime, endTime int64) (*TelemetryData, error) {
	var tc utils.TimeConverter
	devicePath := utils.ConvertHyphenIntoDash(fmt.Sprintf("root.mm1.device_data.%s", deviceUUID))
	sql := fmt.Sprintf("SELECT * FROM %s WHERE time >= %s AND time <= %s ORDER BY time DESC",
		devicePath, tc.SecToISO(startTime), tc.SecToISO(endTime))

	session, err := r.client.SessionPool.GetSession()
	if err != nil {
		return nil, err
	}
	defer r.client.SessionPool.PutBack(session)

	dataSet, err := session.ExecuteQueryStatement(sql, &r.client.Config.IoTDB.QueryTimeoutMs)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer dataSet.Close()

	var results TelemetryData
	for {
		hasNext, err := dataSet.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate result: %w", err)
		}
		if !hasNext {
			break
		}

		record, err := dataSet.GetRowRecord()
		if err != nil {
			return nil, fmt.Errorf("failed to get row record: %w", err)
		}

		timestamp := record.GetTimestamp()
		values := make(map[string]interface{})
		columnCount := dataSet.GetColumnCount()
		var parts, propNames []string
		for i := 0; i < columnCount; i++ {

			columnName := dataSet.GetColumnName(i)
			parts = strings.Split(columnName, ".")
			propName := parts[len(parts)-1]

			value := dataSet.GetValue(columnName)
			propNames = append(propNames, propName)
			values[propName] = value
		}

		results = TelemetryData{
			DeviceUUID:   deviceUUID,
			Timestamp:    timestamp,
			Values:       values,
			Measurements: propNames,
		}
	}

	return &results, nil
}

func (r *iotdbTelemetryRepository) QueryLatestTelemetry(deviceUUID string) (*TelemetryData, error) {
	devicePath := utils.ConvertHyphenIntoDash(fmt.Sprintf("root.mm1.device_data.%s", deviceUUID))
	sql := fmt.Sprintf("SELECT * FROM %s ORDER BY time DESC LIMIT 1", devicePath)

	session, err := r.client.SessionPool.GetSession()
	if err != nil {
		return &TelemetryData{}, err
	}
	defer r.client.SessionPool.PutBack(session)

	dataSet, err := session.ExecuteQueryStatement(sql, &r.client.Config.IoTDB.QueryTimeoutMs)
	if err != nil {
		return &TelemetryData{}, fmt.Errorf("failed to execute query: %w", err)
	}
	defer dataSet.Close()

	hasNext, err := dataSet.Next()
	if err != nil {
		return &TelemetryData{}, fmt.Errorf("failed to iterate result: %w", err)
	}
	if !hasNext {
		return &TelemetryData{}, fmt.Errorf("no data found for device %s", deviceUUID)
	}

	record, err := dataSet.GetRowRecord()
	if err != nil {
		return &TelemetryData{}, fmt.Errorf("failed to get row record: %w", err)
	}

	timestamp := record.GetTimestamp()
	values := make(map[string]interface{})
	columnCount := dataSet.GetColumnCount()

	for i := 0; i < columnCount; i++ {
		columnName := dataSet.GetColumnName(i)
		value := dataSet.GetValue(columnName)
		parts := strings.Split(columnName, ".")
		propName := parts[len(parts)-1]
		values[propName] = value
	}

	return &TelemetryData{
		DeviceUUID: deviceUUID,
		Timestamp:  timestamp,
		Values:     values,
	}, nil
}

func (r *iotdbTelemetryRepository) CreateTimeseries(deviceUUID string, propertyNames []string, dataTypes []client.TSDataType) error {
	session, err := r.client.SessionPool.GetSession()
	if err != nil {
		return err
	}
	defer r.client.SessionPool.PutBack(session)

	for i, propName := range propertyNames {
		path := fmt.Sprintf("root.mm1.device_data.%s.%s", deviceUUID, propName)
		sql := fmt.Sprintf("CREATE TIMESERIES %s WITH DATATYPE=%s, ENCODING=PLAIN, COMPRESSOR=SNAPPY",
			path, TSDataTypeToString(dataTypes[i]))

		status, err := session.ExecuteNonQueryStatement(sql)
		if checkErr := r.client.CheckError(status, err); checkErr != nil {
			return checkErr
		}
	}
	return nil
}
