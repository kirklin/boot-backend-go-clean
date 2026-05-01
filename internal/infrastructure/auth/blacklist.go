package auth

import (
	"sync"
	"time"
)

type TokenBlacklist struct {
	mu        sync.RWMutex
	blacklist map[string]time.Time // 存储令牌和过期时间
}

func NewTokenBlacklist() *TokenBlacklist {
	tb := &TokenBlacklist{
		blacklist: make(map[string]time.Time),
	}
	go tb.startCleanupRoutine()
	return tb
}

func (b *TokenBlacklist) startCleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		b.mu.Lock()
		now := time.Now()
		for token, expiration := range b.blacklist {
			if now.After(expiration) {
				delete(b.blacklist, token)
			}
		}
		b.mu.Unlock()
	}
}

// AddToken adds a token to the blacklist with an expiration time
func (b *TokenBlacklist) AddToken(token string, duration time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.blacklist[token] = time.Now().Add(duration)
}

// IsTokenBlacklisted checks if a token is in the blacklist
func (b *TokenBlacklist) IsTokenBlacklisted(token string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	expiration, exists := b.blacklist[token]
	if !exists {
		return false
	}
	// Remove expired tokens
	if time.Now().After(expiration) {
		delete(b.blacklist, token)
		return false
	}
	return true
}
