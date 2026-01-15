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
