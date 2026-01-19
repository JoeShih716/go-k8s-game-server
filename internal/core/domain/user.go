package domain

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

var ErrInvalidToken = errors.New("invalid token")

// User 代表系統中的一個使用者實體。
// 這是最基礎的資料結構，用於在各個服務層之間傳遞使用者資訊。
// 注意：Balance 為當前餘額快照
type User struct {
	ID        string          // 使用者唯一標識符
	Name      string          // 使用者顯示名稱 (Nickname)
	Balance   decimal.Decimal // 餘額 (Snapshot)
	CreatedAt time.Time       // 帳號建立時間
}

// NewUser 建立一個新的使用者實例
//
// 參數:
//
//	id: string - 使用者 ID
//	name: string - 使用者名稱
//
// 回傳值:
//
//	*User: 初始化後的使用者物件
func NewUser(id string, name string) *User {
	now := time.Now()
	return &User{
		ID:        id,
		Name:      name,
		Balance:   decimal.Zero,
		CreatedAt: now,
	}
}
