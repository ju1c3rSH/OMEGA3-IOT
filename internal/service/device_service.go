package service

import (
	"OMEGA3-IOT/internal/db"
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/repository"
	"OMEGA3-IOT/internal/utils"
	"context"
	"fmt"
	"github.com/apache/iotdb-client-go/client"
	"gorm.io/gorm"
	"log"
	"strconv"
	"strings"
	"time"
)

type DeviceService struct {
	instanceRepo           repository.InstanceRepository
	deviceRegistrationRepo repository.DeviceRegistrationRecordRepository
	iotDBClient            *db.IOTDBClient
	db                     *gorm.DB // Kept for transaction support
}

func NewDeviceService(db *gorm.DB, iotDBClient *db.IOTDBClient) *DeviceService {
	return &DeviceService{
		instanceRepo:           repository.NewInstanceRepository(db),
		deviceRegistrationRepo: repository.NewDeviceRegistrationRecordRepository(db),
		iotDBClient:            iotDBClient,
		db:                     db,
	}
}

func (s *DeviceService) updateDeviceProperties(instance model.Instance, data map[string]model.PropertyItem) error {
	if instance.Properties.Items == nil {
		instance.Properties.Items = make(map[string]*model.PropertyItem)
	}
	for key, value := range data {
		valueCopy := value
		va := valueCopy.Value

		if instance.Properties.Items[key] == nil {
			return fmt.Errorf("failed to update device %s: the key provided was not matched with the properties", instance.InstanceUUID)
		}

		instance.Properties.Items[key].Value = va
	}
	instance.LastSeen = time.Now().Unix()
	instance.UpdatedAt = time.Now()
	instance.Online = true

	if err := s.instanceRepo.Update(&instance); err != nil {
		return fmt.Errorf("failed to save updated instance %s to database: %w", instance.InstanceUUID, err)
	}

	log.Printf("Database record for device %s updated with new properties.", instance.InstanceUUID)

	err := s.ProcessDeviceTelemetryFromInstance(&instance)
	if err != nil {
		log.Printf("[DeviceService] Failed to process device telemetry from instance %s to database: %v", instance.InstanceUUID, err)
		return err
	}

	return nil
}

func (s *DeviceService) GetDeviceHistoryData(instanceUUID string, startTimestamp int64, endTimestamp int64, limit int, offset int, properties []string) (*[]model.DeviceHistoryData, error) {
	session, err := s.iotDBClient.SessionPool.GetSession()
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	defer s.iotDBClient.SessionPool.PutBack(session)

	instance, err := s.GetDeviceByInstanceUUID(instanceUUID)
	if err != nil {
		return nil, err
	}

	devicePath := utils.ConvertHyphenIntoDash(fmt.Sprintf("root.mm1.device_data.%s", instanceUUID))
	if properties == nil || len(properties) == 0 {
		properties = make([]string, 0, len(instance.Properties.Items))
		for propName := range instance.Properties.Items {
			properties = append(properties, propName)
		}
	}
	var propertyPaths []string
	for _, prop := range properties {
		propertyPaths = append(propertyPaths, fmt.Sprintf("%s.%s", devicePath, prop))
	}
	var tc utils.TimeConverter
	startISO := tc.SecToISO(startTimestamp)
	endIso := tc.SecToISO(endTimestamp)

	sql := fmt.Sprintf("SELECT %s FROM %s WHERE time >= %s AND time <= %s ORDER BY time DESC LIMIT %d OFFSET %d",
		strings.Join(properties, ", "), devicePath, startISO, endIso, limit, offset)

	sessionDataSet, err := session.ExecuteQueryStatement(sql, &s.iotDBClient.Config.IoTDB.QueryTimeoutMs)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer sessionDataSet.Close()
	var historyData []model.DeviceHistoryData

	for {
		hasNext, err := sessionDataSet.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate result set: %w", err)
		}
		if !hasNext {
			break
		}
		row, err := sessionDataSet.GetRowRecord()
		if err != nil {
			return nil, fmt.Errorf("failed to get row record: %w", err)
		}
		timestamp := row.GetTimestamp()
		currentTsSec := timestamp / 1000
		var properties model.Properties
		properties.Items = make(map[string]*model.PropertyItem)

		columnCount := sessionDataSet.GetColumnCount()
		for i := 0; i < columnCount; i++ {
			columnName := sessionDataSet.GetColumnName(i)
			val := sessionDataSet.GetValue(columnName)

			parts := strings.Split(columnName, ".")
			propName := parts[len(parts)-1]
			valStr := ""
			if val != nil {
				valStr = fmt.Sprintf("%v", val)
			}

			meta, ok := instance.Properties.Items[propName]
			if !ok {
				meta = &model.PropertyItem{
					Value: valStr,
					Meta: model.PropertyMeta{
						Description: "Unknown Property",
						Format:      "string",
					},
				}
			} else {
				meta.Value = valStr
			}
			properties.Items[propName] = meta
		}

		record := model.DeviceHistoryData{
			Timestamp:  currentTsSec,
			Properties: properties,
		}
		historyData = append(historyData, record)
	}

	return &historyData, nil
}

func (s *DeviceService) IsInstanceExists(instanceUUID string) (bool, error) {
	return s.instanceRepo.Exists(instanceUUID)
}

func (s *DeviceService) GetDeviceByInstanceUUID(instanceUUID string) (*model.Instance, error) {
	return s.instanceRepo.FindByUUID(instanceUUID)
}

func (s *DeviceService) RegisterDeviceAnonymously(deviceTypeID int, verifyCode string) (*model.DeviceRegistrationRecord, error) {
	_, valid := model.GlobalDeviceTypeManager.GetById(deviceTypeID)
	if !valid {
		return nil, fmt.Errorf("invalid device type ID: %d", deviceTypeID)
	}

	hashedVerifyCode := utils.HashVerifyCode(verifyCode)
	record, err := model.NewRegistrationRecord(deviceTypeID, hashedVerifyCode)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = ctx

	if err := s.deviceRegistrationRepo.Create(record); err != nil {
		return nil, fmt.Errorf("failed to create registration record: %w", err)
	}

	return record, nil
}

// AddDevice - Deprecated
func (s *DeviceService) AddDevice(name string, deviceTypeID int, remark string, ownerUUID string) (*model.Instance, error) {
	deviceType, valid := model.GlobalDeviceTypeManager.GetById(deviceTypeID)
	if !valid {
		return nil, fmt.Errorf("invalid device type ID: %d", deviceTypeID)
	}

	instance, err := model.NewInstanceFromConfig(name, ownerUUID, deviceType, "", remark, utils.GenerateUUID().String())
	if err != nil {
		return nil, fmt.Errorf("failed to create device instance: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = ctx

	if err := s.instanceRepo.Create(instance); err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	return instance, nil
}

func (s *DeviceService) PropertiesToTelemetryData(instance *model.Instance) error {
	// This method seems to be a preparation method, not actually saving data
	// Kept for backward compatibility
	return nil
}

func (s *DeviceService) ProcessDeviceTelemetryFromInstance(instance *model.Instance) error {
	historicalDevicePath := utils.ConvertHyphenIntoDash(fmt.Sprintf("root.mm1.device_data.%s", instance.InstanceUUID))

	measurements := make([]string, 0, len(instance.Properties.Items))
	dataTypes := make([]client.TSDataType, 0, len(instance.Properties.Items))
	values := make([]interface{}, 0, len(instance.Properties.Items))

	for propKey, propItem := range instance.Properties.Items {
		measurements = append(measurements, propKey)

		tempValue := propItem.Value
		switch propItem.Meta.Format {
		case "int", "integer":
			intVal, err := strconv.ParseInt(tempValue, 10, 32)
			if err != nil {
				return fmt.Errorf("failed to parse int property '%s' with value '%s': %w", propKey, tempValue, err)
			}
			dataTypes = append(dataTypes, client.INT32)
			values = append(values, int32(intVal))
		case "long":
			intVal, err := strconv.ParseInt(tempValue, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse long property '%s' with value '%s': %w", propKey, tempValue, err)
			}
			dataTypes = append(dataTypes, client.INT64)
			values = append(values, intVal)
		case "float", "single":
			floatVal, err := strconv.ParseFloat(tempValue, 32)
			if err != nil {
				return fmt.Errorf("failed to parse float property '%s' with value '%s': %w", propKey, tempValue, err)
			}
			dataTypes = append(dataTypes, client.FLOAT)
			values = append(values, float32(floatVal))
		case "double":
			floatVal, err := strconv.ParseFloat(tempValue, 64)
			if err != nil {
				return fmt.Errorf("failed to parse double property '%s' with value '%s': %w", propKey, tempValue, err)
			}
			dataTypes = append(dataTypes, client.DOUBLE)
			values = append(values, floatVal)
		case "string", "text":
			dataTypes = append(dataTypes, client.STRING)
			values = append(values, tempValue)
		case "boolean":
			boolVal, err := strconv.ParseBool(tempValue)
			if err != nil {
				return fmt.Errorf("failed to parse boolean property '%s' with value '%s': %w", propKey, tempValue, err)
			}
			dataTypes = append(dataTypes, client.BOOLEAN)
			values = append(values, boolVal)
		default:
			log.Printf("Warning: Unknown format '%s' for property '%s', treating as string.", propItem.Meta.Format, propKey)
			dataTypes = append(dataTypes, client.STRING)
			values = append(values, tempValue)
		}
	}

	timestamp := time.Now().UnixNano() / int64(time.Millisecond)

	if err := s.iotDBClient.InsertRecordTyped(historicalDevicePath, measurements, dataTypes, values, timestamp); err != nil {
		return fmt.Errorf("[DeviceService] failed to save telemetry from instance %s to IoTDB: %w", instance.InstanceUUID, err)
	}

	return nil
}

// UpdateInstanceProperties updates device properties and saves to database
func (s *DeviceService) UpdateInstanceProperties(instanceUUID string, properties model.Properties) error {
	instance, err := s.instanceRepo.FindByUUID(instanceUUID)
	if err != nil {
		return err
	}

	instance.Properties = properties
	instance.UpdatedAt = time.Now()
	instance.LastSeen = time.Now().Unix()

	return s.instanceRepo.Update(instance)
}

// UpdateInstanceOnlineStatus updates device online status
func (s *DeviceService) UpdateInstanceOnlineStatus(instanceUUID string, online bool) error {
	return s.instanceRepo.UpdateOnlineStatus(instanceUUID, online, time.Now().Unix())
}

// GetDevicesByOwner returns all devices owned by a user
func (s *DeviceService) GetDevicesByOwner(ownerUUID string) ([]model.Instance, error) {
	return s.instanceRepo.FindByOwnerUUID(ownerUUID)
}

// GetDeviceByUUIDAndVerifyHash retrieves a device by UUID and verifies the hash
func (s *DeviceService) GetDeviceByUUIDAndVerifyHash(instanceUUID string, verifyHash string) (*model.Instance, error) {
	instance, err := s.instanceRepo.FindByUUID(instanceUUID)
	if err != nil {
		return nil, err
	}
	if instance.VerifyHash != verifyHash {
		return nil, fmt.Errorf("invalid verify hash")
	}
	return instance, nil
}

// UpdateDeviceProperties updates device properties (used by MQTT service)
func (s *DeviceService) UpdateDeviceProperties(instance model.Instance, data map[string]model.PropertyItem) error {
	return s.updateDeviceProperties(instance, data)
}
