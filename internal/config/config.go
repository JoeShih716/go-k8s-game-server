package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// RedisGlobalConfig matches the hierarchy: redis -> (addr, db -> (central -> name))
type RedisGlobalConfig struct {
	Addr     string                   `mapstructure:"addr"`
	Password string                   `mapstructure:"password"`
	DB       map[string]RedisDBConfig `mapstructure:"db"`
}

type RedisDBConfig struct {
	Name int `mapstructure:"name"` // Matches "name: 0" from user request (DB Index)
}

type AppConfig struct {
	Name     string `mapstructure:"name"`
	Env      string `mapstructure:"env"`
	Port     int    `mapstructure:"port"`
	GrpcPort int    `mapstructure:"grpc_port"` // gRPC Server Port (Connector, etc.)
	PodIP    string `mapstructure:"-"`         // Pod IP (runtime injected, not from file)
}

// Config 總配置結構
type Config struct {
	App      AppConfig         `mapstructure:"app"`
	Redis    RedisGlobalConfig `mapstructure:"redis"`
	MySQL    MySQLConfig       `mapstructure:"mysql"`
	WSS      WSSConfig         `mapstructure:"wss"`
	Services map[string]string `mapstructure:"services"`
}

type MySQLConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
}

type WSSConfig struct {
	Path            string   `mapstructure:"path"`
	AllowedOrigins  []string `mapstructure:"allowed_origins"`
	ReadBufferSize  int      `mapstructure:"read_buffer_size"`
	WriteBufferSize int      `mapstructure:"write_buffer_size"`
	WriteWaitSec    int      `mapstructure:"write_wait_sec"`
	PongWaitSec     int      `mapstructure:"pong_wait_sec"`
	MaxMessageSize  int64    `mapstructure:"max_message_size"`
}

// Load 讀取設定檔
// 使用 Viper 讀取 config.yaml 並自動映射環境變數
//
// 環境變數映射規則 (Auto-Env Mapping):
// Viper 會自動將環境變數映射到 Config 結構的欄位。
// 規則是: 全大寫 + 將 "." 替換為 "_"。
//
// 範例:
//   - Redis.Addr     -> REDIS_ADDR
//   - MySQL.Password -> MYSQL_PASSWORD
//   - App.GrpcPort   -> APP_GRPC_PORT
//
// 這意味著你不需要在 env_keys.go 定義所有 Key，
// 只要環境變數名稱符合上述規則，Viper 就會自動讀取並覆蓋 YAML 中的值。
func Load(configPath ...string) (*Config, error) {
	v := viper.New()

	// 1. 設定檔路徑
	dir := "./config"
	if len(configPath) > 0 {
		dir = configPath[0]
	}
	v.AddConfigPath(dir)
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// 2. 設定環境變數自動映射
	// 將 . 替換為 _ (e.g. redis.addr -> REDIS_ADDR)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 3. 讀取設定檔
	if err := v.ReadInConfig(); err != nil {
		// 如果沒有設定檔，仍然繼續 (可能只依賴 Env)
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// 4. 解析到 Struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		// Try unmarshal
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 5. 手動處理某些非 Standard 的 Env 覆蓋 (如果是 Viper 自動映射無法覆蓋的情況)
	// 例如 PodIP 這種不在 yaml 內的
	if podIP := os.Getenv("POD_IP"); podIP != "" {
		cfg.App.PodIP = podIP
	}

	// 確保 App.Env 有值 (從 APP_ENV)
	if env := os.Getenv("APP_ENV"); env != "" {
		cfg.App.Env = env
	}

	// 支援標準 PORT 環境變數 (常見於 Cloud Run/Docker)
	if portStr := os.Getenv("PORT"); portStr != "" {
		var port int
		if _, err := fmt.Sscanf(portStr, "%d", &port); err == nil {
			cfg.App.Port = port
		}
	}
	// 支援 GRPC_PORT 環境變數
	if grpcPortStr := os.Getenv("GRPC_PORT"); grpcPortStr != "" {
		var gPort int
		if _, err := fmt.Sscanf(grpcPortStr, "%d", &gPort); err == nil {
			cfg.App.GrpcPort = gPort
		}
	}

	return &cfg, nil
}
