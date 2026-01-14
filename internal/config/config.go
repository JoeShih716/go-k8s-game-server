package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config 總配置結構
type Config struct {
	App   AppConfig   `yaml:"app"`
	Redis RedisConfig `yaml:"redis"`
	MySQL MySQLConfig `yaml:"mysql"`
	WSS   WSSConfig   `yaml:"wss"`
}

type AppConfig struct {
	Name string `yaml:"name"`
	Env  string `yaml:"env"`
	Port int    `yaml:"port"`
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
// env: 環境名稱 (例如 "local", "dev", "prod")
// configPath: 設定檔目錄路徑 (預設為 "./config")
func Load(env string, configPath ...string) (*Config, error) {
	if env == "" {
		env = "local"
	}

	path := "./config"
	if len(configPath) > 0 {
		path = configPath[0]
	}

	filename := fmt.Sprintf("%s.yaml", env)
	fullPath := filepath.Join(path, filename)

	// 支援讀取 config.yaml 作為預設，再讀取 env 特定檔案覆蓋 (這裡先簡單實作只讀取特定 env)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		// 嘗試讀取絕對路徑 (考慮到 Docker 中路徑可能不同)
		absPath, _ := filepath.Abs(fullPath)
		return nil, fmt.Errorf("failed to read config file at %s (abs: %s): %w", fullPath, absPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	// 允許環境變數覆蓋 (簡單實作: 對於密碼等敏感資訊)
	if p := os.Getenv("MYSQL_PASSWORD"); p != "" {
		cfg.MySQL.Password = p
	}
	if p := os.Getenv("REDIS_PASSWORD"); p != "" {
		cfg.Redis.Password = p
	}

	return &cfg, nil
}
