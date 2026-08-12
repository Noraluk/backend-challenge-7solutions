package model

import (
	"errors"
	"testing"
	"time"

	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestNewUserDocumentAndDomainUser(t *testing.T) {
	id := bson.NewObjectID()
	createdAt := time.Date(2026, time.August, 12, 12, 0, 0, 0, time.UTC)
	document, err := NewUserDocument(domain.User{
		ID:           id.Hex(),
		Name:         "Ada",
		Email:        " ADA@Example.COM ",
		PasswordHash: "hash",
		CreatedAt:    createdAt,
	})
	if err != nil {
		t.Fatalf("NewUserDocument() error = %v", err)
	}
	if document.ID != id || document.Email != "ada@example.com" {
		t.Errorf("document = %#v", document)
	}

	user := document.DomainUser()
	if user.ID != id.Hex() || user.Name != "Ada" || user.PasswordHash != "hash" || !user.CreatedAt.Equal(createdAt) {
		t.Errorf("DomainUser() = %#v", user)
	}
}

func TestNewUserDocumentRejectsInvalidID(t *testing.T) {
	if _, err := NewUserDocument(domain.User{ID: "invalid"}); !errors.Is(err, ports.ErrInvalidUserID) {
		t.Errorf("NewUserDocument() error = %v, want ErrInvalidUserID", err)
	}
}
