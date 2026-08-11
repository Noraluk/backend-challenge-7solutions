package mongodb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type userCollection interface {
	InsertOne(context.Context, any, ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error)
	FindOne(context.Context, any, ...options.Lister[options.FindOneOptions]) *mongo.SingleResult
	Find(context.Context, any, ...options.Lister[options.FindOptions]) (*mongo.Cursor, error)
	FindOneAndUpdate(context.Context, any, any, ...options.Lister[options.FindOneAndUpdateOptions]) *mongo.SingleResult
	DeleteOne(context.Context, any, ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, error)
	CountDocuments(context.Context, any, ...options.Lister[options.CountOptions]) (int64, error)
}

type UserRepository struct {
	collection userCollection
}

type userDocument struct {
	ID           bson.ObjectID `bson:"_id"`
	Name         string        `bson:"name"`
	Email        string        `bson:"email"`
	PasswordHash string        `bson:"password_hash"`
	CreatedAt    time.Time     `bson:"created_at"`
}

var _ ports.UserRepository = (*UserRepository)(nil)

func NewUserRepository(collection *mongo.Collection) *UserRepository {
	return &UserRepository{collection: collection}
}

func (r *UserRepository) Create(ctx context.Context, user domain.User) (domain.User, error) {
	document, err := newUserDocument(user)
	if err != nil {
		return domain.User{}, err
	}

	if _, err := r.collection.InsertOne(ctx, document); err != nil {
		return domain.User{}, mapMongoError(err)
	}

	return document.domainUser(), nil
}

func (r *UserRepository) GetByID(ctx context.Context, id domain.UserID) (domain.User, error) {
	objectID, err := parseUserID(id)
	if err != nil {
		return domain.User{}, err
	}

	return decodeUser(r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: objectID}}))
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	return decodeUser(r.collection.FindOne(ctx, bson.D{{Key: "email", Value: normalizeEmail(email)}}))
}

func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	cursor, err := r.collection.Find(
		ctx,
		bson.D{},
		options.Find().SetSort(bson.D{
			{Key: "created_at", Value: 1},
			{Key: "_id", Value: 1},
		}),
	)
	if err != nil {
		return nil, err
	}

	var documents []userDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, err
	}

	users := make([]domain.User, len(documents))
	for index, document := range documents {
		users[index] = document.domainUser()
	}

	return users, nil
}

func (r *UserRepository) Update(ctx context.Context, id domain.UserID, update ports.UserUpdate) (domain.User, error) {
	objectID, err := parseUserID(id)
	if err != nil {
		return domain.User{}, err
	}

	fields := bson.D{}
	if update.Name != nil {
		fields = append(fields, bson.E{Key: "name", Value: *update.Name})
	}
	if update.Email != nil {
		fields = append(fields, bson.E{Key: "email", Value: normalizeEmail(*update.Email)})
	}
	if len(fields) == 0 {
		return domain.User{}, ports.ErrInvalidUpdate
	}

	result := r.collection.FindOneAndUpdate(
		ctx,
		bson.D{{Key: "_id", Value: objectID}},
		bson.D{{Key: "$set", Value: fields}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	return decodeUser(result)
}

func (r *UserRepository) Delete(ctx context.Context, id domain.UserID) error {
	objectID, err := parseUserID(id)
	if err != nil {
		return err
	}

	result, err := r.collection.DeleteOne(ctx, bson.D{{Key: "_id", Value: objectID}})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ports.ErrUserNotFound
	}

	return nil
}

func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.D{})
}

func newUserDocument(user domain.User) (userDocument, error) {
	objectID := bson.NewObjectID()
	if user.ID != "" {
		var err error
		objectID, err = parseUserID(user.ID)
		if err != nil {
			return userDocument{}, err
		}
	}

	return userDocument{
		ID:           objectID,
		Name:         user.Name,
		Email:        normalizeEmail(user.Email),
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt,
	}, nil
}

func (document userDocument) domainUser() domain.User {
	return domain.User{
		ID:           domain.UserID(document.ID.Hex()),
		Name:         document.Name,
		Email:        document.Email,
		PasswordHash: document.PasswordHash,
		CreatedAt:    document.CreatedAt,
	}
}

func decodeUser(result *mongo.SingleResult) (domain.User, error) {
	var document userDocument
	if err := result.Decode(&document); err != nil {
		return domain.User{}, mapMongoError(err)
	}

	return document.domainUser(), nil
}

func parseUserID(id domain.UserID) (bson.ObjectID, error) {
	objectID, err := bson.ObjectIDFromHex(string(id))
	if err != nil {
		return bson.NilObjectID, fmt.Errorf("%w: %q", ports.ErrInvalidUserID, id)
	}

	return objectID, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func mapMongoError(err error) error {
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		return ports.ErrUserNotFound
	case mongo.IsDuplicateKeyError(err):
		return ports.ErrEmailAlreadyExists
	default:
		return err
	}
}
