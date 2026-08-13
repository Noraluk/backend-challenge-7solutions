package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	secret []byte
	now    func() time.Time
}

func NewJWTService(secret string, now func() time.Time) *JWTService {
	if now == nil {
		now = time.Now
	}

	return &JWTService{secret: []byte(secret), now: now}
}

func (s *JWTService) Generate(claims ports.TokenClaims) (string, error) {
	if claims.UserID == "" {
		return "", errors.New("generate token: user ID is required")
	}
	if claims.ExpiresAt.IsZero() {
		return "", errors.New("generate token: expiration is required")
	}

	registeredClaims := jwt.RegisteredClaims{
		Subject:   claims.UserID,
		IssuedAt:  jwt.NewNumericDate(s.now()),
		ExpiresAt: jwt.NewNumericDate(claims.ExpiresAt),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, registeredClaims)
	signedToken, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signedToken, nil
}

func (s *JWTService) Validate(tokenString string) (ports.TokenClaims, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, domain.ErrInvalidToken
			}
			return s.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(s.now),
	)
	if err != nil || !token.Valid || claims.Subject == "" || claims.IssuedAt == nil || claims.ExpiresAt == nil {
		return ports.TokenClaims{}, domain.ErrInvalidToken
	}

	return ports.TokenClaims{
		UserID:    claims.Subject,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}
