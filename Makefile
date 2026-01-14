# 變數定義，排除 .git 資料夾
FIND_EMPTY := find . -type d -empty -not -path "./.git/*"
# 找出所有包含 .gitkeep 且該資料夾內還有其他檔案的 .gitkeep 檔
# 目錄內檔案數量 > 1 代表除了 .gitkeep 還有別的東西
FIND_REDUNDANT_KEEP := find . -name ".gitkeep" -not -path "./.git/*" -exec sh -c 'test $$(ls -A $$(dirname "{}") | wc -l) -gt 1' \; -print

.PHONY: lint lint-fix install-lint keep-add keep-clean

# 安裝最新版 golangci-lint
install-lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 執行檢查
lint:
	golangci-lint run ./...

# 自動修復
lint-fix:
	golangci-lint run --fix ./...

# 為所有空資料夾補上 .gitkeep
keep-add:
	@echo "正在為空資料夾補上 .gitkeep..."
	@$(FIND_EMPTY) -exec touch {}/.gitkeep \;
	@echo "完成！"

# 檢查並刪除「已經不是空資料夾」中的 .gitkeep
keep-clean:
	@echo "正在清理多餘的 .gitkeep 檔案..."
	@find . -name ".gitkeep" -not -path "./.git/*" | while read -r keepfile; do \
		dir=$$(dirname "$$keepfile"); \
		count=$$(ls -A "$$dir" | wc -l); \
		if [ $$count -gt 1 ]; then \
			rm "$$keepfile"; \
			echo "已刪除: $$keepfile"; \
		fi; \
	done
	@echo "清理完成！"