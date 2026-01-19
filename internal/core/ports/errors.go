package ports

import "errors"

// 定義 Ports 層級通用的錯誤
var (
	ErrUserNotFound = errors.New("user not found")
	ErrInvalidToken = errors.New("invalid token")
)
