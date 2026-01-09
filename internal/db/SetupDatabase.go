package db

import (
	"OMEGA3-IOT/internal/config"
	"OMEGA3-IOT/internal/model"
	"fmt"
	"github.com/apache/iotdb-client-go/client"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
)

var DB *gorm.DB

func InitDB(config config.Config) {
	MYSQLdsn := config.Database.MYSQLDSN

	var err error

	DB, err = gorm.Open(mysql.Open(MYSQLdsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	if err := DB.AutoMigrate(&model.User{}, &model.Instance{}, &model.DeviceRegistrationRecord{}, &model.DeviceShare{}); err != nil {
		log.Fatal(err)
	}

	log.Println("Database Connection inited")
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
