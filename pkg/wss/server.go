package wss

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
)

// Server 是 websocket package 對外的主要門面 (Facade)，並實現了 http.Handler 介面。
type Server struct {
	hub    *hub
	cfg    *Config
	logger *slog.Logger
}

// 確保 Server 實現了 http.Handler 介面
var _ http.Handler = (*Server)(nil)

// NewServer 創建並設定一個完整的 WebSocket 伺服器。
//
// @param ctx - 用於控制伺服器生命週期的上下文。
// @param cfg - WebSocket 伺服器的設定參數。
// @param logger - 用於記錄日誌的 slog 實例。
// @return *Server - 一個初始化完成的 WebSocket 伺服器實例。
func NewServer(ctx context.Context, cfg *Config, logger *slog.Logger) *Server {
	// 如果 PingPeriod 沒有被設定，則根據 PongWait 計算一個合理的值
	if cfg.PingPeriod == 0 && cfg.PongWait > 0 {
		cfg.PingPeriod = (cfg.PongWait * 9) / 10
	}

	h := newHub(ctx, logger.With("component", "hub"))
	go h.run()
	return &Server{
		hub:    h,
		cfg:    cfg,
		logger: logger.With("component", "wss_server"),
	}
}

// Register 將一個業務邏輯處理器 (Subscriber) 註冊到 WebSocket 伺服器。
//
// @param subscriber - 實現了 Subscriber 介面的事件處理器。
func (s *Server) Register(subscriber Subscriber) {
	s.hub.registerSubscriber(subscriber)
}

// ServeHTTP 實現 http.Handler 介面，處理 WebSocket 的升級請求。
//
// @param w - http.ResponseWriter，用於寫入 HTTP 回應。
// @param r - *http.Request，收到的 HTTP 請求。
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  s.cfg.ReadBufferSize,
		WriteBufferSize: s.cfg.WriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			// 若未設定 AllowedOrigins，為了安全起見，預設拒絕所有跨域連線 (只允許同源)
			// 或者可以選擇開發模式下 return true，但這裡採取較嚴格的策略。
			if len(s.cfg.AllowedOrigins) == 0 {
				return false // 或者 return true if you want default distinct from strict
			}

			origin := r.Header.Get("Origin")
			// 如果沒有 Origin 標頭，通常是非瀏覽器請求 (e.g. Server-to-Server)，通常允許
			if origin == "" {
				return true
			}

			// 檢查 Origin 是否在允許列表中
			for _, allowed := range s.cfg.AllowedOrigins {
				if allowed == "*" {
					return true
				}
				if allowed == origin {
					return true
				}
			}
			return false
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Warn("websocket upgrade failed", "error", err)
		return
	}

	clientLogger := s.logger.With("component", "client")
	client := newConnection(s.hub, conn, r, clientLogger)
	client.hub.register <- client

	go client.writePump(s.cfg)
	go client.readPump(s.cfg)
}
