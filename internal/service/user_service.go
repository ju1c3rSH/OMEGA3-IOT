package service

import (
	"OMEGA3-IOT/internal/db"
	"OMEGA3-IOT/internal/logger"
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/repository"
	"OMEGA3-IOT/internal/utils"
	"context"
	"fmt"
	"io"
	"log"
	"time"
)

type UserService struct {
	mqttSvc                *MQTTService
	userRepo               repository.UserRepository
	instanceRepo           repository.InstanceRepository
	deviceRegistrationRepo repository.DeviceRegistrationRecordRepository
	iotDBClient            *db.IOTDBClient
	loggerService          logger.LoggerInterface
	avatarService          *AvatarService
	dhService              *utils.DHService
	nonceRepo              repository.NonceRepository
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
	loggerService logger.LoggerInterface,
	avatarService *AvatarService,
	dhService *utils.DHService,
	nonceRepo repository.NonceRepository,
) *UserService {
	return &UserService{
		mqttSvc:                mqttSvc,
		userRepo:               userRepo,
		instanceRepo:           instanceRepo,
		deviceRegistrationRepo: deviceRegistrationRepo,
		iotDBClient:            iotDBClient,
		loggerService:          loggerService,
		avatarService:          avatarService,
		dhService:              dhService,
		nonceRepo:              nonceRepo,
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

func (s *UserService) Register(username, commitmentHex string, ip string) (*model.User, error) {
	// Check if user already exists
	_, err := s.userRepo.FindByUsername(username)
	if err == nil {
		s.loggerService.EmitUserLog(logger.NewUserLogEvent("", logger.LogLevelWarning, "Registration failed: username already exists", logger.LogEventSystemError))
		return nil, fmt.Errorf("username already exists")
	}

	// 验证 commitment 格式
	commitment, err := utils.HexToBigInt(commitmentHex)
	if err != nil {
		s.loggerService.EmitUserLog(logger.NewUserLogEvent("", logger.LogLevelWarning, "Registration failed: invalid commitment format", logger.LogEventSystemError))
		return nil, fmt.Errorf("invalid commitment format")
	}

	user := &model.User{
		UserName:     username,
		PasswordHash: utils.BigIntToHex(commitment),
		Type:         1,
		Status:       0,
		Role:         1,
		IP:           ip,
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
		UserUUID:     utils.GenerateUUID().String(),
	}

	if err := s.userRepo.Create(user); err != nil {
		s.loggerService.EmitUserLog(logger.NewUserLogEvent("", logger.LogLevelPanic, "Registration failed: database error", logger.LogEventSystemError))
		return nil, err
	}

	logEvent := logger.NewUserLogEvent(user.UserUUID, logger.LogLevelInfo, "User registered successfully", logger.LogEventUserLogin)
	logEvent.IPAddress = ip
	s.loggerService.EmitUserLog(logEvent)

	return user, nil
}

// Challenge 生成登录挑战（Nonce）
func (s *UserService) Challenge(username string) (*model.ChallengeResponse, error) {
	// 验证用户存在
	_, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// 生成随机 Nonce
	nonce, err := utils.GenerateNonce()
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce")
	}

	nonceHex := utils.BigIntToHex(nonce)

	// 存入 Redis（TTL=60s）
	if err := s.nonceRepo.StoreNonce(context.Background(), username, nonceHex, 0); err != nil {
		return nil, fmt.Errorf("failed to store nonce")
	}

	params := s.dhService.Params()
	return &model.ChallengeResponse{
		Nonce: nonceHex,
		P:     utils.BigIntToHex(params.P),
		G:     utils.BigIntToHex(params.G),
	}, nil
}

// Login 使用 DH 证明值验证登录
func (s *UserService) Login(username, proofHex string, clientIP string) (string, *model.User, error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		logEvent := logger.NewUserLogEvent("", logger.LogLevelWarning, "Login failed: user not found", logger.LogEventUserLogin)
		logEvent.IPAddress = clientIP
		s.loggerService.EmitUserLog(logEvent)
		return "", nil, fmt.Errorf("invalid credentials")
	}

	// 从 Redis 获取并删除 Nonce（一次性使用）
	nonceHex, err := s.nonceRepo.GetAndDeleteNonce(context.Background(), username)
	if err != nil {
		logEvent := logger.NewUserLogEvent(user.UserUUID, logger.LogLevelWarning, "Login failed: nonce expired or already used", logger.LogEventUserLogin)
		logEvent.IPAddress = clientIP
		s.loggerService.EmitUserLog(logEvent)
		return "", nil, fmt.Errorf("challenge expired, please request a new one")
	}

	// 解析 commitment 和 proof
	commitment, err := utils.HexToBigInt(user.PasswordHash)
	if err != nil {
		return "", nil, fmt.Errorf("invalid stored commitment")
	}

	proof, err := utils.HexToBigInt(proofHex)
	if err != nil {
		return "", nil, fmt.Errorf("invalid proof format")
	}

	nonce, err := utils.HexToBigInt(nonceHex)
	if err != nil {
		return "", nil, fmt.Errorf("invalid nonce format")
	}

	// 验证 proof == commitment^nonce mod p
	if !s.dhService.VerifyProof(proof, commitment, nonce) {
		logEvent := logger.NewUserLogEvent(user.UserUUID, logger.LogLevelWarning, "Login failed: invalid proof", logger.LogEventUserLogin)
		logEvent.IPAddress = clientIP
		s.loggerService.EmitUserLog(logEvent)
		return "", nil, fmt.Errorf("invalid credentials")
	}

	jti := utils.GenerateUUID().String()
	token, err := utils.GenerateToken(user.UserName, user.UserUUID, user.Role, jti)
	if err != nil {
		s.loggerService.EmitUserLog(logger.NewUserLogEvent(user.UserUUID, logger.LogLevelPanic, "Login failed: token generation error", logger.LogEventUserLogin))
		return "", nil, err
	}

	updates := map[string]interface{}{
		"last_seen": time.Now().Unix(),
		"ip":        clientIP,
	}
	if err := s.userRepo.UpdateFields(user.UserUUID, updates); err != nil {
		log.Printf("Warning: failed to update user last_seen: %v", err)
	}

	logEvent := logger.NewUserLogEvent(user.UserUUID, logger.LogLevelInfo, "User logged in successfully", logger.LogEventUserLogin)
	logEvent.IPAddress = clientIP
	s.loggerService.EmitUserLog(logEvent)

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

	// Log successful device binding
	logEvent := logger.NewUserLogEvent(userUUID, logger.LogLevelInfo,
		fmt.Sprintf("Device bound successfully: %s (%s)", instance.InstanceUUID, instance.Name),
		logger.LogEventUserDeviceBind)
	logEvent.Metadata = map[string]interface{}{
		"device_uuid": instance.InstanceUUID,
		"device_name": instance.Name,
		"device_type": instance.Type,
		"reg_code":    regCode,
	}
	s.loggerService.EmitUserLog(logEvent)

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

// UpdateProfile updates the user's nickname and/or description.
func (s *UserService) UpdateProfile(userUUID string, req model.UpdateProfileRequest) error {
	fields := make(map[string]interface{})
	if req.Nickname != nil {
		fields["nickname"] = *req.Nickname
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if len(fields) == 0 {
		return fmt.Errorf("no fields to update")
	}
	fields["updated_at"] = time.Now().Unix()
	return s.userRepo.UpdateFields(userUUID, fields)
}

// UpdateAvatar saves an uploaded avatar image and updates the user's avatar field.
func (s *UserService) UpdateAvatar(userUUID string, reader io.Reader) (string, error) {
	avatarURL, err := s.avatarService.SaveAvatar(userUUID, reader)
	if err != nil {
		return "", fmt.Errorf("failed to save avatar: %w", err)
	}

	fields := map[string]interface{}{
		"avatar":     avatarURL,
		"updated_at": time.Now().Unix(),
	}
	if err := s.userRepo.UpdateFields(userUUID, fields); err != nil {
		return "", fmt.Errorf("failed to update avatar in database: %w", err)
	}

	return avatarURL, nil
}

// ResetAvatar generates a default identicon avatar for the user.
func (s *UserService) ResetAvatar(userUUID string) (string, error) {
	avatarURL, err := s.avatarService.GenerateDefaultAvatar(userUUID)
	if err != nil {
		return "", fmt.Errorf("failed to generate default avatar: %w", err)
	}

	fields := map[string]interface{}{
		"avatar":     avatarURL,
		"updated_at": time.Now().Unix(),
	}
	if err := s.userRepo.UpdateFields(userUUID, fields); err != nil {
		return "", fmt.Errorf("failed to update avatar in database: %w", err)
	}

	return avatarURL, nil
}
