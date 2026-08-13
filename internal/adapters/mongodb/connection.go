package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
)

const (
	usersCollectionName = "users"
	emailIndexName      = "users_email_unique"
	operationTimeout    = 10 * time.Second
)

type Connection struct {
	client     *mongo.Client
	users      *mongo.Collection
	disconnect func(context.Context) error
}

var createMongoClient = func(uri string) (*mongo.Client, error) {
	return mongo.Connect(options.Client().ApplyURI(uri).SetServerSelectionTimeout(operationTimeout))
}

var pingMongoClient = func(ctx context.Context, client *mongo.Client) error {
	return client.Ping(ctx, nil)
}

var createUserEmailIndex = func(ctx context.Context, users *mongo.Collection) error {
	_, err := users.Indexes().CreateOne(ctx, userEmailIndexModel())
	return err
}

func Connect(ctx context.Context, uri, databaseName string) (*Connection, error) {
	client, err := createMongoClient(uri)
	if err != nil {
		return nil, fmt.Errorf("create MongoDB client: %w", err)
	}

	operationContext, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	if err := pingMongoClient(operationContext, client); err != nil {
		disconnectClient(client)
		return nil, fmt.Errorf("ping MongoDB: %w", err)
	}

	users := client.Database(databaseName).Collection(usersCollectionName)
	if err := createUserEmailIndex(operationContext, users); err != nil {
		disconnectClient(client)
		return nil, fmt.Errorf("create users email index: %w", err)
	}

	return &Connection{client: client, users: users, disconnect: client.Disconnect}, nil
}

func (c *Connection) UserRepository() ports.UserRepository {
	return NewUserRepository(c.users)
}

func (c *Connection) Disconnect(ctx context.Context) error {
	disconnect := c.disconnect
	if disconnect == nil {
		disconnect = c.client.Disconnect
	}
	if err := disconnect(ctx); err != nil {
		return fmt.Errorf("disconnect MongoDB: %w", err)
	}

	return nil
}

func userEmailIndexModel() mongo.IndexModel {
	return mongo.IndexModel{
		Keys: bson.D{{Key: "email", Value: 1}},
		Options: options.Index().
			SetName(emailIndexName).
			SetUnique(true),
	}
}

func disconnectClient(client *mongo.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()
	_ = client.Disconnect(ctx)
}
