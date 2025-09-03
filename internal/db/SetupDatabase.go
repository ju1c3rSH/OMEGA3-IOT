package db

import (
	"OMEGA3-IOT/internal/config"
	"OMEGA3-IOT/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
)

var DB *gorm.DB

func InitDB(config config.Config) {
	dsn := config.Database.DSN

	var err error

	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	if err := DB.AutoMigrate(&model.User{}, &model.Instance{}, &model.DeviceRegistrationRecord{}); err != nil {
		log.Fatal(err)
	}

	log.Println("Database Connection inited")
}
