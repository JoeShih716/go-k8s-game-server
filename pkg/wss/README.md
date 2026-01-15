# WebSocket Server Package

`pkg/wss` 提供了一個高效且可擴展的 WebSocket 伺服器框架，專為遊戲伺服器設計。它封裝了 `gorilla/websocket` 並提供了連線管理、廣播機制與 CSRF 防護。

## 功能特性

-   **Hub & Pattern**: 採用 Hub 模式集中管理連線，支援多房間或頻道的擴充。
-   **安全防護**: 支援 `CheckOrigin` 檢查，可透過 Config 設定允許的來源網域。
-   **讀寫分離**: 每個連線啟動兩個 Goroutine (ReadPump / WritePump) 處理 I/O，互不阻塞。
-   **優雅關閉**: 支援 Context 取消傳播。
-   **Panic Recovery**: 內建 Recover 機制，確保單一連線發生 Panic 時不會導致整個伺服器崩潰。

## 使用範例

### 設定與啟動

```go
// 1. 設定
cfg := &wss.Config{
    WriteWait:       10 * time.Second,
    PongWait:        60 * time.Second,
    PingPeriod:      54 * time.Second,
    MaxMessageSize:  512,
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    // 設定允許跨域的來源 (根據不同環境設定)
    AllowedOrigins:  []string{"http://localhost:3000", "https://game.example.com"},
}

// 2. 初始化 Logger
logger := slog.Default()

// 3. 建立 Server
server := wss.NewServer(context.Background(), cfg, logger)

// 4. 註冊 Handler (Subscriber)
server.Register(&MyGameHandler{})

// 5. 綁定到 HTTP Router
http.Handle("/ws", server)
log.Fatal(http.ListenAndServe(":8080", nil))
```

### 處理事件 (Subscriber)

需實作 `Subscriber` 介面：

```go
type Subscriber interface {
    // 當新連線建立時觸發
    OnConnect(conn Client)
    // 當連線收到訊息時觸發
    OnMessage(conn Client, msg []byte)
    // 當連線斷開時觸發
    OnDisconnect(conn Client)
}
```
