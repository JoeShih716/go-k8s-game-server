package protocol

import (
	"encoding/json"

	"github.com/shopspring/decimal"
)

// ConnectorProtocol 定義指令代碼 (使用 string 方便前端對接)
type ConnectorProtocol string

const (
	ActionLogin     ConnectorProtocol = "login" // 登入
	ActionEnterGame ConnectorProtocol = "enter" // 進入遊戲
)

// Envelope 基礎封包結構 (所有請求的外層包裝)
type Envelope struct {
	Action  ConnectorProtocol `json:"action"`            // 指令代碼
	Payload json.RawMessage   `json:"payload,omitempty"` // 具體請求內容
}

// Response 通用回應結構 (所有回應的外層包裝)
type Response struct {
	Action ConnectorProtocol `json:"action"`            // 對應的指令代碼
	Data   any               `json:"data,omitempty"`    // 成功時的資料
	Error  string            `json:"error,omitempty"`   // 失敗時的錯誤訊息
}

// LoginReq 登入請求
type LoginReq struct {
	Token string `json:"token"`
}

// LoginResp 登入回應
type LoginResp struct {
	Success      bool            `json:"success"`
	ErrorMessage string          `json:"error_message,omitempty"`
	UserID       string          `json:"user_id"`
	Nickname     string          `json:"nickname"`
	Balance      decimal.Decimal `json:"balance"`
}

// EnterGameReq 進入遊戲請求
type EnterGameReq struct {
	GameID int32 `json:"game_id"`
}

// EnterGameResp 進入遊戲回應
type EnterGameResp struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"error_message,omitempty"`
	GameID       int32  `json:"game_id"`
}
