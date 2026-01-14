package wss

import "net/http"

// Client 定義了客戶端連線對外暴露的行為。
// 業務邏輯層將依賴此介面，而非具體的 Connection 實作。
type Client interface {
	// ID 返回客戶端的唯一標識符。
	ID() string
	// SendMessage 發送文字訊息給客戶端。
	SendMessage(message string) error
	// Kick 中斷與客戶端的連線。
	Kick(reason string) error
	// RemoteAddr 返回客戶端的網路位址。
	RemoteAddr() string
	// Headers 返回客戶端升級請求時的 HTTP 標頭。
	Headers() http.Header
	// UserAgent 返回客戶端的 User-Agent。
	UserAgent() string
	// SetTag 在該連線的生命週期內附加一個鍵值對資料。
	SetTag(key string, value any)
	// GetTag 根據鍵名讀取之前用 SetTag 附加的資料。
	GetTag(key string) (value any, exists bool)
}
