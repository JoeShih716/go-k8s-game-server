package rooms

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/room"
)

// DemoRoom 實作了一個簡單的有狀態房間
// 邏輯：每秒 Counter +1，並廣播給房間內所有人
type DemoRoom struct {
	id      string
	mu      sync.RWMutex
	users   map[string]*domain.User
	counter int
	stopCh  chan struct{}
}

// Ensure implementation
var _ room.GameRoom = (*DemoRoom)(nil)

func NewDemoRoom(id string) *DemoRoom {
	return &DemoRoom{
		id:     id,
		users:  make(map[string]*domain.User),
		stopCh: make(chan struct{}),
	}
}

func (r *DemoRoom) ID() string {
	return r.id
}

func (r *DemoRoom) Init(config []byte) error {
	slog.Info("Room Initialized", "roomID", r.id, "config", string(config))
	return nil
}

// OnTick 由 RoomManager 驅動，或者房間自己跑 Loop (取決於 Manager 設計)
// 在這個範例中，我們假設 Manager 會呼叫 OnTick，或者我們在 Init 啟動自己的 Loop。
// 為了符合 Interface 定義，這裡處理單次 Tick 邏輯。
func (r *DemoRoom) OnTick(dt time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 1. 更新狀態
	r.counter++

	// 2. 每 5 秒廣播一次狀態 (避免太吵)
	if r.counter%5 == 0 {
		r.broadcastState()
	}
}

func (r *DemoRoom) OnAction(userID string, action []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	slog.Info("Action Received", "roomID", r.id, "userID", userID, "action", string(action))
	// 簡單實作：收到什麼就回傳什麼 (Echo)
	// 在實際應用中，這裡是處理遊戲指令 Switch Case 的地方
	return nil
}

func (r *DemoRoom) OnJoin(user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.ID]; exists {
		return fmt.Errorf("user already in room")
	}

	r.users[user.ID] = user
	slog.Info("User Joined Room", "roomID", r.id, "userID", user.ID, "count", len(r.users))

	// 廣播加入訊息 (TODO: 實作 Connector 廣播協議)
	return nil
}

func (r *DemoRoom) OnLeave(userID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.users, userID)
	slog.Info("User Left Room", "roomID", r.id, "userID", userID, "count", len(r.users))
}

func (r *DemoRoom) Close() {
	close(r.stopCh)
	slog.Info("Room Closed", "roomID", r.id)
}

// broadcastState 內部廣播方法 (實際應呼叫 Connector 推播)
// 由於 Stateful Service 位於後端，無法直接推 WebSocket。
// 必須透過:
// 1. gRPC Stream (Bi-directional)
// 2. Redis Pub/Sub (簡單但延遲較高)
// 3. Central 轉發
//
// 在此階段，我們僅單純列印 Log 模擬廣播。
func (r *DemoRoom) broadcastState() {
	state := map[string]interface{}{
		"roomID":  r.id,
		"counter": r.counter,
		"users":   len(r.users),
	}
	bytes, _ := json.Marshal(state)
	slog.Info("Broadcasting State", "roomID", r.id, "state", string(bytes))
}
