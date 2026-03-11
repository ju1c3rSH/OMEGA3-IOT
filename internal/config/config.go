package config

import (
	"fmt"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
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
//
//	type TLS struct {
//		Enabled        bool   `mapstructure:"enabled"`
//		CACertFile     string `mapstructure:"ca_cert_file"`
//		ClientCertFile string `mapstructure:"client_cert_file"`
//		ClientKeyFile  string `mapstructure:"client_key_file"`
//	}
var RequiredFlags = []string{
	"database.mysqldsn",
	"server.port",
	"iotdb.host",
	"iotdb.port",
	"mqtt.broker.host",
	"mqtt.broker.port",
	"mqtt.client.id",
}

func DeLoadConfig(configDir string) (config Config, err error) {
	GeneralConfig := viper.New()
	MQTTConfig := viper.New()
	CliInputConfig := viper.New()

	GeneralConfig.AddConfigPath(configDir)
	GeneralConfig.SetConfigName("GeneralConfig")
	GeneralConfig.SetConfigType("yaml")

	MQTTConfig.SetConfigName("mqtt_config")
	MQTTConfig.SetConfigType("yaml")
	MQTTConfig.AddConfigPath(configDir)

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
	// config.MQTT = mqttConfigPart.MQTT
	if len(os.Args) == 1 {
		printUsageAndExit("错误: 必须提供命令行参数")
	}

	defineFlags()
	pflag.Parse()

	if help, _ := pflag.CommandLine.GetBool("help"); help {
		pflag.Usage()
		os.Exit(0)
	}

	checkRequiredFlags()

	if err := CliInputConfig.BindPFlags(pflag.CommandLine); err != nil {
		return config, fmt.Errorf("绑定参数失败: %w", err)
	}

	if err := CliInputConfig.Unmarshal(&config, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),

			func(f, t reflect.Type, data interface{}) (interface{}, error) {
				if t.Kind() == reflect.Int && f.Kind() == reflect.String {
					if val, err := strconv.Atoi(data.(string)); err == nil {
						if val < 0 || val > 2 {
							return nil, fmt.Errorf("QoS 值 %d 超出范围 [0,2]", val)
						}
						return val, nil
					}
				}
				return data, nil
			},
		),
	)); err != nil {
		return config, fmt.Errorf("解析配置到结构体失败: %w", err)
	}

	return config, nil
}
func LoadConfig(path string) (Config, error) {
	var config Config
	v := viper.New()
	defineFlags()

	v.AddConfigPath(path)
	v.SetConfigName("GeneralConfig")
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return config, fmt.Errorf("failed to read config files %w", err)
		}
		log.Println("Undetected config file, using defaults")
	}

	pflag.Parse()

	if help, _ := pflag.CommandLine.GetBool("help"); help {
		pflag.Usage()
		os.Exit(0)
	}
	if err := v.BindPFlags(pflag.CommandLine); err != nil {
		return config, fmt.Errorf("failed to bind the cli params: %w", err)
	}
	if err := v.Unmarshal(&config, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			func(f, t reflect.Type, data interface{}) (interface{}, error) {
				//源类型是int，目标类型是uint8(byte)
				if t.Kind() == reflect.Uint8 && f.Kind() == reflect.Int {
					val := data.(int)
					if val < 0 || val > 2 {
						return nil, fmt.Errorf("QoS 值 %d 超出范围 [0,2]", val)
					}
					return byte(val), nil
				}
				// from number string to int
				if (t.Kind() == reflect.Int || t.Kind() == reflect.Int64) && f.Kind() == reflect.String {
					if str, ok := data.(string); ok && str != "" {
						if t.Kind() == reflect.Int {
							return strconv.Atoi(str)
						}
						return strconv.ParseInt(str, 10, 64)
					}
				}
				return data, nil
			},
		),
	)); err != nil {
		return config, fmt.Errorf("failed to parse the config into struct: %w", err)
	}

	return config, nil
}
func checkRequiredFlags() {
	var missing []string
	requiredFlags := []string{
		"database.mysqldsn",
		"server.port",
		"iotdb.host",
		"iotdb.port",
		"mqtt.broker.host",
		"mqtt.broker.port",
		"mqtt.client.id",
	}

	for _, flag := range requiredFlags {
		if !pflag.CommandLine.Changed(flag) {
			missing = append(missing, flag)
		}
	}

	if len(missing) > 0 {
		printUsageAndExit(fmt.Sprintf("缺少必需参数:\n  %s", strings.Join(missing, "\n  ")))
	}
}

func printUsageAndExit(message string) {
	fmt.Fprintln(os.Stderr, message)
	fmt.Fprintln(os.Stderr)
	pflag.Usage()
	os.Exit(1)
}

func defineFlags() {
	pflag.String("database.mysqldsn", "", "MySQL DSN (Required)")

	pflag.String("server.port", "", "Server Http Port (Required)")

	// IoTDB配置
	pflag.String("iotdb.host", "", "IoTDB Host (Required)")
	pflag.String("iotdb.port", "", "IoTDB Port (Required)")
	pflag.String("iotdb.username", "root", "IoTDB Username (Required)")
	pflag.String("iotdb.password", "root", "IoTDB Password (Required)")
	pflag.Int64("iotdb.querytimeoutms", 30000, "IoTDB query timeout (Required)")

	// IoTDB连接池配置
	pflag.Int("iotdb.pool.maxconnections", 10, "IoTDB 连接池最大连接数")
	pflag.Int64("iotdb.pool.timeout", 5000, "IoTDB 连接池超时")
	pflag.Bool("iotdb.pool.fetchmetadataauto", true, "IoTDB 自动获取元数据")

	// MQTT Broker配置
	pflag.String("mqtt.broker.host", "", "MQTT Broker 主机 (必需)")
	pflag.Int("mqtt.broker.port", 0, "MQTT Broker 端口 (必需)")
	pflag.String("mqtt.broker.protocol", "tcp", "MQTT Broker 协议 (tcp/ssl)")

	// MQTT Client配置
	pflag.String("mqtt.client.id", "", "MQTT 客户端 ID (必需)")
	pflag.String("mqtt.client.username", "", "MQTT 用户名")
	pflag.String("mqtt.client.password", "", "MQTT 密码")
	pflag.Bool("mqtt.client.clean_session", true, "MQTT 清理会话")
	pflag.Bool("mqtt.client.auto_reconnect", true, "MQTT 自动重连")
	pflag.Int("mqtt.client.qos", 1, "MQTT QoS 级别 (0,1,2)")

	pflag.StringP("config", "c", "", "配置文件路径 (可选)")

	pflag.BoolP("help", "h", false, "显示帮助信息")

	pflag.Usage = func() {
		fmt.Fprintf(pflag.CommandLine.Output(), "使用方法: %s [选项]\n\n", "your-program")
		fmt.Fprintln(pflag.CommandLine.Output(), "必需参数:")
		for _, flag := range RequiredFlags {
			f := pflag.CommandLine.Lookup(flag)
			if f != nil {
				fmt.Fprintf(pflag.CommandLine.Output(), "  --%-30s %s\n", f.Name, f.Usage)
			}
		}
		fmt.Fprintln(pflag.CommandLine.Output(), "\n可选参数:")
		pflag.CommandLine.VisitAll(func(f *pflag.Flag) {
			if !isRequired(f.Name) {
				fmt.Fprintf(pflag.CommandLine.Output(), "  --%-30s %s (默认: %s)\n", f.Name, f.Usage, f.DefValue)
			}
		})
		fmt.Fprintln(pflag.CommandLine.Output(), "\n示例:")
		fmt.Fprintln(pflag.CommandLine.Output(), "  ./app \\")
		fmt.Fprintln(pflag.CommandLine.Output(), "    --database.mysqldsn=\"user:pass@tcp(localhost:3306)/db\" \\")
		fmt.Fprintln(pflag.CommandLine.Output(), "    --server.port=8080 \\")
		fmt.Fprintln(pflag.CommandLine.Output(), "    --iotdb.host=localhost --iotdb.port=6667 \\")
		fmt.Fprintln(pflag.CommandLine.Output(), "    --mqtt.broker.host=localhost --mqtt.broker.port=1883 \\")
		fmt.Fprintln(pflag.CommandLine.Output(), "    --mqtt.client.id=mqtt-client-1")
	}
}
func isRequired(flagName string) bool {
	for _, required := range RequiredFlags {
		if required == flagName {
			return true
		}
	}
	return false
}
func loadConfigFile(path string) error {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	for key, value := range v.AllSettings() {
		viper.Set(key, value)
	}
	return nil
}

func validateConfig(config Config) error {
	return nil
}
