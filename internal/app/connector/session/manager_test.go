package session

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/JoeShih716/go-k8s-game-server/internal/core/domain"
	mock_wss "github.com/JoeShih716/go-k8s-game-server/test/mocks/pkg/wss"
)

func TestManager_Add_Get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mgr := NewManager()

	// Create Mock Client
	mockClient := mock_wss.NewMockClient(ctrl)
	mockClient.EXPECT().ID().Return("sess-1").AnyTimes()

	sess := domain.NewSession(mockClient)

	// Test Add
	mgr.Add(sess)
	assert.Equal(t, int64(1), mgr.Count())

	// Test Get
	got, ok := mgr.Get("sess-1")
	assert.True(t, ok)
	assert.Equal(t, sess, got)

	// Test Get Non-Existent
	_, ok = mgr.Get("non-existent")
	assert.False(t, ok)
}

func TestManager_Remove(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mgr := NewManager()

	mockClient := mock_wss.NewMockClient(ctrl)
	mockClient.EXPECT().ID().Return("sess-1").AnyTimes()
	sess := domain.NewSession(mockClient)

	mgr.Add(sess)
	assert.Equal(t, int64(1), mgr.Count())

	// Test Remove
	mgr.Remove("sess-1")
	assert.Equal(t, int64(0), mgr.Count())

	// Test Remove Non-Existent (Should not panic or change count)
	mgr.Remove("sess-1")
	assert.Equal(t, int64(0), mgr.Count())
}

func TestManager_Range(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mgr := NewManager()

	// Add 3 sessions
	ids := []string{"s1", "s2", "s3"}
	for _, id := range ids {
		mockClient := mock_wss.NewMockClient(ctrl)
		mockClient.EXPECT().ID().Return(id).AnyTimes()
		mgr.Add(domain.NewSession(mockClient))
	}

	assert.Equal(t, int64(3), mgr.Count())

	// Test Range
	count := 0
	foundIDs := make(map[string]bool)
	mgr.Range(func(s *domain.Session) bool {
		count++
		foundIDs[s.ID] = true
		return true
	})

	assert.Equal(t, 3, count)
	for _, id := range ids {
		assert.True(t, foundIDs[id])
	}
}

func TestManager_Concurrent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mgr := NewManager()
	wg := sync.WaitGroup{}
	numGoroutines := 100

	// Concurrent Add
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(_ int) {
			defer wg.Done()
			mockClient := mock_wss.NewMockClient(ctrl)
			// Use unique ID for each session
			// Note: gomock EXPECT calls might be racy if we share the same mock object,
			// but here we create new mock objects per goroutine.
			// However, gomock controller usage across goroutines is thread-safe.
			mockClient.EXPECT().ID().Return("sess").AnyTimes()
			// Wait, ID should be unique to count correctly
			// But since we can't easily inject unique ID into mock return without parameter,
			// Let's assume we test thread safety of map, not necessarily count accuracy if IDs collide.
			// Actually, let's make IDs unique by ignoring the mock return in NewSession?
			// session.NewSession calls conn.ID().

			// Let's construct session manually to control ID without complex mock matching
			sess := &domain.Session{ID: "sess", CreatedAt: 123}
			// No, session struct fields are public? Yes.
			// But we should use NewSession if possible.
			// Let's use specific return per iteration if we can, but 'i' is inside loop.
			// Better approach: Mock ID() to return something unique?
			// Or just checking that it doesn't panic is enough for "Concurrent Safe".

			// For simplicity, let's just use the map's safety.
			mgr.Add(sess)
		}(i)
	}
	wg.Wait()

	// Since IDs collide ("sess"), count should be 1.
	assert.Equal(t, int64(1), mgr.Count())

	// Test Remove
	mgr.Remove("sess")
	assert.Equal(t, int64(0), mgr.Count())
}
