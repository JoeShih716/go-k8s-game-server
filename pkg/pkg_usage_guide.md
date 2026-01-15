# 核心工具庫 (pkg/) 開發者指南

這份指南是為了讓團隊成員（以及未來的您）能快速理解 `pkg/` 下各個工具的**設計哲學**、**適用場景**以及在遊戲伺服器中的**實際應用範例**。

---

## 1. `pkg/wss` - 網關守門員 (WebSocket Server)

### 設計哲學
-   **抽象化 (Abstraction)**: 隱藏了 WebSocket 底層的複雜性 (Ping/Pong, Read/Write Loop, Buffer Management)。
-   **安全性 (Security)**: 內建 CSRF 防護 (Origin Check) 與連線限制。
-   **擴充性 (Scale)**: 採用 Pub/Sub 模式 (Subscriber Interface)，讓業務邏輯 (如大廳、斷線重連) 可以輕鬆掛載，而不必修改核心代碼。

### 核心介面
-   **`Client` Interface**: 這是您在業務邏輯中最常打交道的物件。
    -   `SendMessage(msg string)`: 推送訊息給玩家。
    -   `Kick(reason string)`: 踢人。
    -   `GetTag/SetTag`: 用來暫存 Session 資料 (例如 UserID, RoomID)。

### 🎮 遊戲場景應用：玩家登入
```go
// 當玩家連線建立 (OnConnect)
func (h *LobbyHandler) OnConnect(client wss.Client) {
    // 1. 驗證 Token (偽代碼)
    userID, err := authService.Verify(token)

    // 2. 將 UserID 綁定到連線，方便後續取用
    client.SetTag("user_id", userID)

    // 3. 回傳歡迎訊息
    client.SendMessage(`{"type": "welcome", "data": "Hello Player!"}`)
}
```

---

## 2. `pkg/grpc` - 內部高速公路 (gRPC Pool)

### 設計哲學
-   **效能 (Performance)**: 解決了「每次呼叫外部服務都要建立連線」的高昂成本。
-   **並發 (Concurrency)**: 透過 Connection Pool 複用 HTTP/2 連線，支援高並發請求。
-   **可觀測性 (Observability)**: 支援 Interceptor，這意味著我們可以在這裡統一做 Prometheus 監控或 Distributed Tracing (Jaeger)，而不用改每個業務函式。

### 核心行為
-   **`GetConnection(target)`**: 給我一個通往目標 (e.g., `fishing-service`) 的連線，沒有就建一個，有就重複用。

### 🎮 遊戲場景應用：跨服務扣款
當 **Connector** (接受玩家指令) 需要呼叫 **Stateless Service** (Slot 機) 進行旋轉扣款時：

```go
// 1. 從 Pool 拿連線 (假設連去 Slot Service)
conn, _ := grpcPool.GetConnection("slots-service:8080")

// 2. 建立 gRPC Client
client := pb.NewSlotClient(conn)

// 3. 呼叫 (像呼叫本地函式一樣)
resp, err := client.Spin(ctx, &pb.SpinReq{Bet: 100})
```

---

## 3. `pkg/redis` - 高速共享記憶體 (Distributed State)

### 設計哲學
-   **便捷 (Convenience)**: 遊戲開發超常用 Struct (例如 `PlayerState`, `RoomInfo`)，所以我們特地封裝了 `GetStruct`/`SetStruct` 來自動處理 JSON。
-   **協作 (Coordination)**: 在 K8s 多 Pod 環境下，Local Memory 是不可靠的。Redis 是所有服務共享的「真實狀態來源」。

### 核心功能
-   **`AcquireLock`**: 分散式鎖。這是防止「連點兩次按鈕導致重複扣款」或「兩個人同時坐同一個位置」的神器。
-   **Pub/Sub**: 輕量級訊息廣播。

### 🎮 遊戲場景應用：搶位置 (鎖機制)
```go
lockKey := "room:101:seat:3" // 房間 101 的 3 號位

// 嘗試搶鎖 (鎖 3 秒，足夠處理入座邏輯)
if success, _ := redisClient.AcquireLock(ctx, lockKey, userID, 3*time.Second); success {
    // 搶到了！執行入座邏輯
    // ...
    // 最後釋放鎖，或等待自動過期
    redisClient.ReleaseLock(ctx, lockKey, userID)
} else {
    // 沒搶到，回傳「位置已被佔用」
}
```

---

## 4. `pkg/mysql` - 永久金庫 (Persistence)

### 設計哲學
-   **標準化 (Standardization)**: 強制統一的連線池設定 (MaxOpen/MaxIdle)，這是防止資料庫被海量連線衝垮的關鍵。
-   **開發生產力 (Productivity)**: 選用 GORM，讓一般的 CRUD 操作 (Create, Read, Update, Delete) 變得像寫英文句子一樣簡單。

### 核心行為
-   提供 `*gorm.DB` 實例，這是業界標準做法，保留最大彈性。

### 🎮 遊戲場景應用：儲存遊戲紀錄
```go
// 定義這一局的紀錄
record := GameRecord{
    UserID: "user_123",
    GameID: "slot_zeus",
    Bet:    100,
    Win:    500,
    Time:   time.Now(),
}

// 一行寫入 DB
// GORM 會自動產生: INSERT INTO game_records (...) VALUES (...)
err := mysqlClient.DB().Create(&record).Error
```

---

## 總結：它們如何協作？

想像一個 **「老虎機旋轉 (Spin)」** 的流程：

1.  **玩家手機** 透過 **`pkg/wss`** 發送 "Spin" 指令。
2.  **Connector** 收到指令，透過 **`pkg/grpc`** 轉發給 **Slot Service**。
3.  **Slot Service**：
    -   先用 **`pkg/redis`** (AcquireLock) 鎖住玩家餘額，防止重複請求。
    -   計算結果後，用 **`pkg/mysql`** 寫入遊戲紀錄 (GameRecord)。
    -   更新 **`pkg/redis`** 中的玩家餘額緩存。
4.  **Slot Service** 將結果回傳給 **Connector**。
5.  **Connector** 透過 **`pkg/wss`** 將畫面結果推回去給玩家。

這就是我們打造這四個核心 Base 的原因！它們是 building blocks。
