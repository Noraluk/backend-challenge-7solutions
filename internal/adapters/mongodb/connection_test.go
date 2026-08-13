package mongodb

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestUserEmailIndexModel(t *testing.T) {
	model := userEmailIndexModel()
	wantKeys := bson.D{{Key: "email", Value: 1}}
	if !reflect.DeepEqual(model.Keys, wantKeys) {
		t.Errorf("index keys = %#v, want %#v", model.Keys, wantKeys)
	}

	indexOptions := options.IndexOptions{}
	for _, setter := range model.Options.List() {
		if err := setter(&indexOptions); err != nil {
			t.Fatalf("apply index option: %v", err)
		}
	}
	if indexOptions.Name == nil || *indexOptions.Name != emailIndexName {
		t.Errorf("index name = %v, want %q", indexOptions.Name, emailIndexName)
	}
	if indexOptions.Unique == nil || !*indexOptions.Unique {
		t.Errorf("index unique = %v, want true", indexOptions.Unique)
	}
}

func TestConnectFailsWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	connection, err := Connect(ctx, "mongodb://localhost:27017", "test")
	if err == nil {
		t.Fatal("Connect() error = nil")
	}
	if connection != nil {
		t.Fatal("Connect() returned a connection after ping failure")
	}
	if !strings.Contains(err.Error(), "ping MongoDB") {
		t.Errorf("Connect() error = %q, want ping context", err)
	}
}

func TestConnect(t *testing.T) {
	tests := []struct {
		name        string
		clientError error
		pingError   error
		indexError  error
		wantError   string
	}{
		{name: "success"},
		{name: "client error", clientError: errors.New("invalid URI"), wantError: "create MongoDB client"},
		{name: "ping error", pingError: errors.New("unavailable"), wantError: "ping MongoDB"},
		{name: "index error", indexError: errors.New("index failed"), wantError: "create users email index"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			restoreConnectionDependencies(t)
			createMongoClient = func(string) (*mongo.Client, error) {
				if test.clientError != nil {
					return nil, test.clientError
				}
				client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
				if err == nil {
					t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
				}
				return client, err
			}
			pingMongoClient = func(context.Context, *mongo.Client) error { return test.pingError }
			createUserEmailIndex = func(context.Context, *mongo.Collection) error { return test.indexError }

			connection, err := Connect(context.Background(), "mongodb://localhost:27017", "test")
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Errorf("Connect() error = %v, want %q", err, test.wantError)
				}
				if connection != nil {
					t.Errorf("Connect() = %#v, want nil", connection)
				}
				return
			}
			if err != nil || connection == nil || connection.users.Name() != usersCollectionName {
				t.Errorf("Connect() = %#v, %v", connection, err)
			}
		})
	}
}

func TestConnectionUserRepositoryAndDisconnect(t *testing.T) {
	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("mongo.Connect() error = %v", err)
	}
	collection := client.Database("test").Collection("users")
	connection := &Connection{client: client, users: collection}

	repository := connection.UserRepository()
	mongoRepository, ok := repository.(*UserRepository)
	if !ok || mongoRepository.collection != collection {
		t.Errorf("UserRepository() = %#v", repository)
	}
	if err := connection.Disconnect(context.Background()); err != nil {
		t.Errorf("Disconnect() error = %v", err)
	}
}

func TestConnectionDisconnectError(t *testing.T) {
	want := errors.New("disconnect failed")
	connection := &Connection{disconnect: func(context.Context) error { return want }}
	err := connection.Disconnect(context.Background())
	if !errors.Is(err, want) || !strings.Contains(err.Error(), "disconnect MongoDB") {
		t.Errorf("Disconnect() error = %v", err)
	}
}

func restoreConnectionDependencies(t *testing.T) {
	t.Helper()
	originalCreateMongoClient := createMongoClient
	originalPingMongoClient := pingMongoClient
	originalCreateUserEmailIndex := createUserEmailIndex
	t.Cleanup(func() {
		createMongoClient = originalCreateMongoClient
		pingMongoClient = originalPingMongoClient
		createUserEmailIndex = originalCreateUserEmailIndex
	})
}
