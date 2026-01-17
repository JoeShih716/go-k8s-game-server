package domain

import (
	"github.com/JoeShih716/go-k8s-game-server/pkg/wss"
)

// Session 代表一個活躍的連線會話。
// 它封裝了底層的 WebSocket 連線，並維護該連線的業務狀態 (如 UserID)。
type Session struct {
	ID        string     // Session 唯一 ID (通常對應 WebSocket Conn ID)
	UserID    string     // 綁定的使用者 ID
	conn      wss.Client // 底層 WebSocket 連線介面
	CreatedAt int64      // 建立時間 (Unix Timestamp)
}

// NewSession 建立一個新的會話實例
//
// 參數:
//
//	conn: wss.Client - 底層 WebSocket 連線物件
//
// 回傳值:
//
//	*Session: 初始化後的會話物件
func NewSession(conn wss.Client) *Session {
	return &Session{
		ID:   conn.ID(),
		conn: conn,
	}
}

// Send 發送訊息給此會話的客戶端
//
// 參數:
//
//	msg: string - 訊息內容
//
// 回傳值:
//
//	error: 若發送失敗則回傳錯誤
func (s *Session) Send(msg string) error {
	return s.conn.SendMessage(msg)
}

// Kick 強制中斷此會話
//
// 參數:
//
//	reason: string - 踢除原因
//
// 回傳值:
//
//	error: 若操作失敗則回傳錯誤
func (s *Session) Kick(reason string) error {
	return s.conn.Kick(reason)
}
