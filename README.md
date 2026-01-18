# Go K8s Game Server

[![Go Version](https://img.shields.io/badge/go-1.25-blue)](https://go.dev/)
[![CI Status](https://github.com/JoeShih716/go-k8s-game-server/actions/workflows/ci.yaml/badge.svg)](https://github.com/JoeShih716/go-k8s-game-server/actions)
[![Docker](https://img.shields.io/badge/docker-ready-blue)](https://www.docker.com/)
[![Kubernetes](https://img.shields.io/badge/kubernetes-ready-blue)](https://kubernetes.io/)

打造一個生產級的雲原生遊戲伺服器樣板 (Boilerplate) 與框架。
本專案採用 **"Engine & Cartridge"** 設計模式，將「伺服器底層 (Engine)」與「遊戲邏輯 (Content)」分離，並引入 **Game Framework** 簡化開發。

## 核心特色 (Features)

- **混合架構 (Hybrid Architecture)**: 同時支援 Stateless (無狀態, e.g. Slots) 與 Stateful (有狀態, e.g. Battle, MMO) 服務。
- **高效通訊 (High Performance)**:
    - **WebSocket**: 針對高併發優化的 Connector 網關。
    - **gRPC**: 服務間通訊 (Inter-Service Communication)，包含雙向即時訊息推送。
- **Game Framework**:
    - **Bootstrap**: 統一的生命週期管理 (Config, Logger, Graceful Shutdown)。
    - **Session Management**: 自動處理玩家連線狀態 (Transient for Stateless, Persistent for Stateful)。
    - **Abstracted RPC**: 簡化底層 gRPC 複雜度，開發者只需實作 `GameHandler` 介面。
- **雲原生就緒 (Cloud Native Ready)**:
    - **Kubernetes**: 完整的 K8s 部署清單 (Manifests) 與環境變數配置。
    - **Observability**: 結構化日誌 (Slog) 與健康檢查 (Health Checks)。
- **開發體驗 (Developer Experience)**:
    - **Air**: 支援 Docker Compose 環境下的熱重載 (Hot Reload)。
    - **CI/CD**: 整合 GitHub Actions 進行自動化 Lint, Test, Build。

## 架構概覽 (Architecture)

### 核心服務
1.  **Connector (智慧網關)**:
    - 處理 WebSocket 長連線。
    - 負責將客戶端封包路由至後端遊戲服務。
    - 支援 `ConnectorRPC`，允許遊戲服務主動推送訊息 (Push) 或踢除玩家 (Kick)。
2.  **Central (中央控制)**:
    - 服務註冊與發現 (Service Registry via Redis)。
    - 玩家驗證 (Authentication) 與錢包整合 (Mock Wallet)。
3.  **Game Services (遊戲邏輯)**:
    - **Stateless Demo**: 實作類似老虎機的 Request-Response 邏輯，使用短暫 Session。
    - **Stateful Demo**: 實作類似戰鬥房的 Persistent Session 邏輯，支援廣播與狀態維護。

### 目錄結構 (Directory Structure)

```text
go-k8s-game-server/
├── cmd/                        # [部署入口] (Main Applications)
│   ├── central/                # -> 中央服務
│   ├── connector/              # -> 網關服務
│   ├── stateless/              # -> 無狀態遊戲入口 (Demo)
│   └── stateful/               # -> 有狀態遊戲入口 (Demo)
│
├── internal/                   # [內部核心] (Private Code)
│   ├── app/                    # -> 具體業務邏輯 (Handler, Usecase)
│   ├── config/                 # -> 配置加載 (Config.yaml + Env Override)
│   ├── pkg/                    # -> 專案內共用套件
│   │   ├── bootstrap/          #    -> 應用啟動器 (App Lifecycle)
│   │   └── framework/          #    -> 遊戲伺服器框架 (Session, Server)
│   └── ...
│
├── pkg/                        # [通用工具庫] (Public Libraries)
│   ├── grpc/                   # -> gRPC Client Pool
│   ├── mysql/                  # -> MySQL Client (GORM)
│   ├── redis/                  # -> Redis Client
│   └── wss/                    # -> WebSocket Framework
│
├── api/proto/                  # [通訊協議] (Protocol Buffers)
│   ├── centralRPC/             # -> Game <-> Central
│   ├── connectorRPC/           # -> Game -> Connector (Push/Kick)
│   └── gameRPC/                # -> Connector -> Game (Logic)
│
└── deploy/                     # [部署配置]
    ├── k8s/                    # -> Kubernetes Manifests
    └── docker-compose.yaml     # -> Local Development
```

## 快速開始 (Getting Started)

### 先決條件 (Prerequisites)
- Docker & Docker Compose
- Go 1.23+
- Make (Optional)

### 本地開發 (Docker Compose)
這是最快的啟動方式，支援熱重載。

1.  **啟動服務**:
    ```bash
    docker-compose up --build
    ```
    此指令會啟動 Redis, MySQL, Central, Connector 以及 Demo Services。

2.  **測試連線**:
    開啟瀏覽器訪問 `http://localhost:8080` (內建 WebSocket 測試工具)。
    - **Login**: 輸入 UserID。
    - **Connect**: 建立 WebSocket 連線。
    - **Enter Game**: 輸入 GameID (Stateless: 10000, Stateful: 20000)。

### 本地 Kubernetes 開發 (Local K8s)
支援將應用部署至 Orbstack, Docker Desktop K8s 或 Kind。

1.  **建置映像檔**:
    ```bash
    # 支援透過 build args 指定不同服務入口
    docker build -t game-server/central --build-arg SERVICE_PATH=cmd/central -f build/package/Dockerfile.localk8s .
    docker build -t game-server/connector --build-arg SERVICE_PATH=cmd/connector -f build/package/Dockerfile.localk8s .
    docker build -t game-server/stateless-demo --build-arg SERVICE_PATH=cmd/stateless/demo -f build/package/Dockerfile.localk8s .
    docker build -t game-server/stateful-demo --build-arg SERVICE_PATH=cmd/stateful/demo -f build/package/Dockerfile.localk8s .
    ```

2.  **部署**:
    ```bash
    # 1. 基礎設施 (MySQL, Redis)
    kubectl apply -f deploy/k8s/local-infra/

    # 2. 應用程式
    kubectl apply -f deploy/k8s/apps/local/
    ```

3.  **配置更新**:
    所有服務支援完整的環境變數覆寫，請參考 Kubernetes Yaml 中的 `env` 區塊。
    - `PORT`: HTTP/WebSocket Port
    - `GRPC_PORT`: gRPC Port
    - `MYSQL_HOST`, `MYSQL_USER`...: 資料庫連線資訊
    - `REDIS_ADDR`: Redis 連線資訊

## 開發指南 (Development Guide)

### 新增一個遊戲服務
1.  在 `cmd/` 下建立新目錄 (e.g. `cmd/mygame/main.go`)。
2.  實作 `internal/pkg/framework.GameHandler` 介面：
    - `OnJoin(ctx, session, payload)`
    - `OnQuit(ctx, session)`
    - `OnMessage(ctx, session, payload)`
3.  在 `main.go` 中使用 `bootstrap.RunGameServer` 啟動：
    ```go
    func main() {
        config := bootstrap.GameServerConfig{
            ServiceName: "my-game",
            ServiceType: proto.ServiceType_STATELESS, // 或 STATEFUL
            GameIDs:     []int32{30000},
            DefaultPort: 9090,
        }
        handler := mygame.NewHandler()
        bootstrap.RunGameServer(config, handler)
    }
    ```

### CI/CD
本專案包含 GitHub Actions Workflow (`.github/workflows/ci.yaml`)，在 Push 或 PR 時自動執行：
- **Lint**: `golangci-lint`
- **Test**: `go test -race ./...`
- **Build**: 驗證所有微服務編譯正常。
