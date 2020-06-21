package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/vulpemventures/liquid-taxi/pkg/fees"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
)

//Storage holds the Mongo client
type Storage struct {
	Client       *mongo.Client
	DatabaseName string
}

//NewStorage connect to the given mongo endpoint
func NewStorage(uri string) (*Storage, error) {
	connString, err := connstring.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("storage: parse uri: %w", err)
	}

	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("storage: %w", err)
	}
	return &Storage{Client: client, DatabaseName: connString.Database}, nil
}

// InsertOrder let add an order
func (s *Storage) InsertOrder(method int, unsigned, partial string, breakdown fees.Breakdown) (interface{}, error) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err := s.Client.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	defer s.Client.Disconnect(ctx)

	orders := s.Client.Database(s.DatabaseName).Collection("orders")

	document := bson.M{
		"unsigned":  unsigned,
		"partial":   partial,
		"method":    method,
		"status":    0,
		"completed": false,
		"breakdown": breakdown,
		"createdAt": time.Now().Unix(),
	}

	result, err := orders.InsertOne(ctx, document)
	if err != nil {
		return nil, fmt.Errorf("insert: %w", err)
	}

	return result.InsertedID, nil
}
