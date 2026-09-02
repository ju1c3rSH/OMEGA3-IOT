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
	QueryTelemetry(deviceUUID string, startTime, endTime int64, limit int, offset int, properties []string) ([]TelemetryData, error)
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

func (r *iotdbTelemetryRepository) QueryTelemetry(deviceUUID string, startTime, endTime int64, limit int, offset int, properties []string) ([]TelemetryData, error) {
	// Clamp pagination to prevent OOM / abuse; handler already validates limit <=5000
	// but repo enforces again defensively (IoTDB full scan without LIMIT can OOM,
	// see https://github.com/apache/incubator-iotdb/blob/master/RELEASE_NOTES.md#bug-fixes IOTDB-749 and
	// https://iotdb.apache.org/UserGuide/latest-Table/SQL-Manual/Limit-Offset-Clause.html).
	if limit <= 0 {
		limit = 1000
	}
	if limit > 5000 {
		limit = 5000
	}
	if offset < 0 {
		offset = 0
	}
	if offset > 10000 {
		offset = 10000
	}

	// Build projection list: SELECT * vs SELECT time, <props>
	// Avoid SELECT *: GORM/IoTDB best practice is to select only required columns to
	// reduce I/O, memory and network (see https://gorm.io/docs/advanced_query.html Smart Select Fields
	// and https://stackoverflow.com/questions/3180375/select-vs-select-column).
	selectClause := "*"
	if len(properties) > 0 {
		hasStar := false
		for _, p := range properties {
			if p == "*" {
				hasStar = true
				break
			}
		}
		if !hasStar {
			// Whitelist identifier validation to prevent injection; IoTDB identifiers must be [a-zA-Z_][a-zA-Z0-9_]*
			seen := make(map[string]bool, len(properties))
			validProps := make([]string, 0, len(properties))
			for _, p := range properties {
				if p == "" {
					continue
				}
				if seen[p] {
					continue
				}
				valid := true
				for idx, c := range p {
					if idx == 0 {
						if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_') {
							valid = false
							break
						}
					} else {
						if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
							valid = false
							break
						}
					}
				}
				if !valid {
					continue
				}
				seen[p] = true
				validProps = append(validProps, p)
			}
			if len(validProps) > 0 {
				// Include time explicitly for clarity; IoTDB always returns time but
				// projecting time+props avoids SELECT * full-scan overhead and enables
				// server-side LIMIT/OFFSET pushdown per
				// https://iotdb.apache.org/UserGuide/V0.13.x/Query-Data/Pagination.html
				//   select status,temperature from root.ln.wf01.wt01 where time > ... limit 2 offset 3
				selectClause = "time, " + strings.Join(validProps, ", ")
			}
		}
	}

	devicePath := utils.ConvertHyphenIntoDash(fmt.Sprintf("root.mm1.device_data.%s", deviceUUID))
	// Ensure time is in ms: insert path uses UnixMilli (ms) via time.Now().UnixNano()/1e6.
	// Callers pass Unix seconds (handler validates 30d = 15552000s), so convert to ms.
	// If caller already passes ms (>1e12), keep as-is to avoid double scaling.
	toMs := func(ts int64) int64 {
		if ts > 1_000_000_000_000 {
			return ts
		}
		return ts * 1000
	}
	startMs := toMs(startTime)
	endMs := toMs(endTime)
	if startMs > endMs {
		startMs, endMs = endMs, startMs
	}
	sql := fmt.Sprintf("SELECT %s FROM %s WHERE time >= %d AND time <= %d ORDER BY time DESC LIMIT %d OFFSET %d",
		selectClause, devicePath, startMs, endMs, limit, offset)

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

	var results []TelemetryData
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
		propNames := make([]string, 0, columnCount)
		for i := 0; i < columnCount; i++ {
			columnName := dataSet.GetColumnName(i)
			// IoTDB returns fully qualified names like root.mm1.device_data.x.temp ;
			// also returns "Time" column – skip it, time is already in record.GetTimestamp().
			if strings.EqualFold(columnName, "Time") || strings.EqualFold(columnName, "time") {
				continue
			}
			// For ALIGN BY DEVICE queries there is a "Device" column – skip.
			if strings.EqualFold(columnName, "Device") {
				continue
			}
			parts := strings.Split(columnName, ".")
			propName := parts[len(parts)-1]
			if propName == "" {
				continue
			}
			// Skip internal time‐ish columns if any
			if strings.EqualFold(propName, "time") {
				continue
			}
			value := dataSet.GetValue(columnName)
			propNames = append(propNames, propName)
			values[propName] = value
		}

		results = append(results, TelemetryData{
			DeviceUUID:   deviceUUID,
			Timestamp:    timestamp,
			Values:       values,
			Measurements: propNames,
		})
	}

	return results, nil
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
