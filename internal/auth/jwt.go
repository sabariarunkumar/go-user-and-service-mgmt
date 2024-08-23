package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// JWTClaimEmail email
	JWTClaimEmail = "email"
	// JWTClaimRole role
	JWTClaimRole = "role"
	// JWTClaimExpiresAt expiresAt
	JWTClaimExpiresAt = "exp"
)

// As of now, we don't implement token refresh, if product really requires
// and the security risks like Refresh Token Theft, Token Replay Attack
// Extended Exposure needs to be evaluated

// CreateJWT creates jwt with secret and  necessary claims
func CreateJWT(secret []byte, expirationInSec int64, email string, userRole string) (*string, error) {
	expiration := time.Second * time.Duration(expirationInSec)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		JWTClaimEmail:     email,
		JWTClaimRole:      userRole,
		JWTClaimExpiresAt: time.Now().Add(expiration).Unix(),
	})

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return nil, err
	}
	return &tokenString, err
}

// ValidateJWT validates received token against secret
func ValidateJWT(secret []byte, tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid or expired token: %+v", err)
	}

	return token, nil
}
