package entity

import "time"

type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type AccessTokenClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
}

type RefreshTokenClaims struct {
	UserID uint `json:"user_id"`
}
type StandardClaims struct {
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
	Issuer    string `json:"iss"`
}
