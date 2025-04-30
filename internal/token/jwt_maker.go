package token

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTMaker struct {
	secretKey string
}

func NewJWTMaker(secretKey string) *JWTMaker {
	return &JWTMaker{
		secretKey: secretKey,
	}
}

func (maker *JWTMaker) CreateToken(id uint, duration time.Duration) (string, *UserClaims, error) {
	claims, err := NewUserClaims(id, duration)
	if err != nil {
		slog.Error("Failed to create claims", "error", err)
		return "", nil, err
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(maker.secretKey))
	if err != nil {
		slog.Error("Failed to create token", "error", err)
		return "", nil, err
	}
	return tokenStr, claims, err
}

func (maker *JWTMaker) VerifyToken(tokenStr string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &UserClaims{}, func(token *jwt.Token) (any, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(maker.secretKey), nil
	})

	if err != nil {
		slog.Error("Failed to parse token", "error", err)
		return nil, err
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok {
		slog.Error("Invalid token claims")
		return nil, err
	}
	return claims, nil
}
