package entity

import "time"

type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type AccessTokenClaims struct {
	UserID   int64  `json:"user_id,string"`
	Username string `json:"username"`
}

type RefreshTokenClaims struct {
	UserID int64 `json:"user_id,string"`
}
type StandardClaims struct {
	IssuedAt  int64  `json:"iat,string"`
	ExpiresAt int64  `json:"exp,string"`
	Issuer    string `json:"iss"`
}
