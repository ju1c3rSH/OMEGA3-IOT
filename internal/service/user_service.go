package service

import (
	"OMEGA3-IOT/internal/db"
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/repository"
	"OMEGA3-IOT/internal/utils"
	"context"
	"fmt"
	"log"
	"time"
)

type UserService struct {
	mqttSvc                *MQTTService
	userRepo               repository.UserRepository
	instanceRepo           repository.InstanceRepository
	deviceRegistrationRepo repository.DeviceRegistrationRecordRepository
	iotDBClient            *db.IOTDBClient
}

type GetUserAllDevicesResponse struct {
	InstanceCount int              `json:"instance_count"`
	Instances     []model.Instance `json:"instances"`
}

type GetUserAllAccessibleDevicesResponse struct {
	InstanceCount int              `json:"instance_count"`
	Instances     []model.Instance `json:"instances"`
}

func NewUserService(
	mqttSvc *MQTTService,
	userRepo repository.UserRepository,
	instanceRepo repository.InstanceRepository,
	deviceRegistrationRepo repository.DeviceRegistrationRecordRepository,
	iotDBClient *db.IOTDBClient,
) *UserService {
	return &UserService{
		mqttSvc:                mqttSvc,
		userRepo:               userRepo,
		instanceRepo:           instanceRepo,
		deviceRegistrationRepo: deviceRegistrationRepo,
		iotDBClient:            iotDBClient,
	}
}

func (s *UserService) GetUserAllDevices(userUUID string) (*GetUserAllDevicesResponse, error) {
	instances, err := s.instanceRepo.FindByOwnerUUID(userUUID)
	if err != nil {
		return nil, err
	}
	count := len(instances)

	response := &GetUserAllDevicesResponse{
		InstanceCount: count,
		Instances:     instances,
	}
	return response, nil
}

func (s *UserService) Register(username, password string, ip string) (*model.User, error) {
	// Check if user already exists
	_, err := s.userRepo.FindByUsername(username)
	if err == nil {
		return nil, fmt.Errorf("username already exists")
	}

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		UserName:     username,
		PasswordHash: hashedPassword,
		Type:         1,
		Status:       0,
		Role:         1,
		IP:           ip,
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
		UserUUID:     utils.GenerateUUID().String(),
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) Login(username, password string, clientIP string) (string, *model.User, error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return "", nil, fmt.Errorf("invalid credentials")
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return "", nil, fmt.Errorf("invalid credentials")
	}

	token, err := utils.GenerateToken(user.UserName, user.UserUUID, user.Role)
	if err != nil {
		return "", nil, err
	}

	// Update last seen and IP
	updates := map[string]interface{}{
		"last_seen": time.Now().Unix(),
		"ip":        clientIP,
	}
	if err := s.userRepo.UpdateFields(user.UserUUID, updates); err != nil {
		log.Printf("Warning: failed to update user last_seen: %v", err)
	}

	return token, user, nil
}

func (s *UserService) GetUserInfoByID(userID uint) (*model.User, error) {
	return s.userRepo.FindByID(userID)
}

func (s *UserService) GetUserInfoByUUID(userUUID string) (*model.User, error) {
	return s.userRepo.FindByUUID(userUUID)
}

func (s *UserService) BindDeviceByRegCode(userUUID string, regCode string, deviceNick string, deviceRemark string) (*model.Instance, error) {
	// Find the registration record
	record, err := s.deviceRegistrationRepo.FindByRegCode(regCode)
	if err != nil {
		return nil, fmt.Errorf("invalid registration code")
	}

	// Check if expired or already bound
	if record.ExpiresAt < time.Now().Unix() || record.IsBound {
		return nil, fmt.Errorf("registration code expired or already used")
	}

	deviceType, exists := model.GlobalDeviceTypeManager.GetById(record.DeviceTypeID)
	if !exists {
		// Delete invalid record
		s.deviceRegistrationRepo.DeleteByDeviceUUID(record.DeviceUUID)
		return nil, fmt.Errorf("associated device type (ID: %d) not found", record.DeviceTypeID)
	}

	name := deviceNick
	if name == "" {
		name = "Unnamed Device"
	}

	instance, err := model.NewInstanceFromConfig(name, userUUID, deviceType, record.VerifyHash, deviceRemark, record.DeviceUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to create device instance: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create instance with timeout context
	// Note: Repository doesn't support context directly, this is a design decision
	// If needed, we can add context support to repository methods
	_ = ctx // Keep for future use if repository adds context support

	if err := s.instanceRepo.Create(instance); err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	// Create timeseries in IoTDB
	if err := s.createDeviceTimeseriesInIoTDB(*instance); err != nil {
		log.Printf("[UserService] failed to create device timeseries in IoTDB: %v", err)
	}

	// Mark as bound
	if err := s.deviceRegistrationRepo.MarkAsBound(record.DeviceUUID); err != nil {
		log.Printf("Warning: Failed to mark registration record %s as bound: %v", record.DeviceUUID, err)
	}

	// Send action to device
	actionPayload := model.NewEnablePropertiesUploadAction(30)
	if err := s.mqttSvc.PublishActionToDevice(instance.InstanceUUID, "enable_properties_upload", *actionPayload); err != nil {
		log.Printf("Warning: Failed to send enable_properties_upload action to device %s: %v", instance.InstanceUUID, err)
	}

	return instance, nil
}

func (s *UserService) createDeviceTimeseriesInIoTDB(instance model.Instance) error {
	session, err := s.iotDBClient.SessionPool.GetSession()
	if err != nil {
		return fmt.Errorf("[UserService] failed to get session: %w", err)
	}
	defer s.iotDBClient.SessionPool.PutBack(session)

	for _, meta := range instance.Properties.Items {
		dataType, encoding, compression := s.iotDBClient.MapConvertToIotDBType(meta.Meta)

		historicalPath := fmt.Sprintf("root.mm1.device_data.%s.%s", instance.InstanceUUID, meta.Meta.Description)

		sql := fmt.Sprintf("CREATE TIMESERIES %s WITH DATATYPE=%d, ENCODING=%d, COMPRESSION=%d",
			historicalPath, dataType, encoding, compression)
		status, err := s.iotDBClient.ExecuteNonQuery(sql)
		if checkErr := s.iotDBClient.CheckError(status, err); checkErr != nil {
			return fmt.Errorf("[UserService] failed to create timeseries %s: %w", meta.Meta.Description, checkErr)
		}
	}
	return nil
}
