package config

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
)

type Config struct {
	Database struct {
		DSN string `mapstructure:"dsn"`
	} `mapstructure:"database"`
	Server struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"server"`
}

func LoadConfig(path string) (config Config) {
	GeneralConfig := viper.New()

	GeneralConfig.AddConfigPath(path)
	GeneralConfig.SetConfigName("GeneralConfig")
	GeneralConfig.SetConfigType("yaml")

	if err := GeneralConfig.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	if err := GeneralConfig.Unmarshal(&config); err != nil {
		log.Fatalf("Error unmarshalling config, %s", err)
	}

	fmt.Printf("Config Loaded from %s\n\n", GeneralConfig.ConfigFileUsed())
	return
}
