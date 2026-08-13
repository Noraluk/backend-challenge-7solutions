package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
	"github.com/golang-jwt/jwt/v5"
)

func TestJWTServiceGenerateAndValidate(t *testing.T) {
	now := time.Date(2026, time.August, 11, 12, 0, 0, 0, time.UTC)
	service := NewJWTService("a-test-secret-with-at-least-32-characters", func() time.Time { return now })
	want := ports.TokenClaims{UserID: "user-id", ExpiresAt: now.Add(time.Hour)}

	token, err := service.Generate(want)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	got, err := service.Validate(token)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got.UserID != want.UserID || !got.ExpiresAt.Equal(want.ExpiresAt) {
		t.Errorf("Validate() claims = %#v, want %#v", got, want)
	}
}

func TestJWTServiceRejectsInvalidTokens(t *testing.T) {
	now := time.Date(2026, time.August, 11, 12, 0, 0, 0, time.UTC)
	secret := "a-test-secret-with-at-least-32-characters"
	service := NewJWTService(secret, func() time.Time { return now })

	expired, err := service.Generate(ports.TokenClaims{UserID: "user-id", ExpiresAt: now.Add(-time.Minute)})
	if err != nil {
		t.Fatalf("Generate() expired token error = %v", err)
	}
	otherSecret, err := NewJWTService("another-test-secret-with-32-characters", func() time.Time { return now }).Generate(
		ports.TokenClaims{UserID: "user-id", ExpiresAt: now.Add(time.Hour)},
	)
	if err != nil {
		t.Fatalf("Generate() other-secret token error = %v", err)
	}
	hs384, err := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.RegisteredClaims{
		Subject:   "user-id",
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign HS384 token: %v", err)
	}
	missingSubject, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token without subject: %v", err)
	}

	tests := []struct {
		name  string
		token string
	}{
		{name: "expired", token: expired},
		{name: "bad signature", token: otherSecret},
		{name: "wrong algorithm", token: hs384},
		{name: "missing subject", token: missingSubject},
		{name: "malformed", token: "not-a-token"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := service.Validate(test.token); !errors.Is(err, domain.ErrInvalidToken) {
				t.Errorf("Validate() error = %v, want ErrInvalidToken", err)
			}
		})
	}
}

func TestJWTServiceDefaultsClockAndRejectsMissingGenerateClaims(t *testing.T) {
	service := NewJWTService("a-test-secret-with-at-least-32-characters", nil)
	if service.now == nil {
		t.Fatal("NewJWTService() did not set a default clock")
	}

	tests := []ports.TokenClaims{
		{ExpiresAt: time.Now().Add(time.Hour)},
		{UserID: "user-id"},
	}
	for _, claims := range tests {
		if _, err := service.Generate(claims); err == nil {
			t.Errorf("Generate(%#v) error = nil", claims)
		}
	}
}

func TestJWTServiceRejectsMissingRegisteredClaims(t *testing.T) {
	now := time.Date(2026, time.August, 12, 12, 0, 0, 0, time.UTC)
	secret := "a-test-secret-with-at-least-32-characters"
	service := NewJWTService(secret, func() time.Time { return now })
	tests := []jwt.RegisteredClaims{
		{Subject: "user-id", ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))},
		{Subject: "user-id", IssuedAt: jwt.NewNumericDate(now)},
	}

	for _, claims := range tests {
		token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("sign token: %v", err)
		}
		if _, err := service.Validate(token); !errors.Is(err, domain.ErrInvalidToken) {
			t.Errorf("Validate() error = %v, want ErrInvalidToken", err)
		}
	}
}
