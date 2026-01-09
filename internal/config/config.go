package config

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
)

type Config struct {
	Database struct {
		MYSQLDSN string `mapstructure:"mysqldsn"`
	} `mapstructure:"database"`
	Server struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"server"`
	MQTT struct {
		Broker Broker `mapstructure:"broker"`
		Client Client `mapstructure:"client"`
		// TLS    TLS    `mapstructure:"tls"` // 如果需要 TLS
	}
	IoTDB struct {
		Host           string `mapstructure:"host"`
		Port           string `mapstructure:"port"`
		UserName       string `mapstructure:"username"`
		Password       string `mapstructure:"password"`
		QueryTimeoutMs int64  `mapstructure:"queryTimeoutMs"`
		Pool           Pool   `mapstructure:"pool"`
	}
}

type Broker struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Protocol string `mapstructure:"protocol"` // e.g., "tcp", "ssl"
}
type Pool struct {
	MaxConnections    int   `mapstructure:"maxConnections"`
	TimeOut           int64 `mapstructure:"timeout"`
	FetchMetadataAuto bool  `mapstructure:"fetchMetadataAuto"`
}

func (b Broker) Address() string {
	return fmt.Sprintf("%s://%s:%d", b.Protocol, b.Host, b.Port)
}

type Client struct {
	ID            string `mapstructure:"id"`
	Username      string `mapstructure:"username"`
	Password      string `mapstructure:"password"`
	CleanSession  bool   `mapstructure:"clean_session"`
	AutoReconnect bool   `mapstructure:"auto_reconnect"`
	QoS           byte   `mapstructure:"qos"` // 注意：Viper 默认可能解析为 int，需要处理
}

// // 如果需要 TLS 配置
// type TLS struct {
// 	Enabled        bool   `mapstructure:"enabled"`
// 	CACertFile     string `mapstructure:"ca_cert_file"`
// 	ClientCertFile string `mapstructure:"client_cert_file"`
// 	ClientKeyFile  string `mapstructure:"client_key_file"`
// }

func LoadConfig(path string) (config Config, err error) {
	GeneralConfig := viper.New()
	MQTTConfig := viper.New()

	GeneralConfig.AddConfigPath(path)
	GeneralConfig.SetConfigName("GeneralConfig")
	GeneralConfig.SetConfigType("yaml")

	MQTTConfig.SetConfigName("mqtt_config")
	MQTTConfig.SetConfigType("yaml")
	MQTTConfig.AddConfigPath(path)

	if err := GeneralConfig.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
	if err := GeneralConfig.Unmarshal(&config); err != nil {
		log.Fatalf("Error unmarshalling config, %s", err)
	}
	var mqttConfigPart struct {
		MQTT struct {
			Broker Broker `mapstructure:"broker"`
			Client Client `mapstructure:"client"`
		} `mapstructure:"mqtt"`
	}

	if err := MQTTConfig.Unmarshal(&mqttConfigPart); err != nil {
		return config, fmt.Errorf("error unmarshalling MQTT config: %w", err)
	}

	config.MQTT = mqttConfigPart.MQTT

	if err := MQTTConfig.ReadInConfig(); err != nil {
		log.Fatalf("Error reading MQTT config file: %s", err)
	}
	if err := MQTTConfig.Unmarshal(&config); err != nil {
		log.Fatalf("Error unmarshalling MQTT config file: %s", err)
	}
	fmt.Printf("Config Loaded from %s\n\n", GeneralConfig.ConfigFileUsed())
	return config, nil
}
