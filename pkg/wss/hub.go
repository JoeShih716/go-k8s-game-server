package wss

import (
	"context"
	"log/slog"
)

// hub 維護一組活躍的客戶端，並將事件分派給所有已註冊的 Subscriber
type hub struct {
	clients     map[*connection]bool
	register    chan *connection
	unregister  chan *connection
	inbound     chan *clientMessage
	subscribers []Subscriber
	ctx         context.Context
	logger      *slog.Logger
}

// newHub 創建一個新的 hub 實例。
//
// @param ctx - 用於控制 hub 生命週期的上下文。
// @param logger - 用於記錄日誌的 slog 實例。
// @return *hub - 一個初始化完成的 hub 實例。
func newHub(ctx context.Context, logger *slog.Logger) *hub {
	return &hub{
		register:    make(chan *connection),
		unregister:  make(chan *connection),
		inbound:     make(chan *clientMessage),
		clients:     make(map[*connection]bool),
		subscribers: make([]Subscriber, 0),
		ctx:         ctx,
		logger:      logger,
	}
}

// registerSubscriber 註冊一個新的事件處理器 (Subscriber)。
//
// @param subscriber - 實現了 Subscriber 介面的事件處理器。
func (h *hub) registerSubscriber(subscriber Subscriber) {
	if subscriber != nil {
		h.subscribers = append(h.subscribers, subscriber)
	}
}

// run 啟動 hub 的主事件迴圈。
// 這個迴圈會處理客戶端的註冊、註銷、訊息傳遞以及優雅關閉的邏輯。
func (h *hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			h.logger.Info("client registered", "clientID", client.ID())
			for _, subscriber := range h.subscribers {
				subscriber.OnConnect(client)
			}
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.logger.Info("client unregistered", "clientID", client.ID())
				for _, subscriber := range h.subscribers {
					subscriber.OnDisconnect(client)
				}
			}
		case msg := <-h.inbound:
			h.logger.Debug("message received from client", "clientID", msg.client.ID())
			for _, subscriber := range h.subscribers {
				subscriber.OnMessage(msg.client, msg.message)
			}
		case <-h.ctx.Done():
			// Context 被取消，開始關閉程序
			h.logger.Info("hub shutting down")
			for client := range h.clients {
				err := client.Kick("Server is shutting down.")
				if err != nil {
					h.logger.Error("kick client failed", "error", err, "clientID", client.ID())
				}
				delete(h.clients, client)
				close(client.send)
			}
			return // 結束 run 迴圈
		}
	}
}
