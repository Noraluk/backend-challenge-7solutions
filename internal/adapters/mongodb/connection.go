package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	usersCollectionName = "users"
	emailIndexName      = "users_email_unique"
	operationTimeout    = 10 * time.Second
)

type Connection struct {
	client *mongo.Client
	users  *mongo.Collection
}

func Connect(ctx context.Context, uri, databaseName string) (*Connection, error) {
	client, err := mongo.Connect(
		options.Client().ApplyURI(uri).SetServerSelectionTimeout(operationTimeout),
	)
	if err != nil {
		return nil, fmt.Errorf("create MongoDB client: %w", err)
	}

	operationContext, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	if err := client.Ping(operationContext, nil); err != nil {
		disconnectClient(client)
		return nil, fmt.Errorf("ping MongoDB: %w", err)
	}

	users := client.Database(databaseName).Collection(usersCollectionName)
	if _, err := users.Indexes().CreateOne(operationContext, userEmailIndexModel()); err != nil {
		disconnectClient(client)
		return nil, fmt.Errorf("create users email index: %w", err)
	}

	return &Connection{client: client, users: users}, nil
}

func (c *Connection) UserRepository() *UserRepository {
	return NewUserRepository(c.users)
}

func (c *Connection) Disconnect(ctx context.Context) error {
	if err := c.client.Disconnect(ctx); err != nil {
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
