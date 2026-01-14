package domain

import (
	"context"

	"github.com/JoeShih716/go-k8s-game-server/pkg/wss"
)

// Session 代表一個活躍的連線會話。
// 它封裝了底層的 WebSocket 連線，並維護該連線的業務狀態 (如 UserID, 當前所在的 RoomID)。
type Session struct {
	ID        string     // Session 唯一 ID (通常對應 WebSocket Conn ID)
	UserID    string     // 綁定的使用者 ID
	RoomID    string     // 當前所在的房間 ID (若無則為空字串)
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
		ID:     conn.ID(),
		conn:   conn,
		RoomID: "", // 初始狀態不在任何房間
	}
}

// BindUser 將使用者 ID 綁定到此會話
//
// 參數:
//
//	userID: string - 使用者 ID
func (s *Session) BindUser(userID string) {
	s.UserID = userID
	// 同時更新底層連線的 Tag，方便 Log 追蹤
	s.conn.SetTag("user_id", userID)
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

// Context 獲取與底層連線相關的 Context (如果有的話)
// 這裡簡單回傳 Background，實際專案中可能需要從 conn 獲取更豐富的 Context
func (s *Session) Context() context.Context {
	return context.Background()
}
