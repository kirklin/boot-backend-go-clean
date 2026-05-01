package auth

import "github.com/kirklin/boot-backend-go-clean/internal/domain/entity"

// JWTValidatorImpl is the concrete implementation for validating JWT tokens
type JWTValidatorImpl struct{}

// NewJWTValidator creates a new instance of JWTValidatorImpl
func NewJWTValidator() *JWTValidatorImpl {
	return &JWTValidatorImpl{}
}

// ValidateAccessToken delegates the validation to the package-level ValidateAccessToken function
func (v *JWTValidatorImpl) ValidateAccessToken(tokenString string) (*entity.AccessTokenClaims, *entity.StandardClaims, error) {
	return ValidateAccessToken(tokenString)
}
