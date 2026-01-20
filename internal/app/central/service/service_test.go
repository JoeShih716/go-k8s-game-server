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
	"github.com/JoeShih716/go-k8s-game-server/internal/core/ports"
	mock_ports "github.com/JoeShih716/go-k8s-game-server/test/mocks/core/ports"
)

func TestCentralService_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_ports.NewMockUserService(ctrl)
	mockWalletSvc := mock_ports.NewMockWalletService(ctrl)
	mockRegistry := mock_ports.NewMockRegistryService(ctrl)

	svc := service.NewCentralService(mockUserSvc, mockWalletSvc, mockRegistry, slog.Default())

	t.Run("Login Success (Existing User)", func(t *testing.T) {
		token := "user-token-123"
		userID := "user-123"
		mockUser := &domain.User{ID: userID, Name: "TestUser", Balance: decimal.Zero}

		// Expect UserService.GetUser -> Returns User
		mockUserSvc.EXPECT().GetUser(gomock.Any(), token).Return(mockUser, nil)

		// Expect WalletSvc.GetBalance (returns 100)
		mockWalletSvc.EXPECT().GetBalance(gomock.Any(), userID).Return(decimal.NewFromInt(100), nil)

		user, err := svc.Login(context.Background(), token)

		if err != nil {
			t.Fatalf("Login failed: %v", err)
		}
		if user.ID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, user.ID)
		}
		if !user.Balance.Equal(decimal.NewFromInt(100)) {
			t.Errorf("Expected balance 100, got %s", user.Balance)
		}
	})

	t.Run("Login Success (Guest Auto-Register)", func(t *testing.T) {
		token := "guest-token-123"

		// Expect GetUser -> Not Found
		mockUserSvc.EXPECT().GetUser(gomock.Any(), token).Return(nil, ports.ErrUserNotFound)

		// Expect CreateGuestUser
		mockUserSvc.EXPECT().CreateGuestUser(gomock.Any(), token, gomock.Any()).DoAndReturn(func(ctx context.Context, t string, u *domain.User) error {
			// Simulate creating guest user
			u.ID = "new-guest-id"
			u.Name = "Guest-new-guest-id"
			return nil
		})

		// Expect GetBalance (returns 0 for new guest)
		mockWalletSvc.EXPECT().GetBalance(gomock.Any(), "new-guest-id").Return(decimal.Zero, nil)

		user, err := svc.Login(context.Background(), token)

		if err != nil {
			t.Fatalf("Login failed: %v", err)
		}
		if user.ID != "new-guest-id" {
			t.Errorf("Expected new guest ID, got %s", user.ID)
		}
	})

	t.Run("Login Failed (Invalid Token)", func(t *testing.T) {
		_, err := svc.Login(context.Background(), "")
		if err != domain.ErrInvalidToken {
			t.Errorf("Expected ErrInvalidToken, got %v", err)
		}
	})

	t.Run("Login Failed (Repo Error)", func(t *testing.T) {
		token := "error-token"
		// Expect GetUser -> DB Error
		mockUserSvc.EXPECT().GetUser(gomock.Any(), token).Return(nil, errors.New("db connection error"))

		_, err := svc.Login(context.Background(), token)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}
