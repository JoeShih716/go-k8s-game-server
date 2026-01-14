# Protocol Buffers (Protobuf) & API Definitions

這裡存放了所有遊戲伺服器微服務之間的「溝通合約」。我們使用 Google 的 Protocol Buffers (proto3) 來定義這些合約。

## 目錄結構
-   **`common.proto`**: 通用的資料結構 (例如: 封包 Header, 錯誤碼 ErrorCode)。
-   **`gateway.proto`**: Gateway (Connector) 相關的指令 (例如: 踢人, 廣播)。
-   **`routing.proto`**: 路由相關定義 (例如: 服務類型, 路由元數據)。

## 為什麼要用 Protobuf?
1.  **效能**: 序列化後的二進位資料比 JSON 小得多，解析速度快數倍。(這對即時遊戲至關重要)
2.  **強型別**: `int32`, `string`, `bool` 定義清楚，不會有 JSON `100` vs `"100"` 的模糊地帶。
3.  **自動生成**: 透過 `protoc` 編譯器，自動產生 Go (或 C#, Unity) 代碼，省去寫 Boilerplate 的時間。

## 自動生成的檔案 (`*.pb.go`)

當您執行 `make gen-proto` 後，每個 `.proto` 檔案會產生對應的 `.pb.go` 檔案。

### 這些檔案包含什麼？
1.  **Struct 定義**: 對應 `message`。
2.  **Getter 方法**: 例如 `GetRoomId()`，這能安全地存取欄位 (避免 nil pointer)。**請務必習慣使用 Getter！**
3.  **序列化方法**: `Marshal()` / `Unmarshal()` 的底層實作。

> ⚠️ **注意**: 永遠不要手動修改 `*.pb.go` 檔案。如果您需要修改欄位，請改 `.proto` 檔然後重跑 `make gen-proto`。

## 如何新增協議？
1.  在 `api/proto/` 建立或修改 `.proto` 檔案。
2.  在終端機執行：
    ```bash
    make gen-proto
    ```
3.  檢查生成的 `.pb.go` 是否更新。

## 環境建置 (如果 `make gen-proto` 失敗)
請參考根目錄 `Makefile` 的 `install-tools` 指令，或手動安裝：
```bash
brew install protobuf
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```
