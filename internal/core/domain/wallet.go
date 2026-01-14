package domain

import "context"

// Wallet 定義了通用的錢包操作介面。
// 遵循依賴反轉原則 (DIP)，核心層只定義介面，具體實作 (Redis, HTTP API, In-Memory) 由基礎設施層負責。
type Wallet interface {
	// GetBalance 查詢使用者當前餘額
	//
	// 參數:
	//
	//	ctx: context.Context - 上下文
	//	userID: string - 使用者 ID
	//
	// 回傳值:
	//
	//	int64: 當前餘額 (以最小貨幣單位計算，例如: 分)
	//	error: 若查詢失敗則回傳錯誤
	GetBalance(ctx context.Context, userID string) (int64, error)

	// Deduct 扣除使用者餘額
	//
	// 參數:
	//
	//	ctx: context.Context - 上下文
	//	userID: string - 使用者 ID
	//	amount: int64 - 扣除金額 (必須大於 0)
	//	reason: string - 扣款原因 (用於稽核，例如: "bet:spin:123")
	//
	// 回傳值:
	//
	//	int64: 扣款後的最新餘額
	//	error: 若餘額不足或扣款失敗則回傳錯誤
	Deduct(ctx context.Context, userID string, amount int64, reason string) (int64, error)

	// Add 增加使用者餘額
	//
	// 參數:
	//
	//	ctx: context.Context - 上下文
	//	userID: string - 使用者 ID
	//	amount: int64 - 增加金額 (必須大於 0)
	//	reason: string - 加款原因 (用於稽核，例如: "win:spin:123")
	//
	// 回傳值:
	//
	//	int64: 加款後的最新餘額
	//	error: 若加款失敗則回傳錯誤
	Add(ctx context.Context, userID string, amount int64, reason string) (int64, error)
}
