package db

import (
	"OMEGA3-IOT/internal/config"
	"OMEGA3-IOT/internal/model"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/apache/iotdb-client-go/client"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB
var RedisClient *redis.Client

func InitDB(config config.Config) {
	MYSQLdsn := config.Database.MYSQLDSN

	var err error

	DB, err = gorm.Open(mysql.Open(MYSQLdsn), &gorm.Config{
		PrepareStmt:            true,
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatal(err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("failed to get sql.DB from gorm DB: %v", err)
	}
	// Tuned pool: 25 open keeps below MySQL max_connections 151 (even with multiple replicas),
	// 10 idle keeps warm connections for bursty IoT writes.
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(10 * time.Minute)
	sqlDB.SetConnMaxIdleTime(3 * time.Minute)

	if err := DB.AutoMigrate(
		&model.User{},
		&model.Instance{},
		&model.DeviceRegistrationRecord{},
		&model.DeviceShare{},
		&model.DeviceFolder{},
		&model.DeviceFolderItem{},
		&model.UserGroup{},
		&model.GroupMember{},
		&model.GroupPolicy{},
		&model.GroupInvite{},
		&model.GroupDeviceShare{},
		&model.AdminLog{},
	); err != nil {
		log.Fatal(err)
	}

	log.Println("Database Connection inited")
}

func InitRedis(cfg config.Config) {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:            fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password:        cfg.Redis.Password,
		DB:              cfg.Redis.DB,
		PoolSize:        50,
		MinIdleConns:    10,
		DialTimeout:     500 * time.Millisecond,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolTimeout:     4 * time.Second,
		ConnMaxIdleTime: 5 * time.Minute,
		ConnMaxLifetime: 30 * time.Minute,
	})
	if err := RedisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Redis Connection inited")
}
func NewIotDBFromConfig(config config.Config) (*IOTDBClient, error) {
	poolConfig := &client.PoolConfig{
		Host:     config.IoTDB.Host,
		Port:     config.IoTDB.Port,
		UserName: config.IoTDB.UserName,
		Password: config.IoTDB.Password,
	}

	waitTimeoutMs := int(config.IoTDB.Pool.EffectiveWaitTimeoutMs())
	openTimeoutMs := int(config.IoTDB.Pool.EffectiveOpenTimeoutMs())
	readMax := config.IoTDB.Pool.EffectiveReadMaxConnections()
	writeMax := config.IoTDB.Pool.EffectiveWriteMaxConnections()

	newPool := func(maxSize int) SessionPool {
		factory := func() SessionPool {
			sp := client.NewSessionPool(poolConfig, maxSize, openTimeoutMs, waitTimeoutMs, config.IoTDB.Pool.FetchMetadataAuto)
			return &sp
		}
		return NewSessionPoolWrapper(factory)
	}

	readPool := newPool(readMax)
	writePool := newPool(writeMax)

	session, err := writePool.GetSession()
	if err != nil {
		return nil, fmt.Errorf("failed to get a session from pool: %w", err)
	}
	defer writePool.PutBack(session)

	if err := session.Open(false, openTimeoutMs); err != nil {
		return nil, fmt.Errorf("failed to open session from pool: %w", err)
	}
	/*
		if err := session.ExecuteNonQueryStatement("CREATE DATABASE root.omega3"); err != nil {
			log.Printf("Warning: Failed to create database (may already exist): %v", err)
		}

	*/
	log.Printf("IoTDB Connected")
	return &IOTDBClient{
		SessionPool:  writePool,
		readPool:     readPool,
		writePool:    writePool,
		StorageGroup: "",
		Config:       config,
	}, nil
}
