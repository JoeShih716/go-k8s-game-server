package domain

import "time"

// User 代表系統中的一個使用者實體。
// 這是最基礎的資料結構，用於在各個服務層之間傳遞使用者資訊。
// 注意：此處不包含 Balance (餘額)，餘額操作請透過 Wallet 介面。
type User struct {
	ID        string    // 使用者唯一標識符 (UUID 或資料庫 ID)
	Name      string    // 使用者顯示名稱
	AvatarURL string    // 使用者頭像連結
	CreatedAt time.Time // 帳號建立時間
	UpdatedAt time.Time // 最後更新時間
	Tags      []string  // 使用者標籤 (例如: "vip", "newbie")
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
		CreatedAt: now,
		UpdatedAt: now,
		Tags:      make([]string, 0),
	}
}
