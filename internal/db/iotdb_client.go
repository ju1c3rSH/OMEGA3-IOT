package db

import (
	"OMEGA3-IOT/internal/config"
	"OMEGA3-IOT/internal/model"
	"fmt"
	"github.com/apache/iotdb-client-go/client"
	"github.com/apache/iotdb-client-go/common"
	"log"
	"strings"
)

type IOTDBClient struct {
	//session      *client.Session
	SessionPool SessionPool
	readPool    SessionPool
	writePool   SessionPool
	StorageGroup string
	//TODO 尚且需要确定SG是否唯一？
	Config config.Config
}

func (i *IOTDBClient) ReadPool() SessionPool {
	if i.readPool != nil {
		return i.readPool
	}
	return i.SessionPool
}

func (i *IOTDBClient) WritePool() SessionPool {
	if i.writePool != nil {
		return i.writePool
	}
	return i.SessionPool
}

func (i *IOTDBClient) Close() {
	i.WritePool().Close()
	if rp := i.readPool; rp != nil && rp != i.writePool {
		rp.Close()
	}
}

func (i *IOTDBClient) InsertRecord(deviceId string, measurements []string, dataTypes []client.TSDataType, values []interface{}, timestamp int64) (r *common.TSStatus, err error) {
	session, err := i.WritePool().GetSession()
	if err != nil {
		return nil, fmt.Errorf("failed to get session from pool: %w", err)
	}
	defer i.WritePool().PutBack(session)
	return session.InsertRecord(deviceId, measurements, dataTypes, values, timestamp)
}

func (i *IOTDBClient) ExecuteQuery(sql string, result *interface{}) error {
	session, err := i.ReadPool().GetSession()
	if err != nil {
		return fmt.Errorf("failed to get session from pool: %w", err)
	}
	defer i.ReadPool().PutBack(session)
	queryDataSet, err := session.ExecuteQueryStatement(sql, &i.Config.IoTDB.QueryTimeoutMs)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer queryDataSet.Close()
	*result = queryDataSet
	return nil
}
func (i *IOTDBClient) ExecuteNonQuery(sql string) (*common.TSStatus, error) {
	session, err := i.WritePool().GetSession()
	if err != nil {
		return nil, fmt.Errorf("failed to get session from pool: %w", err)
	}
	defer i.WritePool().PutBack(session)

	return session.ExecuteNonQueryStatement(sql)
}

// InsertRecordTyped inserts a record using SQL INSERT to bypass client library schema cache issues.
func (i *IOTDBClient) InsertRecordTyped(deviceId string, measurements []string, dataTypes []client.TSDataType, values []interface{}, timestamp int64) error {
	session, err := i.WritePool().GetSession()
	if err != nil {
		return fmt.Errorf("failed to get session from pool: %w", err)
	}
	defer i.WritePool().PutBack(session)

	// Build SQL: INSERT INTO root.mm1.device_data.{uuid}(timestamp, m1, m2, ...) VALUES(ts, v1, v2, ...)
	measurementStr := strings.Join(measurements, ", ")
	var valueStrs []string
	for idx, v := range values {
		switch dataTypes[idx] {
		case client.INT32:
			valueStrs = append(valueStrs, fmt.Sprintf("%d", v.(int32)))
		case client.INT64:
			valueStrs = append(valueStrs, fmt.Sprintf("%d", v.(int64)))
		case client.FLOAT:
			valueStrs = append(valueStrs, fmt.Sprintf("%f", v.(float32)))
		case client.DOUBLE:
			valueStrs = append(valueStrs, fmt.Sprintf("%f", v.(float64)))
		case client.BOOLEAN:
			valueStrs = append(valueStrs, fmt.Sprintf("%v", v.(bool)))
		default:
			// STRING — escape single quotes
			s := fmt.Sprintf("%v", v)
			s = strings.ReplaceAll(s, "'", "''")
			if len(s) > 32768 {
				s = s[:32768]
			}
			valueStrs = append(valueStrs, fmt.Sprintf("'%s'", s))
		}
	}
	valueStr := strings.Join(valueStrs, ", ")

	sql := fmt.Sprintf("INSERT INTO %s(timestamp, %s) VALUES(%d, %s)", deviceId, measurementStr, timestamp, valueStr)
	status, err := session.ExecuteNonQueryStatement(sql)
	return i.CheckError(status, err)
}

func (i *IOTDBClient) InitializeSchema() error {
	session, err := i.WritePool().GetSession()
	if err != nil {
		return fmt.Errorf("failed to get session from pool: %w", err)
	}
	defer i.WritePool().PutBack(session)

	storageGroup := "root.mm1"
	//latestStorageGroup := "root.mm1_latest"
	//实际上不需要latest，latest在MySQL里
	//TODO SG要拓展性
	if err := i.setStorageGroup(storageGroup); err != nil {
		return err
	}
	//i.setStorageGroup(latestStorageGroup)
	return nil
}
func (i *IOTDBClient) MapConvertToIotDBType(meta model.PropertyMeta) (dataType client.TSDataType, encoding client.TSEncoding, compression client.TSCompressionType) {
	switch meta.Format {
	case "int", "integer":
		dataType = client.INT32
		encoding = client.RLE
	case "long", "time":
		dataType = client.INT64
		encoding = client.RLE
	case "float":
		dataType = client.FLOAT
		encoding = client.GORILLA
	case "double":
		dataType = client.DOUBLE
		encoding = client.GORILLA
	case "string", "text", "markdown":
		dataType = client.STRING
		encoding = client.PLAIN
	case "boolean":
		dataType = client.BOOLEAN
		encoding = client.RLE
	default:
		// 默认使用string
		dataType = client.STRING
		encoding = client.PLAIN
	}
	compression = client.SNAPPY //默认压缩
	return

}

// checkError: 返回nil则是成功
func (i *IOTDBClient) CheckError(status *common.TSStatus, err error) error {
	if err != nil {
		return fmt.Errorf("[IOTDB CLIENT] Error: failed to get session from pool: %w", err)
	}
	if status != nil {
		if status.GetCode() == client.MultipleError ||
			status.GetMessage() != "" && strings.Contains(status.GetMessage(), "already exists") {
			log.Printf("[IOTDB CLIENT] Info: %s ", status.GetMessage())
			return nil
		}
		if verifyErr := client.VerifySuccess(status); verifyErr != nil {
			return fmt.Errorf("[IOTDB CLIENT] Notice: IoTDB operation failed: %w", verifyErr)
		}
	}
	return nil
}

func (i *IOTDBClient) setStorageGroup(storageGroup string) error {
	session, err := i.WritePool().GetSession()
	if err != nil {
		log.Printf("failed to get session from pool: %v", err)
		return fmt.Errorf("failed to get session from pool: %w", err)
	}
	defer i.WritePool().PutBack(session)

	i.StorageGroup = storageGroup
	status, err := session.SetStorageGroup(storageGroup)
	if checkErr := i.CheckError(status, err); checkErr != nil {
		log.Printf("[IOTDB CLIENT] (不需要理睬): %s ", checkErr)
	}
	return nil
}
