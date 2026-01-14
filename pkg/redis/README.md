# Redis Package

`pkg/redis` 為 `go-redis` 提供了輕量級的封裝，旨在簡化遊戲伺服器中常見的緩存與狀態管理操作。

## 功能特性

-   **連線管理**: 簡單的 `NewClient` 初始化與自動 Ping 檢查。
-   **結構體存取**: 提供 `SetStruct` 與 `GetStruct` 泛型方法，自動處理 JSON 序列化與反序列化。
-   **Pub/Sub**: 封裝了發布/訂閱模式，便於跨服務傳遞訊息。
-   **分散式鎖**: 提供 `AcquireLock` 與 `ReleaseLock`，用於處理跨 Pod 的並發控制 (e.g., 搶紅包, 同一房間的狀態修改)。

## 使用範例

### 初始化

```go
cfg := redis.Config{
    Addr:     "localhost:6379",
    Password: "",
    DB:       0,
}
client, err := redis.NewClient(cfg)
if err != nil {
    panic(err)
}
defer client.Close()
```

### 存取結構體 (JSON)

```go
type PlayerState struct {
    Level int `json:"level"`
    Score int `json:"score"`
}

ctx := context.Background()
state := PlayerState{Level: 10, Score: 5000}

// 寫入 (設定 1 小時過期)
err := client.SetStruct(ctx, "player:1001", state, time.Hour)

// 讀取
var savedState PlayerState
err = client.GetStruct(ctx, "player:1001", &savedState)
```

### 使用分散式鎖

```go
lockKey := "room:101:lock"
lockID := uuid.NewString() // 鎖的唯一持有者 ID

// 嘗試獲取鎖 (鎖 5 秒)
acquired, err := client.AcquireLock(ctx, lockKey, lockID, 5*time.Second)
if err != nil {
    // Redis 錯誤
}
if !acquired {
    // 鎖已被其他人持有
    return
}

// ... 執行關鍵區域代碼 ...

// 釋放鎖
err = client.ReleaseLock(ctx, lockKey, lockID)
```
