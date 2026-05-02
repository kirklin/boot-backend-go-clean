package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/gateway"
)

type jwtAuthenticator struct {
	accessSecret      []byte
	refreshSecret     []byte
	issuer            string
	accessExpiration  time.Duration
	refreshExpiration time.Duration
	blacklist         *TokenBlacklist
}

// NewJWTAuthenticator creates a new instance of Authenticator that uses JWT
func NewJWTAuthenticator(
	accessSecret, refreshSecret, issuer string,
	accessExpiration, refreshExpiration time.Duration,
	blacklist *TokenBlacklist,
) gateway.Authenticator {
	return &jwtAuthenticator{
		accessSecret:      []byte(accessSecret),
		refreshSecret:     []byte(refreshSecret),
		issuer:            issuer,
		accessExpiration:  accessExpiration,
		refreshExpiration: refreshExpiration,
		blacklist:         blacklist,
	}
}

// GenerateTokenPair generates an access token and a refresh token for a user
func (a *jwtAuthenticator) GenerateTokenPair(user *entity.User) (*entity.TokenPair, error) {
	accessToken, err := a.generateAccessToken(user, a.accessExpiration)
	if err != nil {
		return nil, err
	}

	refreshToken, err := a.generateRefreshToken(user, a.refreshExpiration)
	if err != nil {
		return nil, err
	}

	return &entity.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(a.accessExpiration),
	}, nil
}

func (a *jwtAuthenticator) generateAccessToken(user *entity.User, expiration time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"iat":      time.Now().Unix(),
		"iss":      a.issuer,
	}

	if expiration > 0 {
		claims["exp"] = time.Now().Add(expiration).Unix()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.accessSecret)
}

func (a *jwtAuthenticator) generateRefreshToken(user *entity.User, expiration time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"iat":     time.Now().Unix(),
		"iss":     a.issuer,
	}

	if expiration > 0 {
		claims["exp"] = time.Now().Add(expiration).Unix()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.refreshSecret)
}

// ValidateAccessToken validates an access token and returns the token claims
func (a *jwtAuthenticator) ValidateAccessToken(tokenString string) (*entity.AccessTokenClaims, *entity.StandardClaims, error) {
	claims, err := a.extractClaims(tokenString, true)
	if err != nil {
		return nil, nil, err
	}

	userID, _ := claims["user_id"].(float64)
	username, _ := claims["username"].(string)

	accessClaims := &entity.AccessTokenClaims{
		UserID:   int64(userID),
		Username: username,
	}

	standardClaims := a.extractStandardClaims(claims)

	return accessClaims, standardClaims, nil
}

// ValidateRefreshToken validates a refresh token and returns the token claims
func (a *jwtAuthenticator) ValidateRefreshToken(tokenString string) (*entity.RefreshTokenClaims, *entity.StandardClaims, error) {
	claims, err := a.extractClaims(tokenString, false)
	if err != nil {
		return nil, nil, err
	}

	userID, _ := claims["user_id"].(float64)

	refreshClaims := &entity.RefreshTokenClaims{
		UserID: int64(userID),
	}

	standardClaims := a.extractStandardClaims(claims)

	return refreshClaims, standardClaims, nil
}

// extractClaims extracts claims from a token string
func (a *jwtAuthenticator) extractClaims(tokenString string, isAccessToken bool) (jwt.MapClaims, error) {
	var secret []byte
	if isAccessToken {
		secret = a.accessSecret
	} else {
		secret = a.refreshSecret
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Validate issuer to prevent cross-service token usage
		if iss, ok := claims["iss"].(string); !ok || iss != string(a.issuer) {
			return nil, errors.New("invalid token issuer")
		}
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (a *jwtAuthenticator) extractStandardClaims(claims jwt.MapClaims) *entity.StandardClaims {
	return &entity.StandardClaims{
		IssuedAt:  int64(claims["iat"].(float64)),
		ExpiresAt: int64(claims["exp"].(float64)),
		Issuer:    claims["iss"].(string),
	}
}

func (a *jwtAuthenticator) BlacklistToken(token string, duration time.Duration) {
	a.blacklist.AddToken(token, duration)
}

func (a *jwtAuthenticator) IsTokenBlacklisted(token string) bool {
	return a.blacklist.IsTokenBlacklisted(token)
}
