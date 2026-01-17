package manager

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/JoeShih716/go-k8s-game-server/internal/applications/stateful-demo/rooms"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/room"
)

// Manager 負責管理所有的 GameRoom 生命週期
type Manager struct {
	mu    sync.RWMutex
	rooms map[string]room.GameRoom

	// Ticker Channel 用於全域驅動所有房間
	stopChan chan struct{}
}

func NewManager() *Manager {
	return &Manager{
		rooms:    make(map[string]room.GameRoom),
		stopChan: make(chan struct{}),
	}
}

// Start 啟動主要的 Tick Loop
// 在 Goroutine-per-room 的架構下，其實每個 Room 可以自己跑 Ticker。
// 但為了更好的控制 (e.g. Pause, Rate Limit)，由 Manager 統一驅動也是一種選擇。
// 這裡展示：Manager 啟動一個 Global Ticker，每 100ms 觸發一次所有房間的 OnTick。
func (m *Manager) Start(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond) // 10Hz
	defer ticker.Stop()

	slog.Info("Room Manager Started")

	for {
		select {
		case <-m.stopChan:
			slog.Info("Room Manager Stopped")
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.tickAll(100 * time.Millisecond)
		}
	}
}

func (m *Manager) Stop() {
	close(m.stopChan)
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, r := range m.rooms {
		r.Close()
	}
}

func (m *Manager) tickAll(dt time.Duration) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 這裡示範序列執行，若房間數多，應改為 Worker Pool 平行執行
	for _, r := range m.rooms {
		// 為了避免單一房間卡死整個 Loop，建議 OnTick 內部不要做 I/O
		// 或者在這裡開 goroutine: go r.OnTick(dt)
		go r.OnTick(dt)
	}
}

// EnsureRoom 取得或建立房間
func (m *Manager) EnsureRoom(roomID string) (room.GameRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if r, exists := m.rooms[roomID]; exists {
		return r, nil
	}

	// 建立新房間
	newRoom := rooms.NewDemoRoom(roomID)
	if err := newRoom.Init(nil); err != nil {
		return nil, fmt.Errorf("failed to init room: %w", err)
	}

	m.rooms[roomID] = newRoom
	slog.Info("Created New Room", "roomID", roomID)
	return newRoom, nil
}

// GetRoom 取得房間
func (m *Manager) GetRoom(roomID string) (room.GameRoom, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	r, exists := m.rooms[roomID]
	if !exists {
		return nil, fmt.Errorf("room not found: %s", roomID)
	}
	return r, nil
}

// JoinRoom 處理玩家加入
func (m *Manager) JoinRoom(roomID string, user *domain.User) error {
	r, err := m.EnsureRoom(roomID)
	if err != nil {
		return err
	}
	return r.OnJoin(user)
}

// LeaveRoom 處理玩家離開
func (m *Manager) LeaveRoom(roomID string, userID string) {
	m.mu.RLock()
	r, exists := m.rooms[roomID]
	m.mu.RUnlock()

	if exists {
		r.OnLeave(userID)
		// TODO: 檢查空房回收邏輯 (如果房間沒人，是否要在幾分鐘後銷毀？)
	}
}
