package mongodb

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type fakeUserCollection struct {
	insertOne        func(context.Context, any, ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error)
	findOne          func(context.Context, any, ...options.Lister[options.FindOneOptions]) *mongo.SingleResult
	find             func(context.Context, any, ...options.Lister[options.FindOptions]) (*mongo.Cursor, error)
	findOneAndUpdate func(context.Context, any, any, ...options.Lister[options.FindOneAndUpdateOptions]) *mongo.SingleResult
	deleteOne        func(context.Context, any, ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, error)
	countDocuments   func(context.Context, any, ...options.Lister[options.CountOptions]) (int64, error)
}

func (f *fakeUserCollection) InsertOne(ctx context.Context, document any, opts ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error) {
	return f.insertOne(ctx, document, opts...)
}

func (f *fakeUserCollection) FindOne(ctx context.Context, filter any, opts ...options.Lister[options.FindOneOptions]) *mongo.SingleResult {
	return f.findOne(ctx, filter, opts...)
}

func (f *fakeUserCollection) Find(ctx context.Context, filter any, opts ...options.Lister[options.FindOptions]) (*mongo.Cursor, error) {
	return f.find(ctx, filter, opts...)
}

func (f *fakeUserCollection) FindOneAndUpdate(ctx context.Context, filter, update any, opts ...options.Lister[options.FindOneAndUpdateOptions]) *mongo.SingleResult {
	return f.findOneAndUpdate(ctx, filter, update, opts...)
}

func (f *fakeUserCollection) DeleteOne(ctx context.Context, filter any, opts ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, error) {
	return f.deleteOne(ctx, filter, opts...)
}

func (f *fakeUserCollection) CountDocuments(ctx context.Context, filter any, opts ...options.Lister[options.CountOptions]) (int64, error) {
	return f.countDocuments(ctx, filter, opts...)
}

func TestUserRepositoryCreate(t *testing.T) {
	createdAt := time.Date(2026, time.August, 11, 12, 0, 0, 0, time.UTC)
	var inserted userDocument
	collection := &fakeUserCollection{
		insertOne: func(_ context.Context, document any, _ ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error) {
			inserted = document.(userDocument)
			return &mongo.InsertOneResult{InsertedID: inserted.ID}, nil
		},
	}
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
	id := bson.NewObjectID()
	var gotFilter any
	collection := &fakeUserCollection{
		findOne: func(_ context.Context, filter any, _ ...options.Lister[options.FindOneOptions]) *mongo.SingleResult {
			gotFilter = filter
			return mongo.NewSingleResultFromDocument(userDocument{ID: id, Email: "ada@example.com"}, nil, nil)
		},
	}
	repository := &UserRepository{collection: collection}

	if _, err := repository.GetByEmail(context.Background(), " ADA@Example.COM "); err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}

	wantFilter := bson.D{{Key: "email", Value: "ada@example.com"}}
	if !reflect.DeepEqual(gotFilter, wantFilter) {
		t.Errorf("filter = %#v, want %#v", gotFilter, wantFilter)
	}
}

func TestUserRepositoryListUsesDeterministicSort(t *testing.T) {
	documents := []any{
		userDocument{ID: bson.NewObjectID(), Name: "Ada", CreatedAt: time.Unix(1, 0)},
		userDocument{ID: bson.NewObjectID(), Name: "Grace", CreatedAt: time.Unix(2, 0)},
	}
	var gotSort any
	collection := &fakeUserCollection{
		find: func(_ context.Context, _ any, optionList ...options.Lister[options.FindOptions]) (*mongo.Cursor, error) {
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
	}
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
	id := bson.NewObjectID()
	name := "Grace Hopper"
	email := " GRACE@Example.COM "
	var gotUpdate any
	collection := &fakeUserCollection{
		findOneAndUpdate: func(_ context.Context, _ any, update any, _ ...options.Lister[options.FindOneAndUpdateOptions]) *mongo.SingleResult {
			gotUpdate = update
			return mongo.NewSingleResultFromDocument(userDocument{
				ID: id, Name: name, Email: "grace@example.com",
			}, nil, nil)
		},
	}
	repository := &UserRepository{collection: collection}

	user, err := repository.Update(context.Background(), domain.UserID(id.Hex()), ports.UserUpdate{
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
	duplicateError := mongo.WriteException{WriteErrors: mongo.WriteErrors{{Code: 11000}}}
	collection := &fakeUserCollection{
		insertOne: func(_ context.Context, _ any, _ ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error) {
			return nil, duplicateError
		},
		findOneAndUpdate: func(_ context.Context, _, _ any, _ ...options.Lister[options.FindOneAndUpdateOptions]) *mongo.SingleResult {
			return mongo.NewSingleResultFromDocument(bson.D{}, duplicateError, nil)
		},
	}
	repository := &UserRepository{collection: collection}

	if _, err := repository.Create(context.Background(), domain.User{Email: "ada@example.com"}); !errors.Is(err, ports.ErrEmailAlreadyExists) {
		t.Errorf("Create() error = %v, want ErrEmailAlreadyExists", err)
	}

	id := domain.UserID(bson.NewObjectID().Hex())
	email := "ada@example.com"
	if _, err := repository.Update(context.Background(), id, ports.UserUpdate{Email: &email}); !errors.Is(err, ports.ErrEmailAlreadyExists) {
		t.Errorf("Update() error = %v, want ErrEmailAlreadyExists", err)
	}
}

func TestUserRepositoryRejectsInvalidIDAndEmptyUpdate(t *testing.T) {
	repository := &UserRepository{collection: &fakeUserCollection{}}

	if _, err := repository.GetByID(context.Background(), "invalid"); !errors.Is(err, ports.ErrInvalidUserID) {
		t.Errorf("GetByID() error = %v, want ErrInvalidUserID", err)
	}

	id := domain.UserID(bson.NewObjectID().Hex())
	if _, err := repository.Update(context.Background(), id, ports.UserUpdate{}); !errors.Is(err, ports.ErrInvalidUpdate) {
		t.Errorf("Update() error = %v, want ErrInvalidUpdate", err)
	}
}

func TestUserRepositoryDeleteAndCount(t *testing.T) {
	id := domain.UserID(bson.NewObjectID().Hex())
	collection := &fakeUserCollection{
		deleteOne: func(_ context.Context, _ any, _ ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, error) {
			return &mongo.DeleteResult{DeletedCount: 0}, nil
		},
		countDocuments: func(_ context.Context, _ any, _ ...options.Lister[options.CountOptions]) (int64, error) {
			return 42, nil
		},
	}
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
