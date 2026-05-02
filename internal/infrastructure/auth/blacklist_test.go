package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenBlacklist_AddAndCheck(t *testing.T) {
	bl := &TokenBlacklist{
		blacklist: make(map[string]time.Time),
	}

	bl.AddToken("token1", 1*time.Hour)

	assert.True(t, bl.IsTokenBlacklisted("token1"))
	assert.False(t, bl.IsTokenBlacklisted("token2"))
}

func TestTokenBlacklist_ExpiredToken(t *testing.T) {
	bl := &TokenBlacklist{
		blacklist: make(map[string]time.Time),
	}

	// Add a token that already expired
	bl.mu.Lock()
	bl.blacklist["expired"] = time.Now().Add(-1 * time.Second)
	bl.mu.Unlock()

	// Should return false for expired tokens
	assert.False(t, bl.IsTokenBlacklisted("expired"))
}

func TestTokenBlacklist_ConcurrentAccess(t *testing.T) {
	bl := &TokenBlacklist{
		blacklist: make(map[string]time.Time),
	}

	done := make(chan struct{})

	// Concurrent writers
	go func() {
		for i := 0; i < 100; i++ {
			bl.AddToken("token", 1*time.Hour)
		}
		done <- struct{}{}
	}()

	// Concurrent readers
	go func() {
		for i := 0; i < 100; i++ {
			bl.IsTokenBlacklisted("token")
		}
		done <- struct{}{}
	}()

	<-done
	<-done
	// If we get here without a data race panic, the test passes.
}
