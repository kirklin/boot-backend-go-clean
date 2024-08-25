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
	return &TokenBlacklist{
		blacklist: make(map[string]time.Time),
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
