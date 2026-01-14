package wss

import "time"

// Config 定義了 WebSocket 伺服器的所有可設定參數。
type Config struct {
	WriteWait       time.Duration // 寫入操作的超時時間
	PongWait        time.Duration // 等待 Pong 訊息的超時時間
	PingPeriod      time.Duration // 發送 Ping 訊息的間隔
	MaxMessageSize  int64         // 允許接收的最大訊息大小
	ReadBufferSize  int           // 讀取緩衝區的大小
	WriteBufferSize int           // 寫入緩衝區的大小
	AllowedOrigins  []string      // 允許進行跨域請求的來源列表 (e.g., "http://localhost:3000")
}
