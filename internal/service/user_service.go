package service

import (
	"OMEGA3-IOT/internal/db"
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/utils"
	"context"
	"fmt"
	"gorm.io/gorm"
	"log"
	"time"
)

type UserService struct {
	mqttSvc     *MQTTService
	mysqlDB     *gorm.DB
	iotDBClient *db.IOTDBClient
}
type GetUserAllDevicesResponse struct {
	InstanceCount int              `json:"instance_count"`
	Instances     []model.Instance `json:"instances"`
}

func NewUserService(mqttSvc *MQTTService, mysqlDB *gorm.DB, iotDBClient *db.IOTDBClient) *UserService {
	return &UserService{
		mqttSvc:     mqttSvc,
		mysqlDB:     mysqlDB,
		iotDBClient: iotDBClient,
	}
}
func (s *UserService) GetUserAllDevices(userUUID string) (*GetUserAllDevicesResponse, error) {
	var instances []model.Instance

	if err := s.mysqlDB.Where("owner_uuid = ?", userUUID).Find(&instances).Error; err != nil {
		return nil, fmt.Errorf("user %s not found, detals: %s", userUUID, err)
	}
	count := len(instances)

	response := &GetUserAllDevicesResponse{
		InstanceCount: count,
		Instances:     instances,
	}
	return response, nil
}
func (s *UserService) Register(username, password string, ip string) (*model.User, error) {
	var existingUser model.User
	if err := s.mysqlDB.Where("user_name = ?", username).First(&existingUser).Error; err == nil {
		return nil, gorm.ErrDuplicatedKey
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

	if err := s.mysqlDB.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) Login(username, password string, clientIP string) (string, *model.User, error) {
	var user model.User
	if err := s.mysqlDB.Where("user_name = ?", username).First(&user).Error; err != nil {
		return "", nil, gorm.ErrRecordNotFound
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return "", nil, gorm.ErrRecordNotFound
	}

	token, err := utils.GenerateToken(user.UserName, user.UserUUID, user.Role)
	if err != nil {
		return "", nil, err
	}

	s.mysqlDB.Model(&user).Updates(map[string]interface{}{
		"last_seen": time.Now().Unix(),
		"ip":        clientIP,
	})

	return token, &user, nil
}

func (s *UserService) GetUserInfoByID(userID uint) (*model.User, error) {
	var user model.User
	if err := s.mysqlDB.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) BindDeviceByRegCode(userUUID string, regCode string, deviceNick string, deviceRemark string) (*model.Instance, error) {
	var record model.DeviceRegistrationRecord
	if err := s.mysqlDB.Where("reg_code = ? AND expires_at > ? AND is_bound = ?", regCode, time.Now().Unix(), false).First(&record).Error; err != nil {
		fmt.Errorf("invalid or expired or used registration code: %w ,called up by user %w", err, userUUID)
		return nil, fmt.Errorf("invalid or expired or used registration code: %w ,called up by user %w", err, userUUID) // 使用 wrap 保留原始错误信息
	}
	deviceType, exists := model.GlobalDeviceTypeManager.GetById(record.DeviceTypeID)
	if !exists {
		// 理论上初始化时加载了类型，这里找不到可能意味着数据不一致或类型被移除
		// 记录日志或返回特定错误
		// 删除无效的记录
		s.mysqlDB.Delete(&record)
		return nil, fmt.Errorf("associated device type (ID: %d) not found", record.DeviceTypeID)
	}
	name := deviceNick
	if name == "" {
		name = "Unnamed Device" //Nick为空时
	}

	instance, err := model.NewInstanceFromConfig(name, userUUID, deviceType, record.VerifyHash, deviceRemark, record.DeviceUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to create device instance: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.mysqlDB.WithContext(ctx).Create(instance).Error; err != nil {
		// 判断是否为唯一键冲突（如设备名重复）
		if err != nil && len(err.Error()) > 0 && (err.Error() == "UNIQUE constraint failed" || err.Error() == "duplicate key value violates unique constraint") {
			return nil, gorm.ErrDuplicatedKey
		}
		return nil, err
	}

	if err := s.createDeviceTimeseriesInIoTDB(*instance); err != nil {
		log.Printf("[UserService] failed to create device timeseries in IoTDB: %v", err)
	}

	if err := s.mysqlDB.Model(&model.DeviceRegistrationRecord{}).Where("device_uuid = ?", record.DeviceUUID).Update("is_bound", true).Error; err != nil {
		// 如果更新 IsBound 失败，记录警告日志。
		// Instance 已创建成功，但记录状态不一致。
		// 这种情况应被监控，因为它可能导致后续问题（例如，如果清理任务依赖 IsBound）。
		// log.Printf("Warning: Failed to mark registration record %s as bound: %v", record.DeviceUUID, err)
		// 为简化，此处不添加日志代码，但生产环境建议添加。
		_ = err // 显式忽略错误，或按需处理（如返回特定错误）
		// 注意：如果更新 IsBound 失败是一个严重问题，
		// 你可能需要考虑回滚 Instance 的创建（这通常很复杂，需要事务或补偿操作）。
		// 但在大多数情况下，标记失败比删除已成功创建的实例更安全。
	}
	/*
		actionPayload := map[string]interface{}{
			"command":   "enable_properties_upload",
			"timestamp": time.Now().Unix(),
			"params": map[string]interface{}{
				"interval_sec": 30,
			},
		}
	*/

	actionPayload := model.NewEnablePropertiesUploadAction(30)
	//TODO 将user_service通过事件驱动等的方式和mqtt_service解耦。
	if err := s.mqttSvc.PublishActionToDevice(instance.InstanceUUID, "enable_properties_upload", *actionPayload); err != nil {
		log.Printf("Warning: Failed to send enable_properties_upload action to device %s: %v", instance.InstanceUUID, err)
		//TODO 这里可以加上一个 Retry Pool..
	}
	return instance, nil
	//should return a instance obj
}

func (s *UserService) createDeviceTimeseriesInIoTDB(instance model.Instance) error {
	session, err := s.iotDBClient.SessionPool.GetSession()
	if err != nil {
		return fmt.Errorf("[DeviceService] failed to get session: %w", err)
	}
	defer s.iotDBClient.SessionPool.PutBack(session)

	for _, meta := range instance.Properties.Items {
		dataType, encoding, compression := s.iotDBClient.MapConvertToIotDBType(meta.Meta)

		historicalPath := fmt.Sprintf("root.mm1.device_data.%s.%s", instance.InstanceUUID, meta.Meta.Description)

		sql := fmt.Sprintf("CREATE TIMESERIES %s WITH DATATYPE=%s, ENCODING=%s, COMPRESSION=%s", historicalPath, dataType, encoding, compression)
		status, err := s.iotDBClient.ExecuteNonQuery(sql)
		if checkErr := s.iotDBClient.CheckError(status, err); err != nil {
			return fmt.Errorf("[UserService] failed to create timeseries %s: %w,for %s", meta.Meta.Description, err, checkErr)
		}

	}
	return nil
}
