package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type UserDocument struct {
	ID           bson.ObjectID `bson:"_id"`
	Name         string        `bson:"name"`
	Email        string        `bson:"email"`
	PasswordHash string        `bson:"password_hash"`
	CreatedAt    time.Time     `bson:"created_at"`
}

func NewUserDocument(user domain.User) (UserDocument, error) {
	objectID := bson.NewObjectID()
	if user.ID != "" {
		var err error
		objectID, err = bson.ObjectIDFromHex(string(user.ID))
		if err != nil {
			return UserDocument{}, fmt.Errorf("%w: %q", ports.ErrInvalidUserID, user.ID)
		}
	}

	return UserDocument{
		ID:           objectID,
		Name:         user.Name,
		Email:        strings.ToLower(strings.TrimSpace(user.Email)),
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt,
	}, nil
}

func (document UserDocument) DomainUser() domain.User {
	return domain.User{
		ID:           document.ID.Hex(),
		Name:         document.Name,
		Email:        document.Email,
		PasswordHash: document.PasswordHash,
		CreatedAt:    document.CreatedAt,
	}
}
