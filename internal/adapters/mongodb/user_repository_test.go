package mongodb

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Noraluk/backend-challenge-7solutions/internal/adapters/mongodb/model"
	"github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
	"github.com/Noraluk/backend-challenge-7solutions/internal/mocks"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/mock/gomock"
)

func TestUserRepositoryCreate(t *testing.T) {
	controller := gomock.NewController(t)
	collection := mocks.NewMockUserCollection(controller)
	createdAt := time.Date(2026, time.August, 11, 12, 0, 0, 0, time.UTC)
	var inserted model.UserDocument
	collection.EXPECT().InsertOne(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, document any, _ ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error) {
			inserted = document.(model.UserDocument)
			return &mongo.InsertOneResult{InsertedID: inserted.ID}, nil
		},
	)
	repository := &UserRepository{collection: collection}

	user, err := repository.Create(context.Background(), domain.User{
		Name:         "Ada Lovelace",
		Email:        " ADA@Example.COM ",
		PasswordHash: "hashed-password",
		CreatedAt:    createdAt,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if user.ID == "" {
		t.Error("Create() returned an empty ID")
	}
	if inserted.Email != "ada@example.com" || user.Email != inserted.Email {
		t.Errorf("normalized email = %q, want %q", inserted.Email, "ada@example.com")
	}
	if inserted.PasswordHash != "hashed-password" || user.PasswordHash != inserted.PasswordHash {
		t.Error("password hash was not preserved")
	}
	if !user.CreatedAt.Equal(createdAt) {
		t.Errorf("CreatedAt = %s, want %s", user.CreatedAt, createdAt)
	}
}

func TestUserRepositoryGetByEmailNormalizesFilter(t *testing.T) {
	controller := gomock.NewController(t)
	collection := mocks.NewMockUserCollection(controller)
	id := bson.NewObjectID()
	var gotFilter any
	collection.EXPECT().FindOne(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, filter any, _ ...options.Lister[options.FindOneOptions]) *mongo.SingleResult {
			gotFilter = filter
			return mongo.NewSingleResultFromDocument(model.UserDocument{ID: id, Email: "ada@example.com"}, nil, nil)
		},
	)
	repository := &UserRepository{collection: collection}

	if _, err := repository.GetByEmail(context.Background(), " ADA@Example.COM "); err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}

	wantFilter := bson.D{{Key: "email", Value: "ada@example.com"}}
	if !reflect.DeepEqual(gotFilter, wantFilter) {
		t.Errorf("filter = %#v, want %#v", gotFilter, wantFilter)
	}
}

func TestNewUserRepository(t *testing.T) {
	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("mongo.Connect() error = %v", err)
	}
	collection := client.Database("test").Collection("users")
	repository := NewUserRepository(collection)
	if repository == nil || repository.collection != collection {
		t.Errorf("NewUserRepository() = %#v", repository)
	}
}

func TestUserRepositoryGetByID(t *testing.T) {
	controller := gomock.NewController(t)
	collection := mocks.NewMockUserCollection(controller)
	id := bson.NewObjectID()
	collection.EXPECT().FindOne(gomock.Any(), bson.D{{Key: "_id", Value: id}}).Return(
		mongo.NewSingleResultFromDocument(model.UserDocument{ID: id, Name: "Ada"}, nil, nil),
	)
	repository := &UserRepository{collection: collection}

	user, err := repository.GetByID(context.Background(), id.Hex())
	if err != nil || user.ID != id.Hex() {
		t.Errorf("GetByID() = %#v, %v", user, err)
	}
}

func TestUserRepositoryListUsesDeterministicSort(t *testing.T) {
	controller := gomock.NewController(t)
	collection := mocks.NewMockUserCollection(controller)
	documents := []any{
		model.UserDocument{ID: bson.NewObjectID(), Name: "Ada", CreatedAt: time.Unix(1, 0)},
		model.UserDocument{ID: bson.NewObjectID(), Name: "Grace", CreatedAt: time.Unix(2, 0)},
	}
	var gotSort any
	collection.EXPECT().Find(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ any, optionList ...options.Lister[options.FindOptions]) (*mongo.Cursor, error) {
			findOptions := options.FindOptions{}
			for _, option := range optionList {
				for _, setter := range option.List() {
					if err := setter(&findOptions); err != nil {
						t.Fatalf("apply find option: %v", err)
					}
				}
			}
			gotSort = findOptions.Sort
			return mongo.NewCursorFromDocuments(documents, nil, nil)
		},
	)
	repository := &UserRepository{collection: collection}

	users, err := repository.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(users) != 2 || users[0].Name != "Ada" || users[1].Name != "Grace" {
		t.Errorf("List() users = %#v", users)
	}

	wantSort := bson.D{{Key: "created_at", Value: 1}, {Key: "_id", Value: 1}}
	if !reflect.DeepEqual(gotSort, wantSort) {
		t.Errorf("sort = %#v, want %#v", gotSort, wantSort)
	}
}

func TestUserRepositoryUpdateUsesExplicitFields(t *testing.T) {
	controller := gomock.NewController(t)
	collection := mocks.NewMockUserCollection(controller)
	id := bson.NewObjectID()
	name := "Grace Hopper"
	email := " GRACE@Example.COM "
	var gotUpdate any
	collection.EXPECT().FindOneAndUpdate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ any, update any, _ ...options.Lister[options.FindOneAndUpdateOptions]) *mongo.SingleResult {
			gotUpdate = update
			return mongo.NewSingleResultFromDocument(model.UserDocument{
				ID: id, Name: name, Email: "grace@example.com",
			}, nil, nil)
		},
	)
	repository := &UserRepository{collection: collection}

	user, err := repository.Update(context.Background(), id.Hex(), dto.UpdateUserInput{
		Name:  &name,
		Email: &email,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if user.Email != "grace@example.com" {
		t.Errorf("Email = %q, want normalized email", user.Email)
	}

	wantUpdate := bson.D{{Key: "$set", Value: bson.D{
		{Key: "name", Value: name},
		{Key: "email", Value: "grace@example.com"},
	}}}
	if !reflect.DeepEqual(gotUpdate, wantUpdate) {
		t.Errorf("update = %#v, want %#v", gotUpdate, wantUpdate)
	}
}

func TestUserRepositoryMapsDuplicateEmailOnCreateAndUpdate(t *testing.T) {
	controller := gomock.NewController(t)
	collection := mocks.NewMockUserCollection(controller)
	duplicateError := mongo.WriteException{WriteErrors: mongo.WriteErrors{{Code: 11000}}}
	collection.EXPECT().InsertOne(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ any, _ ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error) {
			return nil, duplicateError
		},
	)
	collection.EXPECT().FindOneAndUpdate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _, _ any, _ ...options.Lister[options.FindOneAndUpdateOptions]) *mongo.SingleResult {
			return mongo.NewSingleResultFromDocument(bson.D{}, duplicateError, nil)
		},
	)
	repository := &UserRepository{collection: collection}

	if _, err := repository.Create(context.Background(), domain.User{Email: "ada@example.com"}); !errors.Is(err, ports.ErrEmailAlreadyExists) {
		t.Errorf("Create() error = %v, want ErrEmailAlreadyExists", err)
	}

	id := bson.NewObjectID().Hex()
	email := "ada@example.com"
	if _, err := repository.Update(context.Background(), id, dto.UpdateUserInput{Email: &email}); !errors.Is(err, ports.ErrEmailAlreadyExists) {
		t.Errorf("Update() error = %v, want ErrEmailAlreadyExists", err)
	}
}

func TestUserRepositoryRejectsInvalidIDAndEmptyUpdate(t *testing.T) {
	controller := gomock.NewController(t)
	repository := &UserRepository{collection: mocks.NewMockUserCollection(controller)}

	if _, err := repository.GetByID(context.Background(), "invalid"); !errors.Is(err, ports.ErrInvalidUserID) {
		t.Errorf("GetByID() error = %v, want ErrInvalidUserID", err)
	}
	if _, err := repository.Create(context.Background(), domain.User{ID: "invalid"}); !errors.Is(err, ports.ErrInvalidUserID) {
		t.Errorf("Create() error = %v, want ErrInvalidUserID", err)
	}
	if err := repository.Delete(context.Background(), "invalid"); !errors.Is(err, ports.ErrInvalidUserID) {
		t.Errorf("Delete() error = %v, want ErrInvalidUserID", err)
	}

	id := bson.NewObjectID().Hex()
	if _, err := repository.Update(context.Background(), id, dto.UpdateUserInput{}); !errors.Is(err, ports.ErrInvalidUpdate) {
		t.Errorf("Update() error = %v, want ErrInvalidUpdate", err)
	}
}

func TestUserRepositoryListReturnsFindError(t *testing.T) {
	controller := gomock.NewController(t)
	collection := mocks.NewMockUserCollection(controller)
	databaseError := errors.New("find failed")
	collection.EXPECT().Find(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, databaseError)
	repository := &UserRepository{collection: collection}

	if _, err := repository.List(context.Background()); !errors.Is(err, databaseError) {
		t.Errorf("List() error = %v", err)
	}
}

func TestUserRepositoryListReturnsDecodeError(t *testing.T) {
	controller := gomock.NewController(t)
	collection := mocks.NewMockUserCollection(controller)
	cursor, err := mongo.NewCursorFromDocuments([]any{bson.D{{Key: "_id", Value: "invalid"}}}, nil, nil)
	if err != nil {
		t.Fatalf("NewCursorFromDocuments() error = %v", err)
	}
	collection.EXPECT().Find(gomock.Any(), gomock.Any(), gomock.Any()).Return(cursor, nil)
	repository := &UserRepository{collection: collection}

	if _, err := repository.List(context.Background()); err == nil {
		t.Error("List() error = nil, want decode error")
	}
}

func TestUserRepositoryUpdateNameOnlyAndInvalidID(t *testing.T) {
	controller := gomock.NewController(t)
	collection := mocks.NewMockUserCollection(controller)
	id := bson.NewObjectID()
	name := "Grace"
	collection.EXPECT().FindOneAndUpdate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
		mongo.NewSingleResultFromDocument(model.UserDocument{ID: id, Name: name}, nil, nil),
	)
	repository := &UserRepository{collection: collection}

	if _, err := repository.Update(context.Background(), id.Hex(), dto.UpdateUserInput{Name: &name}); err != nil {
		t.Errorf("Update() error = %v", err)
	}
	if _, err := repository.Update(context.Background(), "invalid", dto.UpdateUserInput{Name: &name}); !errors.Is(err, ports.ErrInvalidUserID) {
		t.Errorf("Update() error = %v, want ErrInvalidUserID", err)
	}
}

func TestUserRepositoryDeleteResults(t *testing.T) {
	databaseError := errors.New("delete failed")
	tests := []struct {
		name   string
		result *mongo.DeleteResult
		err    error
		want   error
	}{
		{name: "success", result: &mongo.DeleteResult{DeletedCount: 1}},
		{name: "database error", err: databaseError, want: databaseError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			collection := mocks.NewMockUserCollection(controller)
			collection.EXPECT().DeleteOne(gomock.Any(), gomock.Any()).Return(test.result, test.err)
			repository := &UserRepository{collection: collection}
			err := repository.Delete(context.Background(), bson.NewObjectID().Hex())
			if !errors.Is(err, test.want) {
				t.Errorf("Delete() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestUserRepositoryCountReturnsError(t *testing.T) {
	controller := gomock.NewController(t)
	collection := mocks.NewMockUserCollection(controller)
	databaseError := errors.New("count failed")
	collection.EXPECT().CountDocuments(gomock.Any(), gomock.Any()).Return(int64(0), databaseError)
	repository := &UserRepository{collection: collection}

	if _, err := repository.Count(context.Background()); !errors.Is(err, databaseError) {
		t.Errorf("Count() error = %v", err)
	}
}

func TestUserRepositoryDeleteAndCount(t *testing.T) {
	controller := gomock.NewController(t)
	collection := mocks.NewMockUserCollection(controller)
	id := bson.NewObjectID().Hex()
	collection.EXPECT().DeleteOne(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ any, _ ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, error) {
			return &mongo.DeleteResult{DeletedCount: 0}, nil
		},
	)
	collection.EXPECT().CountDocuments(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ any, _ ...options.Lister[options.CountOptions]) (int64, error) {
			return 42, nil
		},
	)
	repository := &UserRepository{collection: collection}

	if err := repository.Delete(context.Background(), id); !errors.Is(err, ports.ErrUserNotFound) {
		t.Errorf("Delete() error = %v, want ErrUserNotFound", err)
	}
	count, err := repository.Count(context.Background())
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 42 {
		t.Errorf("Count() = %d, want 42", count)
	}
}

func TestMapMongoError(t *testing.T) {
	databaseError := errors.New("database unavailable")
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "not found", err: mongo.ErrNoDocuments, want: ports.ErrUserNotFound},
		{
			name: "duplicate email",
			err:  mongo.WriteException{WriteErrors: mongo.WriteErrors{{Code: 11000}}},
			want: ports.ErrEmailAlreadyExists,
		},
		{name: "database error", err: databaseError, want: databaseError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := mapMongoError(test.err); !errors.Is(err, test.want) {
				t.Errorf("mapMongoError() = %v, want %v", err, test.want)
			}
		})
	}
}
