# gRPC Package

`pkg/grpc` 提供了一個高效的 Connection Pool (連線池) 實作，專為微服務架構設計。它解決了重複建立連線的開銷問題，並支援並發存取。

## 功能特性

-   **連線池 (Pooling)**: 針對相同目標地址 (Target) 複用單一底層連線 (`grpc.ClientConn`)。
-   **Lazy Connect**: 僅在第一次請求時建立連線。
-   **自動維護**: 內建 Keepalive 參數，防止連線因閒置被防火牆切斷。
-   **中間件支援**: 支援注入 `UnaryClientInterceptor`，便於整合 Logging, Metrics (Prometheus), OpenTelemetry Tracing 或 Auth Token。

## 使用範例

### 初始化與使用

```go
// 1. 建立 Pool
// 可選: 傳入 Interceptor 實作統一的 Logging 或 Auth
pool := grpcpool.NewPool(
    grpcpool.WithInterceptor(MyLoggingInterceptor),
)
defer pool.Close()

// 2. 獲取連線 (Target 可以是 K8s DNS, e.g., "fishing-service:8080")
conn, err := pool.GetConnection("localhost:50051")
if err != nil {
    log.Fatal(err)
}

// 3. 建立 Client 並呼叫
client := pb.NewGreeterClient(conn)
resp, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: "World"})
```
