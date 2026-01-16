# Go K8s Game Server

打造一個生產級的雲原生遊戲伺服器樣板 (Boilerplate) 與框架。
本專案採用 **"Engine & Cartridge"** 設計模式，將「伺服器底層 (Engine)」與「遊戲邏輯 (Content)」分離。

## 核心架構 (Architecture)

本專案支援混合架構，同時容納無狀態與有狀態服務：

- **Stateless Services (無狀態)**:
    - **Central (中央服務)**: 整個集群的大腦。負責服務註冊 (Registry)、發現 (Discovery) 與權限驗證 (Auth)。
    - **Slots / Demo**: 透過 gRPC 處理請求，可水平擴展 (Horizontal Scaling)。
- **Stateful Services (有狀態)**:
    - **Fishing / Battle**: 負責高頻率的遊戲邏輯 (Tick Loop)。
    - **Sticky Routing**: 玩家連線固定於特定 Pod (StatefulSet)。
- **Connector (智慧網關)**:
    - 負責 WebSocket 長連線維護。
    - 根據 `Routing Strategy` 將封包轉發至正確的後端服務。

## 目錄結構 (Directory Structure)

```text
go-k8s-game-server/
├── cmd/                        # [部署入口]
│   ├── central/                # -> 中央服務 (Service Registry)
│   ├── connector/              # -> 網關服務 (WebSocket, Router)
│   ├── stateless/              # -> 無狀態遊戲入口
│   │   └── demo/               #    (Stateless Framework 範例)
│   └── stateful/               # -> 有狀態遊戲入口
│
├── internal/                   # [核心代碼]
│   ├── central/                # -> Central 實作細節
│   ├── connector/              # -> Connector 實作 (Router, Handler)
│   ├── core/                   # -> 共享介面 (GameRoom, User Entity)
│   └── stateless/              # -> 遊戲邏輯實作 (Usecase)
│
├── pkg/                        # [通用工具庫]
│   ├── grpc/                   # -> gRPC Client Pool
│   ├── mysql/                  # -> MySQL Client (GORM + Resilience)
│   ├── redis/                  # -> Redis Client (PubSub, Lock, Resilience)
│   └── wss/                    # -> WebSocket Framework (Hub, Pump)
│
├── api/proto/                  # [通訊協議]
│   ├── central.proto           # -> Central 服務接口
│   ├── common.proto            # -> 通用封包結構
│   └── routing.proto           # -> 路由規則定義
│
└── deploy/                     # [K8s Manifests & Dockerfiles]
```

## 快速開始 (Getting Started)

### 本地開發 (Local Development)

本專案使用 **Docker Compose** 搭配 **Air** 進行熱重載 (Hot Reload) 開發。

1. **啟動服務**:
    ```bash
    docker-compose up --build
    ```
    這將會啟動：
    - MySQL & Redis
    - Central Service (:9003)
    - Connector Service (:8080)
    - Stateless Demo Service (:9001)

2. **驗證**:
    - **Client**: 打開瀏覽器訪問 `http://localhost:8080` (內建測試用 HTML Client)。
    - **Logs**: 觀察 Console，應能看到 `stateless-demo` 成功向 `central` 註冊的日誌。

## 基礎建設 (Infrastructure)

詳見 `pkg/` 下各套件的說明文檔：

- [MySQL](pkg/mysql/README.md): 具備連線池與斷線重連機制的 GORM 封裝。
- [Redis](pkg/redis/README.md): 支援泛型存取、分散式鎖與自動重試。
- [gRPC](pkg/grpc/README.md): 高效能 gRPC 連線池 (Client Pool)。
- [WebSocket](pkg/wss/README.md): 內建 Hub 模式與安全防護的 WebSocket 框架。

## 本地 Kubernetes 開發環境 (Local K8s)

除了一般的 Docker Compose，本專案也支援本地 Kubernetes (如 via Orbstack, Docker Desktop, Kind) 部署。

### 1. 初次部署 (Initial Setup)

若您是第一次啟動 K8s 環境，請執行以下步驟建置所有基礎映像檔並部署：

```bash
# 1. 建置所有服務映像檔
docker build -t game-server/central --build-arg SERVICE_PATH=cmd/central -f build/package/Dockerfile.localk8s .
docker build -t game-server/connector --build-arg SERVICE_PATH=cmd/connector -f build/package/Dockerfile.localk8s .
docker build -t game-server/demo --build-arg SERVICE_PATH=cmd/stateless/demo -f build/package/Dockerfile.localk8s .

# 2. 部署到 Kubernetes (包含 MySQL, Redis, Services)
kubectl apply -f deploy/k8s/
```

### 2. 日常開發流程 (Daily Workflow)

與 Docker Compose 不同，K8s 部署**不會**自動監測檔案變更。開發流程如下：

1. **修改程式碼** (e.g. `handler.go`)
2. **建置映像檔 (Build Images)**:
    ```bash
    # 使用專用的 localk8s Dockerfile，將 Config 封裝進去
    docker build -t game-server/demo --build-arg SERVICE_PATH=cmd/stateless/demo -f build/package/Dockerfile.localk8s .
    ```
3. **滾動更新 (Rollout Restart)**:
    ```bash
    # 通知 K8s 使用新映像檔重啟 Pod
    kubectl rollout restart deployment/stateless-demo
    ```
4. **驗證**: 使用 `kubectl get pods` 觀察狀態。

### 2. 架構設計：客戶端負載平衡 (Client-Side Load Balancing)

在本專案的 K8s 部署中，我們對於 **gRPC** 通訊採取了特殊的路由策略：

*   **Stateless 服務 (Demo)**：
    *   **不使用 Kubernetes ClusterIP (Service IP)** 做核心路由。因為 ClusterIP 是 L4 Load Balancer，無法對長連線 (gRPC/HTTP2) 做 Request-Level 的負載平衡 (會導致 Sticky Connection 問題)。
    *   **採用 Client-Side Round-Robin**：
        1.  服務啟動時，透過 Downward API 獲取自身 **Pod IP**，並註冊到 Central (Redis)。
        2.  Connector (Client) 從 Central 獲取可用 Pod IP 列表。
        3.  Connector 自行維護連線池 (`grpcPool`)，並對 Stateless 請求實作 **Packet-Level Round-Robin** (意即：每一個封包都重新選擇一個 Pod IP)。

*   **Stateful 服務 (Sticky)**：
    *   維持 **Sticky Session** 機制。Connector 在玩家 `Login/Enter` 後鎖定特定 Pod IP，後續訊息固定轉發，確保狀態一致。

這種設計讓我們不需引入 Istio 等重型 Service Mesh，即可在 Go 應用層實現高效、靈活的 gRPC 負載平衡。

> 詳細操作指令請參閱 [Docs: Kubernetes 開發指南](docs/K8S_GUIDE.md)

### 3. 清理環境 (Clean Up)

若要刪除所有部署的服務 (包含 Pods, Services, ConfigMaps 等)，主要用於重置環境或釋放資源：

```bash
kubectl delete -f deploy/k8s/
```
這只會刪除 K8s 資源，不會刪除 Docker Images。若要連同 Image 一起清理，可額外執行 `docker rmi`。
