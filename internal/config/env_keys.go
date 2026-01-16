package config

// Environment Variable Keys
const (
	// EnvAppEnv 定義應用程式執行環境 (local, local_k8s, dev, prod)
	EnvAppEnv = "APP_ENV"

	// EnvPort 定義 HTTP/Websocket 服務 Port
	EnvPort = "PORT"

	// EnvPodIP 定義 Pod IP (K8s Downward API)
	EnvPodIP = "POD_IP"

	// EnvCentralAddr 定義 Central 服務地址 (host:port)
	EnvCentralAddr = "CENTRAL_ADDR"

	// EnvRedisAddr 定義 Redis 服務地址 (host:port)
	EnvRedisAddr = "REDIS_ADDR"

	// EnvRedisPassword 定義 Redis 密碼
	EnvRedisPassword = "REDIS_PASSWORD"

	// EnvMySQLHost 定義 MySQL 主機
	EnvMySQLHost = "MYSQL_HOST"

	// EnvMySQLPassword 定義 MySQL 密碼
	EnvMySQLPassword = "MYSQL_PASSWORD"
)
