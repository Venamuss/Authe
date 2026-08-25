package security

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenManager struct {
	secretKey []byte
	tokenTTL  time.Duration
}

func NewTokenManager(secretKey string, tokenTTL time.Duration) *TokenManager {
	return &TokenManager{
		secretKey: []byte(secretKey),
		tokenTTL:  tokenTTL,
	}
}

func (tm *TokenManager) CreateToken(username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"username": username,
			"exp":      tm.tokenTTL,
		})

	tokenString, err := token.SignedString(tm.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return tokenString, nil
}

func (tm *TokenManager) VerifyToken(tokenString string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return tm.secretKey, nil
	})

	if err != nil {
		return fmt.Errorf("failed to verify token: %w", err)
	}

	if !token.Valid {
		return fmt.Errorf("Token is invalid")
	}

	return nil
}

func (tm *TokenManager) ExtractClaimsWithMap(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return tm.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}
