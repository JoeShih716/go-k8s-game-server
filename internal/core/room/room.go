package room

import (
	"time"

	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
)

// GameRoom 定義了所有遊戲房間必須遵循的標準生命週期介面。
// 無論是捕魚機、對戰遊戲還是卡牌遊戲，都必須實作此介面以接入 Stateful Engine。
type GameRoom interface {
	// ID 回傳房間的唯一標識符
	//
	// 回傳值:
	//
	//	string: 房間 ID
	ID() string

	// Init 初始化房間
	// 通常在房間建立時呼叫一次，用於載入設定檔或初始化地圖。
	//
	// 參數:
	//
	//	config: []byte - 初始設定資料 (例如 JSON 格式的遊戲參數)
	//
	// 回傳值:
	//
	//	error: 若初始化失敗則回傳錯誤
	Init(config []byte) error

	// OnTick 每幀更新 (Game Loop 核心)
	// 引擎會以固定的頻率 (例如 20Hz) 呼叫此方法。
	//
	// 參數:
	//
	//	dt: time.Duration - 距離上一幀的時間差 (Delta Time)
	OnTick(dt time.Duration)

	// OnAction 處理玩家操作
	//
	// 參數:
	//
	//	userID: string - 操作者的 User ID
	//	action: []byte - 操作指令資料 (通常是 Protobuf 序列化後的 binary)
	//
	// 回傳值:
	//
	//	error: 若處理失敗則回傳錯誤
	OnAction(userID string, action []byte) error

	// OnJoin 當玩家嘗試加入房間時觸發
	//
	// 參數:
	//
	//	user: *domain.User - 加入的使用者物件
	//
	// 回傳值:
	//
	//	error: 若拒絕加入 (例如滿房、餘額不足) 則回傳錯誤
	OnJoin(user *domain.User) error

	// OnLeave 當玩家離開房間時觸發
	//
	// 參數:
	//
	//	userID: string - 離開的使用者 ID
	OnLeave(userID string)

	// Close 關閉房間並釋放資源
	// 應在此處儲存最終遊戲狀態到資料庫。
	Close()
}
