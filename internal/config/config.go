package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config 總配置結構
type Config struct {
	App      AppConfig         `yaml:"app"`
	Redis    RedisConfig       `yaml:"redis"`
	MySQL    MySQLConfig       `yaml:"mysql"`
	WSS      WSSConfig         `yaml:"wss"`
	Services map[string]string `yaml:"services"`
}

type AppConfig struct {
	Name     string `yaml:"name"`
	Env      string `yaml:"env"`
	Port     int    `yaml:"port"`
	GrpcPort int    `yaml:"grpc_port"` // gRPC Server Port (Connector, etc.)
	PodIP    string `yaml:"-"`         // Pod IP (runtime injected, not from file)
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
}

type WSSConfig struct {
	Path            string   `yaml:"path"`
	AllowedOrigins  []string `yaml:"allowed_origins"`
	ReadBufferSize  int      `yaml:"read_buffer_size"`
	WriteBufferSize int      `yaml:"write_buffer_size"`
	WriteWaitSec    int      `yaml:"write_wait_sec"`
	PongWaitSec     int      `yaml:"pong_wait_sec"`
	MaxMessageSize  int64    `yaml:"max_message_size"`
}

// Load 讀取設定檔
// 優先讀取 config/config.yaml，然後使用環境變數覆蓋
func Load(configPath ...string) (*Config, error) {
	// 1. 決定設定檔路徑
	dir := "./config"
	if len(configPath) > 0 {
		dir = configPath[0]
	}
	filename := "config.yaml"
	fullPath := filepath.Join(dir, filename)

	var cfg Config

	// 2. 讀取 YAML 檔案 (如果存在)
	// 若檔案不存在，也許是純 Env Var 運行模式，我們先 Log 或 Ignore，取決於策略。
	// 但 User 要求 "預設讀一份 config.yaml"，所以我們假設它存在。
	data, err := os.ReadFile(fullPath)
	if err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse yaml at %s: %w", fullPath, err)
		}
	} else {
		// 檔案讀取失敗，如果是找不到檔案，我們或許可以接受 (全靠 Env)，但這裡先 return error 比較安全
		// 除非我們確定想要 fallback 到 empty config
		return nil, fmt.Errorf("failed to read config file at %s: %w", fullPath, err)
	}

	// 3. 環境變數覆蓋 (Environment Variable Override)
	overrideWithEnv(&cfg)

	return &cfg, nil
}

func overrideWithEnv(cfg *Config) {
	// App
	if env := os.Getenv(EnvAppEnv); env != "" {
		cfg.App.Env = env
	}
	if portVal := os.Getenv(EnvPort); portVal != "" {
		if p, err := strconv.Atoi(portVal); err == nil {
			cfg.App.Port = p
		}
	}
	if grpcPortVal := os.Getenv(EnvGrpcPort); grpcPortVal != "" {
		if p, err := strconv.Atoi(grpcPortVal); err == nil {
			cfg.App.GrpcPort = p
		}
	}
	if podIP := os.Getenv(EnvPodIP); podIP != "" {
		cfg.App.PodIP = podIP
	}

	// MySQL
	if val := os.Getenv(EnvMySQLHost); val != "" {
		cfg.MySQL.Host = val
	}
	if val := os.Getenv(EnvMySQLPassword); val != "" {
		cfg.MySQL.Password = val
	}
	// TODO: 更多 MySQL Env Override (Port, User, DBName) 如果需要
	if val := os.Getenv(EnvMySQLUser); val != "" {
		cfg.MySQL.User = val
	}
	if val := os.Getenv(EnvMySQLDB); val != "" {
		cfg.MySQL.DBName = val
	}
	if val := os.Getenv(EnvMySQLPort); val != "" {
		if p, err := strconv.Atoi(val); err == nil {
			cfg.MySQL.Port = p
		}
	}

	// Redis
	if val := os.Getenv(EnvRedisAddr); val != "" {
		cfg.Redis.Addr = val
	}
	if val := os.Getenv(EnvRedisPassword); val != "" {
		cfg.Redis.Password = val
	}

	// Services
	if cfg.Services == nil {
		cfg.Services = make(map[string]string)
	}
	if val := os.Getenv(EnvCentralAddr); val != "" {
		cfg.Services["central"] = val
	}

	// Pod IP usually maps to Host/Endpoint logic, typically handled in main,
	// but can be stored if we added a field. For now, keep it out of Config struct
	// or handle it where needed (Registrar).
}
