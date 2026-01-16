# Kubernetes (K8s) 本地開發指南

這份文件專為 `go-k8s-game-server` 專案設計，幫助您快速理解 K8s 運作流程與常用指令。

## 1. 核心觀念：從程式碼到運作

在 K8s 中，您的程式碼要變成能跑的服務，需要經過三個階段：

1.  **原始碼 (Source Code)**: 您寫的 Go 程式。
2.  **映像檔 (Docker Image)**: 把 Go 程式編譯並打包成一個檔案 (包含 OS 環境)。K8s 不認識 Go code，只認識 Image。
3.  **容器 (Pod)**: K8s 根據 Image 跑起來的實體 (類似一台小虛擬機)。

### 為什麼要 Build?
當您修改了 Go 程式碼 (例如 `handler.go`)，K8s 裡面的 Pod **不會** 自動變更，因為它跑的還是舊的 Image。
所以流程永遠是：
`修改程式碼` -> `Build 新 Image` -> `通知 K8s 重啟 Pod (讀取新 Image)`

---

## 2. 專案的 K8s 檔案結構 (`deploy/k8s/`)

為了簡化管理，我們將檔案分為「本地基礎建設」與「應用程式環境」：

```text
deploy/k8s/
├── local-infra/            # [本地基礎建設] MySQL, Redis (本地開發才需要)
│
└── apps/                   # [應用程式]
    ├── local/              # -> 本地開發環境 (Replicas=1, ENV=local_k8s)
    └── prod/               # -> 正式環境 (Replicas=3, ENV=production)
```

### `deploy/k8s/apps/local/` (本地開發)
| 檔案 | 用途 |
| :--- | :--- |
| **`central.yaml`** | 部署 **Central** 服務 |
| **`connector.yaml`** | 部署 **Connector** 服務 |
| **`stateless-demo.yaml`** | 部署 **Demo** 遊戲邏輯 |

### `deploy/k8s/local-infra/` (本地 Infra)
| 檔案 | 用途 |
| :--- | :--- |
| **`redis.yaml`** | 部署 **Redis** |
| **`mysql.yaml`** | 部署 **MySQL** |

---

## 3. 開發常用指令大全

### A. 建置 (Build)
*修改完程式碼後，必須執行。*

```bash
# 建置 Demo
docker build -t game-server/demo --build-arg SERVICE_PATH=cmd/stateless/demo -f build/package/Dockerfile.localk8s .

# 建置 Connector
docker build -t game-server/connector --build-arg SERVICE_PATH=cmd/connector -f build/package/Dockerfile.localk8s .

# 建置 Central
docker build -t game-server/central --build-arg SERVICE_PATH=cmd/central -f build/package/Dockerfile.localk8s .
```

### B. 部署與更新 (Deploy & Update)
*有了新 Image 後，要讓 K8s 跑起來。*

```bash
# 1. 首次部署 (本地開發)
# 先跑基礎建設
kubectl apply -f deploy/k8s/local-infra/
# 再跑應用程式 (Local版)
kubectl apply -f deploy/k8s/apps/local/

# (若要部署到正式環境)
# kubectl apply -f deploy/k8s/apps/prod/

# 2. 熱更新 (只修改了程式碼，已 Build 好 Image)
# 這會讓舊 Pod 變為 Terminating，新 Pod 變為 Running (Rolling Update)
kubectl rollout restart deployment/stateless-demo
kubectl rollout restart deployment/connector
```

### C. 檢查狀態 (Check Status)

```bash
# 查看所有 Pod 狀態 (加 -w 可以持續監控)
kubectl get pods -o wide

# 查看特定 Pod 的 Log (除錯用)
# -f 代表 follow (持續輸出)，-l app=xxx 代表選取特定標籤
kubectl logs -f -l app=central
kubectl logs -f -l app=stateless-demo
kubectl logs -f -l app=connector
```

### D. 進入容器與資料庫 (Exec)

```bash
# 進入 Redis 查看資料
kubectl exec -it $(kubectl get pod -l app=redis -o jsonpath='{.items[0].metadata.name}') -- redis-cli

# 進入 Pod 內部 Shell (例如檢查檔案是否存在)
kubectl exec -it <pod-name> -- sh
```

---

## 4. 常見情境 Q&A

**Q: 我改了 config/local_k8s.yaml，為什麼重啟沒生效？**
**A:** 因為 Config 是被 `COPY` 到 Docker Image 裡面的。
**解法**: 您必須執行 **Step A (docker build)** 重新打包 Image，然後 **Step B (rollout restart)**。

**Q: 我改了 deploy/k8s/xxx.yaml (例如加了環境變數)，要怎麼生效？**
**A:** 這是修改「設計圖」，不需要重 build image。
**解法**: 直接執行 `kubectl apply -f deploy/k8s/xxx.yaml`。

**Q: 為什麼 `kubectl get pods` 顯示 CrashLoopBackOff？**
**A:** 代表程式啟動失敗 (Panic 或 Exit)。
**解法**: 用 `kubectl logs <pod-name>` 看錯誤訊息。常見原因是 config 讀不到、連不上 DB、或是 Port 衝突。
