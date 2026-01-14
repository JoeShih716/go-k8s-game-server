package session

import (
	"sync"
	"sync/atomic"

	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
)

// Manager 負責管理 Connector 內所有的 Session
// 它是 Thread-Safe 的，支援並發讀寫。
type Manager struct {
	sessions sync.Map // Map[string]*domain.Session
	count    int64    // 在線人數計數器
}

// NewManager 建立新的 Session 管理器
func NewManager() *Manager {
	return &Manager{}
}

// Add 新增一個 Session
func (m *Manager) Add(session *domain.Session) {
	_, loaded := m.sessions.LoadOrStore(session.ID, session)
	if !loaded {
		atomic.AddInt64(&m.count, 1)
	}
}

// Remove 移除一個 Session
func (m *Manager) Remove(sessionID string) {
	_, loaded := m.sessions.LoadAndDelete(sessionID)
	if loaded {
		atomic.AddInt64(&m.count, -1)
	}
}

// Get 取得 Session
func (m *Manager) Get(sessionID string) (*domain.Session, bool) {
	val, ok := m.sessions.Load(sessionID)
	if !ok {
		return nil, false
	}
	return val.(*domain.Session), true
}

// Count 取得當前在線人數
func (m *Manager) Count() int64 {
	return atomic.LoadInt64(&m.count)
}

// Range 遍歷所有 Session (用於廣播等操作)
// handler 回傳 false 則停止遍歷
func (m *Manager) Range(handler func(s *domain.Session) bool) {
	m.sessions.Range(func(key, value any) bool {
		return handler(value.(*domain.Session))
	})
}
