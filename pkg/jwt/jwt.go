package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
)

var (
	AccessSecret  []byte
	RefreshSecret []byte
	Issuer        string
)

// InitJWT initializes the JWT package with the necessary secrets and issuer
func InitJWT(accessSecret, refreshSecret, issuer string) {
	AccessSecret = []byte(accessSecret)
	RefreshSecret = []byte(refreshSecret)
	Issuer = issuer
}

// GenerateTokenPair generates an access token and a refresh token for a user
func GenerateTokenPair(user *entity.User, accessExpiration, refreshExpiration time.Duration) (*entity.TokenPair, error) {
	accessToken, err := GenerateAccessToken(user, accessExpiration)
	if err != nil {
		return nil, err
	}

	refreshToken, err := GenerateRefreshToken(user, refreshExpiration)
	if err != nil {
		return nil, err
	}

	return &entity.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(accessExpiration),
	}, nil
}

// GenerateAccessToken generates an access token for a user
func GenerateAccessToken(user *entity.User, expiration time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"iat":      time.Now().Unix(),
		"iss":      Issuer,
	}

	if expiration > 0 {
		claims["exp"] = time.Now().Add(expiration).Unix()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(AccessSecret)
}

// GenerateRefreshToken generates a refresh token for a user
func GenerateRefreshToken(user *entity.User, expiration time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"iat":     time.Now().Unix(),
		"iss":     Issuer,
	}

	if expiration > 0 {
		claims["exp"] = time.Now().Add(expiration).Unix()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(RefreshSecret)
}

// ValidateAccessToken validates an access token and returns the token claims
func ValidateAccessToken(tokenString string) (*entity.AccessTokenClaims, *entity.StandardClaims, error) {
	claims, err := ExtractClaims(tokenString, true)
	if err != nil {
		return nil, nil, err
	}

	userID, _ := claims["user_id"].(float64)
	username, _ := claims["username"].(string)

	accessClaims := &entity.AccessTokenClaims{
		UserID:   uint(userID),
		Username: username,
	}

	standardClaims := extractStandardClaims(claims)

	return accessClaims, standardClaims, nil
}

// ValidateRefreshToken validates a refresh token and returns the token claims
func ValidateRefreshToken(tokenString string) (*entity.RefreshTokenClaims, *entity.StandardClaims, error) {
	claims, err := ExtractClaims(tokenString, false)
	if err != nil {
		return nil, nil, err
	}

	userID, _ := claims["user_id"].(float64)

	refreshClaims := &entity.RefreshTokenClaims{
		UserID: uint(userID),
	}

	standardClaims := extractStandardClaims(claims)

	return refreshClaims, standardClaims, nil
}

// IsAuthorized checks if a token is valid and not expired
func IsAuthorized(tokenString string, isAccessToken bool) bool {
	var secret []byte
	if isAccessToken {
		secret = AccessSecret
	} else {
		secret = RefreshSecret
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})

	return err == nil && token.Valid
}

// ExtractClaims extracts claims from a token string
func ExtractClaims(tokenString string, isAccessToken bool) (jwt.MapClaims, error) {
	var secret []byte
	if isAccessToken {
		secret = AccessSecret
	} else {
		secret = RefreshSecret
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
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func extractStandardClaims(claims jwt.MapClaims) *entity.StandardClaims {
	return &entity.StandardClaims{
		IssuedAt:  int64(claims["iat"].(float64)),
		ExpiresAt: int64(claims["exp"].(float64)),
		Issuer:    claims["iss"].(string),
	}
}
