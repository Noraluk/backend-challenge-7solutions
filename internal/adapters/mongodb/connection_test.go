package mongodb

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
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
