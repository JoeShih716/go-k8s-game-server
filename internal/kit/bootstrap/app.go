package bootstrap

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/JoeShih716/go-k8s-game-server/internal/kit/config"
)

// App 封裝了應用程式的基礎組件
type App struct {
	Name   string
	Config *config.Config
	Logger *slog.Logger
}

// NewApp 建立一個新的應用程式實例
//
// 1. 初始化 Default Logger
// 2. 載入 Config (config.yaml + Env Override)
func NewApp(appName string) *App {
	// 1. 初始化基礎 Logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. 載入設定
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// 3. 根據環境重新配置 Logger
	// Production -> JSON (Structured Logging)
	// Others     -> Text (Readable)
	var handler slog.Handler

	switch cfg.App.Env {
	case "production", "prod":
		handler = slog.NewJSONHandler(os.Stdout, nil)
	default:
		handler = slog.NewTextHandler(os.Stdout, nil)
	}
	logger = slog.New(handler)
	slog.SetDefault(logger) // 更新 Default Logger

	return &App{
		Name:   appName,
		Config: cfg,
		Logger: logger,
	}
}

// Run 啟動應用程式並等待停止信號
//
// startFunc: 啟動服務的邏輯 (Blocking operation like http.ListenAndServe or grpc.Serve)
// cleanupFunc: 收到停止信號後的清理邏輯
func (a *App) Run(startFunc func() error, cleanupFunc func()) {
	// 背景啟動服務
	go func() {
		a.Logger.Info("Starting service", "app", a.Name, "env", a.Config.App.Env)
		if err := startFunc(); err != nil {
			a.Logger.Error("Service startup failed", "error", err)
			os.Exit(1)
		}
	}()

	// 等待停止信號
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	a.Logger.Info("Shutting down service...", "app", a.Name)
	if cleanupFunc != nil {
		cleanupFunc()
	}
	a.Logger.Info("Service exited", "app", a.Name)
}
