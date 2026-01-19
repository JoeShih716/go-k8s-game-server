package service_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"

	"github.com/JoeShih716/go-k8s-game-server/internal/app/central/service"
	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
	mock_ports "github.com/JoeShih716/go-k8s-game-server/test/mocks/core/ports"
)

func TestCentralService_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mock_ports.NewMockUserRepository(ctrl)
	mockWalletSvc := mock_ports.NewMockWalletService(ctrl)
	mockRegistry := mock_ports.NewMockServiceRegistry(ctrl)

	svc := service.NewCentralService(mockUserRepo, mockWalletSvc, mockRegistry, slog.Default())

	t.Run("Login Success (Existing User)", func(t *testing.T) {
		userID := "user-123"
		mockUser := &domain.User{ID: userID, Name: "TestUser", Balance: decimal.Zero}

		// Expect UserRepo.GetByID
		mockUserRepo.EXPECT().GetByID(gomock.Any(), userID).Return(mockUser, nil)

		// Expect WalletSvc.GetBalance (returns 100)
		mockWalletSvc.EXPECT().GetBalance(gomock.Any(), userID).Return(decimal.NewFromInt(100), nil)

		user, err := svc.Login(context.Background(), userID)

		if err != nil {
			t.Fatalf("Login failed: %v", err)
		}
		if user.ID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, user.ID)
		}
		if !user.Balance.Equal(decimal.NewFromInt(100)) {
			t.Errorf("Expected balance 100, got %d", user.Balance)
		}
	})

	t.Run("Login Success (New User Auto-Register)", func(t *testing.T) {
		userID := "new-user"

		// Expect GetByID -> Not Found (nil, nil)
		mockUserRepo.EXPECT().GetByID(gomock.Any(), userID).Return(nil, nil)

		// Expect Create
		mockUserRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

		// Expect GetBalance from wallet (e.g. initial gift)
		mockWalletSvc.EXPECT().GetBalance(gomock.Any(), userID).Return(decimal.NewFromInt(500), nil)

		user, err := svc.Login(context.Background(), userID)

		if err != nil {
			t.Fatalf("Login failed: %v", err)
		}
		if user.ID != userID {
			t.Errorf("Expected new user ID %s, got %s", userID, user.ID)
		}
		if !user.Balance.Equal(decimal.NewFromInt(500)) {
			t.Errorf("Expected balance 500, got %d", user.Balance)
		}
	})

	t.Run("Login Failed (Invalid Token)", func(t *testing.T) {
		_, err := svc.Login(context.Background(), "invalid-token")
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("Login Failed (Repo Error)", func(t *testing.T) {
		userID := "error-user"
		mockUserRepo.EXPECT().GetByID(gomock.Any(), userID).Return(nil, errors.New("db error"))

		_, err := svc.Login(context.Background(), userID)
		if err == nil {
			t.Error("Expected error from Repo")
		}
	})
}
