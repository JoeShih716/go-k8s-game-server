package redis

import (
	"context"
)

// MessageHandler 定義訂閱訊息的處理函式類型
//
// 參數:
//
//	payload: string - 接收到的訊息內容
type MessageHandler func(payload string)

// Publish 發送訊息到指定頻道
//
// 參數:
//
//	ctx: context.Context - 上下文
//	channel: string - 目標頻道名稱
//	message: any - 要發送的訊息內容，可以是字串或可序列化的物件
//
// 回傳值:
//
//	error: 若發送失敗則回傳錯誤
func (c *Client) Publish(ctx context.Context, channel string, message any) error {
	return c.rdb.Publish(ctx, channel, message).Err()
}

// Subscribe 訂閱指定頻道並處理接收到的訊息
// 此方法會啟動一個背景 goroutine 來處理接收到的訊息。
//
// 參數:
//
//	ctx: context.Context - 上下文
//	channel: string - 要訂閱的頻道名稱
//	handler: MessageHandler - 訊息處理函式
//
// 回傳值:
//
//	error: 若訂閱失敗則回傳錯誤
func (c *Client) Subscribe(ctx context.Context, channel string, handler MessageHandler) error {
	pubsub := c.rdb.Subscribe(ctx, channel)

	// 驗證訂閱是否成功
	// Receive 會等待直到接收到訂閱確認訊息或發生錯誤
	if _, err := pubsub.Receive(ctx); err != nil {
		return err
	}

	// 啟動 goroutine 處理訊息
	go func() {
		defer pubsub.Close()
		ch := pubsub.Channel()

		// 監聽 Go channel，當 pubsub 被關閉時迴圈會結束
		for msg := range ch {
			handler(msg.Payload)
		}
	}()

	return nil
}
