package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rohenaz/go-bmap"
	"github.com/tonicpow/bap-planaria-go/attestation"
	"github.com/tonicpow/bap-planaria-go/identity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const databaseName = "bap"

// Connection is a mongo client
type Connection struct {
	*mongo.Client
}

// Connect establishes a connection to the mongo db
func Connect(ctx context.Context) (*Connection, error) {
	bapMongoURL := os.Getenv("BAP_MONGO_URL")
	if len(bapMongoURL) == 0 {
		return nil, fmt.Errorf("Set BAP_MONGAO_URL before running %s", bapMongoURL)
	}
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(bapMongoURL))
	if err != nil {
		fmt.Println("Failed", err)
		return nil, err
	}

	return &Connection{client}, nil
}

// GetIdentityStateFromAddress gets a single document for a state collection
func (c *Connection) GetIdentityStateFromAddress(address string) (*identity.State, error) {
	collection := c.Database(databaseName).Collection("identityState")
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	filter := bson.M{"IDHistory.address": bson.M{"$eq": address}}
	opts := options.FindOne()
	document := collection.FindOne(ctx, filter, opts)

	idState := identity.State{}
	err := document.Decode(&idState)
	if err != nil {
		return nil, err
	}

	return &idState, nil
}

// GetIdentityState gets a single document for a state collection
func (c *Connection) GetIdentityState(idKey string) (*identity.State, error) {
	collection := c.Database(databaseName).Collection("identityState")
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	filter := bson.M{"IDKey": bson.M{"$eq": idKey}}
	opts := options.FindOne()
	document := collection.FindOne(ctx, filter, opts)

	idState := identity.State{}
	err := document.Decode(&idState)
	if err != nil {
		return nil, err
	}

	return &idState, nil
}

// GetAttestationState gets a single document for a state collection
func (c *Connection) GetAttestationState(urnHash string) (*attestation.State, error) {
	collection := c.Database(databaseName).Collection("attestationState")
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	filter := bson.M{"urnHash": bson.M{"$eq": urnHash}}
	opts := options.FindOne()
	document := collection.FindOne(ctx, filter, opts)

	attestationState := attestation.State{}
	err := document.Decode(&attestationState)
	if err != nil {
		return nil, err
	}

	return &attestationState, nil
}

func (c *Connection) ClearState() error {
	collection := c.Database(databaseName).Collection("identityState")
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	return collection.Drop(ctx)
	// TODO: Clear attestationState
}

// GetDocs gets a number of documents for a given collection
// TODO: Supply query like above
func (c *Connection) GetDocs(collectionName string, limit int64, skip int64) ([]bmap.Tx, error) {
	collection := c.Database(databaseName).Collection(collectionName)
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	cur, err := collection.Find(ctx, bson.D{}, &options.FindOptions{
		Skip:  &skip,
		Limit: &limit,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cur.Close(context.Background())
	var txs []bmap.Tx
	for cur.Next(context.Background()) {
		// To decode into a bmap.Tx
		bmapTx := bmap.Tx{}
		err := cur.Decode(&bmapTx)
		if err != nil {
			return nil, err
		}

		txs = append(txs, bmapTx)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return txs, nil
}

// InsertOne connects and inserts the provided data into the provided collection
func (c *Connection) InsertOne(collectionName string, data bson.M) (interface{}, error) {

	collection := c.Database(databaseName).Collection(collectionName)
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	res, err := collection.InsertOne(ctx, data)
	if err != nil {
		return 0, err
	}

	return res.InsertedID, nil
}

// UpsertOne connects and updates the provided data into the provided collection given the filter
func (c *Connection) UpsertOne(collectionName string, filter interface{}, data bson.M) (interface{}, error) {

	collection := c.Database(databaseName).Collection(collectionName)
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	opts := options.Update().SetUpsert(true)

	update := bson.M{"$set": data}

	res, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return 0, err
	}

	return res.UpsertedID, nil
}

// CountCollectionDocs returns the number of records in a given colletion
func (c *Connection) CountCollectionDocs(collectionName string, filter bson.M) (int64, error) {
	collection := c.Database(databaseName).Collection(collectionName)
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}

	return count, nil
}
