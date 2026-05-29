package db

import (
	"OMEGA3-IOT/internal/config"
	"OMEGA3-IOT/internal/model"
	"context"
	"fmt"
	"github.com/apache/iotdb-client-go/client"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
)

var DB *gorm.DB
var RedisClient *redis.Client

func InitDB(config config.Config) {
	MYSQLdsn := config.Database.MYSQLDSN

	var err error

	DB, err = gorm.Open(mysql.Open(MYSQLdsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	if err := DB.AutoMigrate(
		&model.User{},
		&model.Instance{},
		&model.DeviceRegistrationRecord{},
		&model.DeviceShare{},
		&model.DeviceGroup{},
		&model.DeviceGroupRelation{},
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
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
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

	sessionPool := client.NewSessionPool(
		poolConfig,
		config.IoTDB.Pool.MaxConnections,
		int(config.IoTDB.Pool.TimeOut),
		int(config.IoTDB.Pool.TimeOut),
		config.IoTDB.Pool.FetchMetadataAuto,
	)

	session, err := sessionPool.GetSession()
	if err != nil {
		return nil, fmt.Errorf("failed to get a session from pool: %v", err)
	}
	defer sessionPool.PutBack(session)

	if err := session.Open(false, int(config.IoTDB.Pool.TimeOut)); err != nil { // Open 通常不带 timeout 参数
		return nil, fmt.Errorf("failed to open session from pool: %v", err)
	}
	/*
		if err := session.ExecuteNonQueryStatement("CREATE DATABASE root.omega3"); err != nil {
			log.Printf("Warning: Failed to create database (may already exist): %v", err)
		}

	*/
	log.Printf("IoTDB Connected")
	return &IOTDBClient{
		SessionPool: sessionPool,
		Config:      config,
	}, nil
}
