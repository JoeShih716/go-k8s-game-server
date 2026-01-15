package router

import (
	"errors"
	"fmt"
	"sync"
)

// ErrServiceNotFound 表示找不到請求的服務
var ErrServiceNotFound = errors.New("service not found")

// Discovery 定義服務發現介面
// 負責根據服務名稱和類型查找可用的服務實例地址
type Discovery interface {
	// GetServiceAddr 根據服務名稱取得地址
	GetServiceAddr(serviceName string) (string, error)
}

// StaticDiscovery 靜態服務發現實作 (讀取 Config)
type StaticDiscovery struct {
	services map[string]string
	mu       sync.RWMutex
}

// NewStaticDiscovery 建立一個基於靜態設定的 Discovery
func NewStaticDiscovery(services map[string]string) *StaticDiscovery {
	return &StaticDiscovery{
		services: services,
	}
}

// GetServiceAddr 實作 Discovery 介面
func (d *StaticDiscovery) GetServiceAddr(serviceName string) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	addr, ok := d.services[serviceName]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrServiceNotFound, serviceName)
	}
	return addr, nil
}
