package auth

import (
	"errors"
	"testing"

	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
	"golang.org/x/crypto/bcrypt"
)

func TestBcryptPasswordHasher(t *testing.T) {
	hasher := NewBcryptPasswordHasher()
	hash, err := hasher.Hash("correct-password")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if hash == "correct-password" {
		t.Fatal("Hash() returned the plaintext password")
	}
	if cost, err := bcrypt.Cost([]byte(hash)); err != nil || cost != bcrypt.DefaultCost {
		t.Errorf("bcrypt cost = %d, %v, want %d", cost, err, bcrypt.DefaultCost)
	}
	if err := hasher.Compare(hash, "correct-password"); err != nil {
		t.Errorf("Compare() correct password error = %v", err)
	}
	if err := hasher.Compare(hash, "wrong-password"); !errors.Is(err, ports.ErrInvalidCredentials) {
		t.Errorf("Compare() wrong password error = %v, want ErrInvalidCredentials", err)
	}
}

func TestBcryptPasswordHasherPropagatesInternalErrors(t *testing.T) {
	hasher := NewBcryptPasswordHasher()

	if err := hasher.Compare("not-a-bcrypt-hash", "password"); err == nil || errors.Is(err, ports.ErrInvalidCredentials) {
		t.Errorf("Compare() error = %v, want internal hash error", err)
	}
	if _, err := hasher.Hash(string(make([]byte, 73))); err == nil {
		t.Fatal("Hash() overlong password error = nil")
	}
}
