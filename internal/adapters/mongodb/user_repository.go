package mongodb

import (
	"context"
	"errors"

	"github.com/Noraluk/backend-challenge-7solutions/internal/adapters/mongodb/model"
	"github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
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

func NewUserRepository(collection *mongo.Collection) *UserRepository {
	return &UserRepository{collection: collection}
}

func (r *UserRepository) Create(ctx context.Context, user domain.User) (domain.User, error) {
	document, err := model.NewUserDocument(user)
	if err != nil {
		return domain.User{}, err
	}

	if _, err := r.collection.InsertOne(ctx, document); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.User{}, domain.ErrEmailAlreadyExists
		}
		return domain.User{}, err
	}

	return document.DomainUser(), nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return domain.User{}, err
	}

	return decodeUser(r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: objectID}}))
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	return decodeUser(r.collection.FindOne(ctx, bson.D{{Key: "email", Value: email}}))
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

	var documents []model.UserDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, err
	}

	users := make([]domain.User, len(documents))
	for index, document := range documents {
		users[index] = document.DomainUser()
	}

	return users, nil
}

func (r *UserRepository) Update(ctx context.Context, id string, update dto.UpdateUserInput) (domain.User, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return domain.User{}, err
	}

	fields := bson.D{}
	if update.Name != nil {
		fields = append(fields, bson.E{Key: "name", Value: *update.Name})
	}
	if update.Email != nil {
		fields = append(fields, bson.E{Key: "email", Value: *update.Email})
	}
	if len(fields) == 0 {
		return domain.User{}, domain.ErrInvalidUpdate
	}

	result := r.collection.FindOneAndUpdate(
		ctx,
		bson.D{{Key: "_id", Value: objectID}},
		bson.D{{Key: "$set", Value: fields}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	return decodeUser(result)
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	result, err := r.collection.DeleteOne(ctx, bson.D{{Key: "_id", Value: objectID}})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.D{})
}

func decodeUser(result *mongo.SingleResult) (domain.User, error) {
	var document model.UserDocument
	if err := result.Decode(&document); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.User{}, domain.ErrUserNotFound
		}
		if mongo.IsDuplicateKeyError(err) {
			return domain.User{}, domain.ErrEmailAlreadyExists
		}
		return domain.User{}, err
	}

	return document.DomainUser(), nil
}
