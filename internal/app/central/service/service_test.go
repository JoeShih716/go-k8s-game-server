package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/shopspring/decimal"

	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/ports"
	mock_ports "github.com/JoeShih716/go-k8s-game-server/test/mocks/core/ports"
)

func TestCentralService_Login_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_ports.NewMockUserService(ctrl)
	mockWalletSvc := mock_ports.NewMockWalletService(ctrl)
	mockRegistry := mock_ports.NewMockRegistryService(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	svc := NewCentralService(mockUserSvc, mockWalletSvc, mockRegistry, logger)

	ctx := context.Background()
	token := "valid-token"
	userID := "user-123"
	expectedUser := &domain.User{
		ID:      userID,
		Name:    "Test User",
		Balance: decimal.NewFromInt(100),
	}

	// Mock UserSvc.GetUser -> Return user
	mockUserSvc.EXPECT().GetUser(ctx, token).Return(expectedUser, nil)

	// Mock WalletSvc.GetBalance -> Return balance
	mockWalletSvc.EXPECT().GetBalance(ctx, userID).Return(decimal.NewFromInt(1000), nil)

	user, err := svc.Login(ctx, token)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, userID, user.ID)
	// Balance should be updated from WalletService
	assert.Equal(t, decimal.NewFromInt(1000), user.Balance)
}

func TestCentralService_Login_InvalidToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_ports.NewMockUserService(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	svc := NewCentralService(mockUserSvc, nil, nil, logger)

	_, err := svc.Login(context.Background(), "")
	assert.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestCentralService_Login_AutoRegister_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_ports.NewMockUserService(ctrl)
	mockWalletSvc := mock_ports.NewMockWalletService(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	svc := NewCentralService(mockUserSvc, mockWalletSvc, nil, logger)
	ctx := context.Background()
	token := "new-user-token"

	// 1. GetUser returns UserNotFound
	mockUserSvc.EXPECT().GetUser(ctx, token).Return(nil, ports.ErrUserNotFound)

	// 2. CreateGuestUser should be called
	// We use gomock.Any() for the user argument because the ID is generated internally
	mockUserSvc.EXPECT().CreateGuestUser(ctx, token, gomock.Any()).DoAndReturn(func(ctx context.Context, token string, u *domain.User) error {
		assert.NotEmpty(t, u.ID)
		assert.Contains(t, u.Name, "guest-")
		return nil
	})

	// 3. GetBalance (called after registration) - assuming new user has 0 balance or whatever mocked
	mockWalletSvc.EXPECT().GetBalance(ctx, gomock.Any()).Return(decimal.Zero, nil)

	user, err := svc.Login(ctx, token)
	assert.NoError(t, err)
	assert.NotNil(t, user)
}

func TestCentralService_Login_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_ports.NewMockUserService(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	svc := NewCentralService(mockUserSvc, nil, nil, logger)
	ctx := context.Background()
	token := "error-token"

	expectedErr := errors.New("db error")
	mockUserSvc.EXPECT().GetUser(ctx, token).Return(nil, expectedErr)

	_, err := svc.Login(ctx, token)
	assert.ErrorIs(t, err, expectedErr)
}
