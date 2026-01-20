package handler

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/JoeShih716/go-k8s-game-server/api/proto"
	"github.com/JoeShih716/go-k8s-game-server/api/proto/centralRPC"
	"github.com/JoeShih716/go-k8s-game-server/internal/app/connector/protocol"
	"github.com/JoeShih716/go-k8s-game-server/internal/app/connector/session"
	mock_handlers "github.com/JoeShih716/go-k8s-game-server/test/mocks/handlers"
	mock_wss "github.com/JoeShih716/go-k8s-game-server/test/mocks/pkg/wss"
)

func TestWebsocketHandler_OnConnect(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mocks
	mockWssClient := mock_wss.NewMockClient(ctrl)
	mockPool := mock_handlers.NewMockGRPCPool(ctrl)
	mockCentral := mock_handlers.NewMockCentralClient(ctrl)
	mgr := session.NewManager()

	handler := NewWebsocketHandler(mgr, mockPool, mockCentral, "connector-1")

	// Expectation
	mockWssClient.EXPECT().ID().Return("sess-1").AnyTimes()
	// Should set login timer
	mockWssClient.EXPECT().SetTag("login_timer", gomock.Any())

	// Act
	handler.OnConnect(mockWssClient)

	// Assert
	assert.Equal(t, int64(1), mgr.Count())
	sess, ok := mgr.Get("sess-1")
	assert.True(t, ok)
	assert.NotNil(t, sess)
}

func TestWebsocketHandler_OnDisconnect(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWssClient := mock_wss.NewMockClient(ctrl)
	mockPool := mock_handlers.NewMockGRPCPool(ctrl)
	mockCentral := mock_handlers.NewMockCentralClient(ctrl)
	mgr := session.NewManager()

	handler := NewWebsocketHandler(mgr, mockPool, mockCentral, "connector-1")

	// Setup Session
	mockWssClient.EXPECT().ID().Return("sess-1").AnyTimes()
	mockWssClient.EXPECT().SetTag("login_timer", gomock.Any())
	handler.OnConnect(mockWssClient)
	assert.Equal(t, int64(1), mgr.Count())

	// Expectation for Disconnect
	// cleanup timers
	mockWssClient.EXPECT().GetTag("login_timer").Return(nil, false)
	mockWssClient.EXPECT().GetTag("enter_game_timer").Return(nil, false)

	// Check routing tags (assume none for simple disconnect)
	mockWssClient.EXPECT().GetTag("target_endpoint").Return(nil, false)
	mockWssClient.EXPECT().GetTag("current_game_id").Return(nil, false)

	// Act
	handler.OnDisconnect(mockWssClient)

	// Assert
	assert.Equal(t, int64(0), mgr.Count())
}

func TestWebsocketHandler_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWssClient := mock_wss.NewMockClient(ctrl)
	mockPool := mock_handlers.NewMockGRPCPool(ctrl)
	mockCentral := mock_handlers.NewMockCentralClient(ctrl)
	mgr := session.NewManager()

	handler := NewWebsocketHandler(mgr, mockPool, mockCentral, "connector-1")

	// Setup: Session exists
	mockWssClient.EXPECT().ID().Return("sess-1").AnyTimes()
	// Replace catch-all with specific expectation for setup
	mockWssClient.EXPECT().SetTag("login_timer", gomock.Any())
	handler.OnConnect(mockWssClient)

	// Prepare Login Payload
	loginPayload := []byte(`{"token":"valid-token"}`)
	msg, _ := json.Marshal(protocol.Envelope{
		Action:  protocol.ActionLogin,
		Payload: json.RawMessage(loginPayload),
	})

	// Login Expectations
	mockWssClient.EXPECT().GetTag("user_id").Return(nil, false) // Check not logged in
	mockWssClient.EXPECT().GetTag("login_timer").Return(nil, false) // Stop timer

	// Central Login verification
	mockCentral.EXPECT().Login(gomock.Any(), "valid-token").Return(&centralRPC.LoginResponse{
		UserId:   "user-100",
		Nickname: "TestUser",
		Balance:  "1000",
	}, nil)

	// Success Response
	mockWssClient.EXPECT().SetTag("user_id", "user-100")
	mockWssClient.EXPECT().SendMessage(gomock.Any()).DoAndReturn(func(msg string) error {
		assert.Contains(t, msg, "user-100")
		assert.Contains(t, msg, "1000") // Balance
		return nil
	})

	// Start Enter Game Timer
	mockWssClient.EXPECT().SetTag("enter_game_timer", gomock.Any())

	// Act
	handler.OnMessage(mockWssClient, msg)
}

func TestWebsocketHandler_EnterGame(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWssClient := mock_wss.NewMockClient(ctrl)
	mockPool := mock_handlers.NewMockGRPCPool(ctrl)
	mockCentral := mock_handlers.NewMockCentralClient(ctrl)
	mgr := session.NewManager()

	handler := NewWebsocketHandler(mgr, mockPool, mockCentral, "connector-1")

	// Setup: Session exists and Logged In
	mockWssClient.EXPECT().ID().Return("sess-1").AnyTimes()
	mockWssClient.EXPECT().SetTag("login_timer", gomock.Any()) // Setup
	handler.OnConnect(mockWssClient)

	// Payload
	payload := []byte(`{"game_id": 1001}`)
	msg, _ := json.Marshal(protocol.Envelope{
		Action:  protocol.ActionEnterGame,
		Payload: payload,
	})

	// Expectations
	mockWssClient.EXPECT().GetTag("current_game_id").Return(nil, false) // Not in game
	mockWssClient.EXPECT().GetTag("user_id").Return("user-100", true) // Logged in
	mockWssClient.EXPECT().GetTag("enter_game_timer").Return(nil, false) // Stop timer

	// Central GetRoute
	mockCentral.EXPECT().GetRoute(gomock.Any(), int32(1001)).Return("node-1:8090", proto.ServiceType_STATELESS, nil)

	// Game Server OnPlayerJoin (gRPC)
	// We need to mock the gRPC Client.
	// PROBLEM: NewGameRPCClient takes a *grpc.ClientConn. We can't easily mock the client returned by NewGameRPCClient
	// without wrapping the generated gRPC client creation logic or networking.
	// For this test, verifying interaction with GRPCPool is often the boundary unless we mock NewGameRPCClient.
	// However, GetConnection returns *grpc.ClientConn. We can't mock methods on *grpc.ClientConn.
	//
	// Implication: The current design (GRPCPool returns *ClientConn) makes testing gRPC logic hard without a real connection or strict integration test.
	// To unit test this purely, we usually wrap the GameRPCClient creation or the client itself behind an interface.
	//
	// Given the constraints and current scope, I will skip verifying the actual gRPC call details (OnPlayerJoin)
	// unless I refactor NewGameRPCClient usage.
	//
	// Alternative: Mock GRPCPool to return error, to verifies error handling path at least.
	// Or accept that this test will fail if it tries to dial "node-1:8090".
	//
	// Let's modify the expectation to fail at GRPCPool.GetConnection for now to verify the flow up to that point,
	// because fully mocking gRPC client requires more refactoring (Abstract Factory for RPC Clients).

	mockPool.EXPECT().GetConnection("node-1:8090").Return(nil, fmt.Errorf("mock connection error"))

	// Expect Error Response to Client
	mockWssClient.EXPECT().SendMessage(gomock.Any()).Do(func(msg string) {
		assert.Contains(t, msg, "Game Server Unavailable")
	})

	// Act
	handler.OnMessage(mockWssClient, msg)
}
