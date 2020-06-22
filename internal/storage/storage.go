package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/vulpemventures/liquid-taxi/pkg/fees"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"

	pb "github.com/vulpemventures/taxi-protobuf/rpc/taxi"
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

	err = client.Connect(context.Background())
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	return &Storage{Client: client, DatabaseName: connString.Database}, nil
}

// Close closes the connection
func (s *Storage) Close() error {
	err := s.Client.Disconnect(context.Background())
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	return nil
}

//GetOrderByID let get an order by id
func (s *Storage) GetOrderByID(id string) (order *Order, notFound bool, err error) {
	orderID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, false, fmt.Errorf("objectID: %w", err)
	}

	orders := s.Client.Database(s.DatabaseName).Collection("orders")

	filter := bson.M{"_id": orderID}
	err = orders.FindOne(context.Background(), filter).Decode(&order)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, true, fmt.Errorf("FindOne: %w", err)
		}
		return nil, false, fmt.Errorf("FindOne: %w", err)
	}

	return order, false, nil
}

// InsertOrder let add an order
func (s *Storage) InsertOrder(method int, unsigned, partial string, breakdown *fees.Breakdown) (string, error) {

	orders := s.Client.Database(s.DatabaseName).Collection("orders")

	document := &Order{
		Status:      0,
		IsCompleted: false,
		Unsigned:    unsigned,
		Partial:     partial,
		Method:      method,
		Breakdown:   breakdown,
		CreatedAt:   time.Now().Unix(),
	}

	result, err := orders.InsertOne(context.Background(), document)
	if err != nil {
		return "", fmt.Errorf("insert: %w", err)
	}

	objID := result.InsertedID.(primitive.ObjectID)
	return objID.Hex(), nil
}

//CheckLock compare the given unspent if present in the locks collection
func (s *Storage) CheckLock(txid string, vout int) (isLocked bool, err error) {
	locks := s.Client.Database(s.DatabaseName).Collection("locks")

	var uns Unspent
	filter := bson.M{"txid": txid, "vout": vout}
	err = locks.FindOne(context.Background(), filter).Decode(&uns)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return true, fmt.Errorf("GetLock: %w", err)
	}

	return true, nil
}

//LockUnspent lock an UTXO so will not be picked up by others
func (s *Storage) LockUnspent(txid string, vout int) error {
	locks := s.Client.Database(s.DatabaseName).Collection("locks")

	document := &Unspent{txid, vout}
	_, err := locks.InsertOne(context.Background(), document)
	if err != nil {
		return fmt.Errorf("insert: %w", err)
	}

	return nil
}

//UnlockUnspent lock an UTXO so will not be picked up by others
func (s *Storage) UnlockUnspent(txid string, vout int) error {
	locks := s.Client.Database(s.DatabaseName).Collection("locks")

	document := &Unspent{txid, vout}
	_, err := locks.DeleteOne(context.Background(), document)
	if err != nil {
		return fmt.Errorf("insert: %w", err)
	}

	return nil
}

//Unspent defines the locked unspent defined by the outpoint
type Unspent struct {
	TxID string
	Vout int
}

//Order defines the stored order
type Order struct {
	Status      int
	IsCompleted bool
	Unsigned    string
	Breakdown   *fees.Breakdown
	Partial     string
	CreatedAt   int64
	Method      int
}

//ToProtobuf converts a Storage.Order to pb.Order
func (order *Order) ToProtobuf(orderID string) *pb.Order {
	breakdown := order.Breakdown
	return &pb.Order{
		OrderId:     orderID,
		Status:      pb.Order_Status(order.Status),
		IsCompleted: order.IsCompleted,
		Method:      pb.Order_PaymentMethod(order.Method),
		Unsigned:    &pb.Unsigned{Pset: order.Unsigned},
		Partial:     &pb.PartialWithFees{Pset: order.Partial},
		CreatedAt:   order.CreatedAt,
		Breakdown: &pb.Breakdown{
			Size:       breakdown.Size,
			Fee:        breakdown.Fee,
			Spread:     breakdown.Spread,
			SatPerByte: breakdown.SatPerByte,
			Rate:       breakdown.Rate,
			Total:      breakdown.Total,
		},
	}
}
