package dao

import (
	"OMEGA3-IOT/internal/db"
	"context"
	"fmt"
	"github.com/apache/iotdb-client-go/client"
	"time"
)

// Telemetry 表示遥测数据的结构。
type Telemetry struct {
	DeviceUUID string
	Timestamp  int64
	Values     map[string]interface{}
}

// QueryTelemetry 表示查询遥测数据的结构。
type QueryTelemetry struct {
	DeviceUUID string
	Timestamp  int64                  // Unix 时间戳
	Values     map[string]interface{} // 键值对
}

// IotdbDao 是一个接口，定义了与 IoTDB 交互的方法。
type IotdbDao interface {
	Save(ctx context.Context, telemetry Telemetry) error
	BatchSave(ctx context.Context, telemetry []*Telemetry) error
	QueryWithRange(ctx context.Context, deviceUUID string, startTime time.Time, endTime time.Time) ([]Telemetry, error)
	QueryLastOne(ctx context.Context, deviceUUID string) (Telemetry, error)
}

// IotDBDAO 是实现了 IotdbDao 接口的结构体。
type IotDBDAO struct {
	client *db.IOTDBClient
}

// NewIotdbDAO 创建一个新的 IotDBDAO 实例。
func NewIotdbDAO(client *db.IOTDBClient) *IotDBDAO {
	return &IotDBDAO{client: client}
}

// Save 将单条遥测数据保存到 IoTDB。
func (d *IotDBDAO) Save(ctx context.Context, telemetry Telemetry) error {
	measurements := make([]string, 0, len(telemetry.Values))
	dataTypes := make([]client.TSDataType, 0, len(telemetry.Values))
	values := make([]interface{}, 0, len(telemetry.Values))

	for k, v := range telemetry.Values {
		measurements = append(measurements, k)
		switch v.(type) {
		case int64:
			dataTypes = append(dataTypes, client.INT64)
		case float64:
			dataTypes = append(dataTypes, client.FLOAT)
		case string:
			dataTypes = append(dataTypes, client.STRING)
		default:
			return fmt.Errorf("unsupported value type: %T", v)
		}
		values = append(values, v)
	}

	_, err := d.client.InsertRecord(telemetry.DeviceUUID, measurements, dataTypes, values, telemetry.Timestamp)
	return err
}

// BatchSave 批量保存遥测数据到 IoTDB。
func (d *IotDBDAO) BatchSave(ctx context.Context, telemetry []*Telemetry) error {
	// 实现批量保存逻辑
	// 这里可以调用 IOTDBClient 的相应方法来批量插入数据

	return nil
}

// QueryWithRange 根据时间范围查询遥测数据。
func (d *IotDBDAO) QueryWithRange(ctx context.Context, deviceUUID string, startTime time.Time, endTime time.Time) ([]Telemetry, error) {
	sql := fmt.Sprintf("SELECT * FROM \"%s\" WHERE time >= %d AND time <= %d", deviceUUID, startTime.UnixNano()/int64(time.Millisecond), endTime.UnixNano()/int64(time.Millisecond))
	var result interface{}
	if err := d.client.ExecuteQuery(sql, &result); err != nil {
		return nil, err
	}
	// 解析查询结果并转换为 []Telemetry
	// 这里需要根据实际的查询结果格式进行解析
	return nil, nil
}

// QueryLastOne 查询指定设备的最新一条遥测数据。
func (d *IotDBDAO) QueryLastOne(ctx context.Context, deviceUUID string) (Telemetry, error) {
	// 实现查询最新一条遥测数据的逻辑
	// 这里可以构建 SQL 语句并调用 IOTDBClient 的 ExecuteQuery 方法
	return Telemetry{}, nil
}
