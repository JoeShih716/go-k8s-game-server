package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config 定義 Redis 連線配置
type Config struct {
	Addr     string // Redis 伺服器地址 (e.g., "localhost:6379")
	Password string // Redis 密碼 (若無則留空)
	DB       int    // 使用的資料庫編號
}

// Client 封裝 redis.Client 以提供更簡易的介面
type Client struct {
	rdb *redis.Client
}

// NewClient 建立並回傳一個新的 Redis 客戶端實例
//
// 參數:
//
//	cfg: Config - Redis 連線配置資訊
//
// 回傳值:
//
//	*Client: 封裝後的 Redis 客戶端實例
//	error: 若連線失敗則回傳錯誤
func NewClient(cfg Config) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 測試連線 (Retry logic)
	ctx := context.Background()
	maxRetries := 10
	retryInterval := 2 * time.Second
	var err error

	for i := 0; i < maxRetries; i++ {
		if err = rdb.Ping(ctx).Err(); err == nil {
			break // Connection successful
		}

		if i < maxRetries-1 {
			fmt.Printf("Failed to connect to Redis (attempt %d/%d): %v. Retrying in %v...\n", i+1, maxRetries, err, retryInterval)
			time.Sleep(retryInterval)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis after %d attempts: %w", maxRetries, err)
	}

	return &Client{rdb: rdb}, nil
}

// Close 關閉 Redis 連線
//
// 回傳值:
//
//	error: 若關閉失敗則回傳錯誤
func (c *Client) Close() error {
	return c.rdb.Close()
}

// SetStruct 將結構體序列化為 JSON 並儲存到 Redis
//
// 參數:
//
//	ctx: context.Context - 上下文
//	key: string - Redis 鍵
//	value: any - 要儲存的結構體 (必須能被 json.Marshal)
//	expiration: ...time.Duration - (選填) 過期時間，若不填則預設為 0 (不過期)
func (c *Client) SetStruct(ctx context.Context, key string, value any, expiration ...time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	var exp time.Duration
	if len(expiration) > 0 {
		exp = expiration[0]
	}

	return c.rdb.Set(ctx, key, data, exp).Err()
}

// GetStruct 從 Redis 讀取 JSON 並反序列化為結構體
//
// 參數:
//
//	ctx: context.Context - 上下文
//	key: string - Redis 鍵
//	dest: any - 目標結構體的指標 (必須能被 json.Unmarshal)
func (c *Client) GetStruct(ctx context.Context, key string, dest any) error {
	val, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return fmt.Errorf("key not found: %s", key)
	} else if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}
	return nil
}

// AcquireLock 嘗試獲取分散式鎖 (使用 SETNX)
//
// 參數:
//
//	ctx: context.Context - 上下文
//	key: string - 鎖的鍵名
//	value: string - 鎖的持有者標識 (通常是 uuid，用於釋放時驗證)
//	expiration: ...time.Duration - (選填) 鎖的自動過期時間，為了安全起見，強烈建議設定。若不填則預設為 0 (需謹慎使用)
//
// 回傳值:
//
//	bool: 是否成功獲取鎖
//	error: Redis 系統錯誤
func (c *Client) AcquireLock(ctx context.Context, key string, value string, expiration ...time.Duration) (bool, error) {
	var exp time.Duration
	if len(expiration) > 0 {
		exp = expiration[0]
	}

	success, err := c.rdb.SetNX(ctx, key, value, exp).Result()
	if err != nil {
		return false, err
	}
	return success, nil
}

// ReleaseLock 釋放分散式鎖
// 只有當鎖的值與傳入的 value 相符時才會刪除，確保不會釋放別人的鎖。
//
// 參數:
//
//	ctx: context.Context - 上下文
//	key: string - 鎖的鍵名
//	value: string - 鎖的持有者標識
func (c *Client) ReleaseLock(ctx context.Context, key string, value string) error {
	// 使用 Lua script 確保原子性檢查與刪除
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	_, err := c.rdb.Eval(ctx, script, []string{key}, value).Result()
	return err
}

// -----------------------------------------------------------
// Basic Commands (String & Key)
// -----------------------------------------------------------

// Set 設定 Key-Value
func (c *Client) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	return c.rdb.Set(ctx, key, value, expiration).Err()
}

// Get 取得 Key-Value
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

// Del 刪除 Key
func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

// Exists 檢查 Key 是否存在
func (c *Client) Exists(ctx context.Context, keys ...string) (bool, error) {
	val, err := c.rdb.Exists(ctx, keys...).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}

// Expire 設定 Key 過期時間
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.rdb.Expire(ctx, key, expiration).Err()
}

// -----------------------------------------------------------
// Set Commands
// -----------------------------------------------------------

// SAdd 加入集合
func (c *Client) SAdd(ctx context.Context, key string, members ...any) error {
	return c.rdb.SAdd(ctx, key, members...).Err()
}

// SMembers 取得集合所有成員
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.rdb.SMembers(ctx, key).Result()
}

// SRem 移除集合成員
func (c *Client) SRem(ctx context.Context, key string, members ...any) error {
	return c.rdb.SRem(ctx, key, members...).Err()
}

// SRandMember 隨機取得一個集合成員
func (c *Client) SRandMember(ctx context.Context, key string) (string, error) {
	return c.rdb.SRandMember(ctx, key).Result()
}

// IsNil 檢查是否為 Redis Key 不存在錯誤
func IsNil(err error) bool {
	return err == redis.Nil
}
