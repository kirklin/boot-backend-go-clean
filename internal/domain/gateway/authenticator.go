package gateway

import (
	"time"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
)

// Authenticator defines the interface for generating, validating, and blacklisting authentication tokens.
// This interface belongs to the domain layer, ensuring that usecases do not depend on specific JWT or infrastructure logic.
type Authenticator interface {
	// GenerateTokenPair generates an access token and a refresh token for a user
	GenerateTokenPair(user *entity.User) (*entity.TokenPair, error)

	// ValidateAccessToken validates an access token string and returns the claims
	ValidateAccessToken(tokenString string) (*entity.AccessTokenClaims, *entity.StandardClaims, error)

	// ValidateRefreshToken validates a refresh token string and returns the claims
	ValidateRefreshToken(tokenString string) (*entity.RefreshTokenClaims, *entity.StandardClaims, error)

	// BlacklistToken adds a token to the blacklist with an expiration duration
	BlacklistToken(token string, duration time.Duration)

	// IsTokenBlacklisted checks if a token is present in the blacklist
	IsTokenBlacklisted(token string) bool
}
