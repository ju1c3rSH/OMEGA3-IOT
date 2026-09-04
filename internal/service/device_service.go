package service

import (
	"OMEGA3-IOT/internal/db"
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/repository"
	"OMEGA3-IOT/internal/spec"
	"OMEGA3-IOT/internal/utils"
	"context"
	"fmt"
	"github.com/apache/iotdb-client-go/client"
	"gorm.io/gorm"
	"log"
	"strconv"
	"time"
)

type DeviceService struct {
	instanceRepo           repository.InstanceRepository
	deviceRegistrationRepo repository.DeviceRegistrationRecordRepository
	iotDBClient            *db.IOTDBClient
	db                     *gorm.DB
	iotDBRepo              repository.TelemetryRepository
}

func NewDeviceService(db *gorm.DB, iotDBClient *db.IOTDBClient) *DeviceService {
	return &DeviceService{
		instanceRepo:           repository.NewInstanceRepository(db),
		deviceRegistrationRepo: repository.NewDeviceRegistrationRecordRepository(db),
		iotDBClient:            iotDBClient,
		db:                     db,
		iotDBRepo:              repository.NewTelemetryRepository(iotDBClient),
	}
}

func (s *DeviceService) updateDeviceProperties(instance model.Instance, data map[string]model.TypedInstancePropertyItem) error {
	if instance.Properties.Items == nil {
		instance.Properties.Items = make(map[string]*model.TypedInstancePropertyItem)
	}

	// Look up the device type spec for validation
	typeDef, _ := model.GlobalDeviceTypeManager.GetByName(instance.Type)

	for key, value := range data {
		valueCopy := value
		va := valueCopy.Value

		if instance.Properties.Items[key] == nil {
			return fmt.Errorf("failed to update device %s: the key provided was not matched with the properties", instance.InstanceUUID)
		}

		// Validate against spec if device type is known
		if typeDef != nil {
			if propMeta, ok := typeDef.Properties[key]; ok {
				if err := spec.ValidatePropertyValue(propMeta, va.Raw()); err != nil {
					log.Printf("[DeviceService] Property validation failed for device %s, key '%s': %v", instance.InstanceUUID, key, err)
					continue // skip invalid property, don't block other updates
				}
			}
		}

		instance.Properties.Items[key].Value = va
	}
	instance.LastSeen = time.Now().Unix()
	instance.UpdatedAt = time.Now()
	instance.Online = true

	if err := s.instanceRepo.UpdateFields(instance.InstanceUUID, map[string]interface{}{
		"properties": instance.Properties,
		"last_seen":  instance.LastSeen,
		"online":     instance.Online,
		"updated_at": instance.UpdatedAt,
	}); err != nil {
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
	// Defensive clamp: handler already validates max 5000 via binding, but service/repo
	// must also enforce to avoid OOM full scans (IOTDB-749 Avoid select * from root OOM).
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

	instance, err := s.GetDeviceByInstanceUUID(instanceUUID)
	if err != nil {
		return nil, err
	}

	// Sanitize properties: intersect with known instance properties to avoid querying
	// non-existent columns; keep "*" / empty as-is for SELECT * handling in repo.
	var filteredProps []string
	if len(properties) > 0 {
		hasStar := false
		for _, p := range properties {
			if p == "*" {
				hasStar = true
				break
			}
		}
		if hasStar {
			filteredProps = properties
		} else {
			seen := make(map[string]bool, len(properties))
			for _, p := range properties {
				if p == "" || seen[p] {
					continue
				}
				// Only keep properties that exist on this instance/type to avoid
				// useless I/O and to allow server-side projection pushdown.
				if _, ok := instance.Properties.Items[p]; !ok {
					continue
				}
				seen[p] = true
				filteredProps = append(filteredProps, p)
			}
			// If none of the requested props exist, fall back to empty (SELECT *)
			// so IoTDB returns existing columns and caller can see available keys.
			if len(filteredProps) == 0 {
				filteredProps = nil
			}
		}
	}

	telemetryList, err := s.iotDBRepo.QueryTelemetry(instanceUUID, startTimestamp, endTimestamp, limit, offset, filteredProps)
	if err != nil {
		return nil, err
	}

	// Fast lookup for requested property filtering per row
	requestedSet := make(map[string]bool, len(filteredProps))
	for _, p := range filteredProps {
		if p != "*" {
			requestedSet[p] = true
		}
	}
	hasFilter := len(requestedSet) > 0

	historyData := make([]model.DeviceHistoryData, 0, len(telemetryList))
	for _, tel := range telemetryList {
		props := model.Properties{Items: make(map[string]*model.TypedInstancePropertyItem)}
		for meas, val := range tel.Values {
			if hasFilter && !requestedSet[meas] {
				continue
			}
			instanceProp, ok := instance.Properties.Items[meas]
			if !ok {
				// Column exists in IoTDB but not in current instance definition – skip
				continue
			}
			valStr := ""
			if val != nil {
				valStr = fmt.Sprintf("%v", val)
			}
			if valStr == "" {
				propCopy := *instanceProp
				props.Items[meas] = &propCopy
				continue
			}
			it, convertErr := model.NewTypedValueFromOld(valStr, instanceProp.Meta.Format)
			if it == nil {
				log.Printf("[DEBUG] getHistoryData TypedValue is nil - device: %s, measurement: '%s', valStr: '%s', format: '%s', convertErr: %v",
					instanceUUID, meas, valStr, instanceProp.Meta.Format, convertErr)
				propCopy := *instanceProp
				props.Items[meas] = &propCopy
				continue
			}
			propCopy := *instanceProp
			propCopy.Value = *it
			props.Items[meas] = &propCopy
		}
		// Even if props is empty (all values filtered/nil) still preserve timestamp row
		// so pagination is consistent with IoTDB LIMIT/OFFSET.
		historyData = append(historyData, model.DeviceHistoryData{
			Timestamp:  tel.Timestamp,
			Properties: props,
		})
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

	instance, err := model.NewInstanceFromConfig(name, ownerUUID, deviceType, "", remark, utils.GenerateUUID().String(), model.BindByWiFi)
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

	// Only process properties defined in the device type
	typeDef, _ := model.GlobalDeviceTypeManager.GetByName(instance.Type)

	measurements := make([]string, 0, len(instance.Properties.Items))
	dataTypes := make([]client.TSDataType, 0, len(instance.Properties.Items))
	values := make([]interface{}, 0, len(instance.Properties.Items))

	log.Printf("[DeviceService] Processing telemetry for %s: %d properties in instance", instance.InstanceUUID, len(instance.Properties.Items))

	for propKey, propItem := range instance.Properties.Items {
		// Skip properties not in the device type definition
		if typeDef != nil {
			if _, ok := typeDef.Properties[propKey]; !ok {
				log.Printf("[DeviceService] Skipping property '%s' (not in device type)", propKey)
				continue
			}
		}
		measurements = append(measurements, propKey)

		tempValue, err := propItem.Value.ToString()
		if err != nil {
			return fmt.Errorf("failed to parse '%s' with value '%s': %w", propKey, tempValue, err)
		}

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
		case "string", "text", "markdown":
			dataTypes = append(dataTypes, client.STRING)
			values = append(values, tempValue)
		case "time":
			// time values may arrive as float64 from JSON, convert directly from raw value
			var intVal int64
			switch v := propItem.Value.V.(type) {
			case int64:
				intVal = v
			case float64:
				intVal = int64(v)
			case int:
				intVal = int64(v)
			default:
				intVal, err = strconv.ParseInt(tempValue, 10, 64)
				if err != nil {
					return fmt.Errorf("failed to parse time property '%s' with value '%v': %w", propKey, propItem.Value.V, err)
				}
			}
			dataTypes = append(dataTypes, client.INT64)
			values = append(values, intVal)
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

	// Safety check: arrays must have the same length
	if len(measurements) != len(dataTypes) || len(measurements) != len(values) {
		return fmt.Errorf("[DeviceService] array length mismatch for %s: measurements=%d, dataTypes=%d, values=%d",
			instance.InstanceUUID, len(measurements), len(dataTypes), len(values))
	}

	if len(measurements) == 0 {
		log.Printf("[DeviceService] No valid properties to insert for %s", instance.InstanceUUID)
		return nil
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

// GetDeviceActions returns the supported actions for a device instance by its UUID
func (s *DeviceService) GetDeviceActions(instanceUUID string) (map[string]model.ActionMeta, error) {
	instance, err := s.instanceRepo.FindByUUID(instanceUUID)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}

	typeDef, ok := model.GlobalDeviceTypeManager.GetByName(instance.Type)
	if !ok {
		return nil, fmt.Errorf("unknown device type: %s", instance.Type)
	}

	// Ensure nil slices are serialized as [] instead of null
	actions := make(map[string]model.ActionMeta, len(typeDef.Actions))
	for key, action := range typeDef.Actions {
		if action.InputParams == nil {
			action.InputParams = []model.InputParam{}
		}
		actions[key] = action
	}

	return actions, nil
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
func (s *DeviceService) UpdateDeviceProperties(instance model.Instance, data map[string]model.TypedInstancePropertyItem) error {
	return s.updateDeviceProperties(instance, data)
}
