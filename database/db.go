package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bitcoinschema/go-bmap"
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
		return nil, fmt.Errorf("set BAP_MONGAO_URL before running %s", bapMongoURL)
	}
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(bapMongoURL))
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &Connection{client}, nil
}

// GetIdentityStateFromAddress gets a single document for a state collection
func (c *Connection) GetIdentityStateFromAddress(address string) (*identity.State, error) {
	collection := c.Database(databaseName).Collection("identityState")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()
	filter := bson.M{"history.address": bson.M{"$eq": address}}
	opts := options.FindOne()
	document := collection.FindOne(ctx, filter, opts)

	idState := identity.State{}
	if err := document.Decode(&idState); err != nil {
		return nil, err
	}

	return &idState, nil
}

// GetIdentityStateByTxID gets a single document for a state collection by txid
func (c *Connection) GetIdentityStateByTxID(txid string) (*identity.State, error) {
	collection := c.Database(databaseName).Collection("identityState")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()
	filter := bson.M{"Tx.h": bson.M{"$eq": txid}}
	opts := options.FindOne()
	document := collection.FindOne(ctx, filter, opts)

	idState := identity.State{}
	if err := document.Decode(&idState); err != nil {
		return nil, err
	}

	return &idState, nil
}

// GetIdentityState gets a single document for a state collection
func (c *Connection) GetIdentityState(idKey string) (*identity.State, error) {
	collection := c.Database(databaseName).Collection("identityState")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()
	filter := bson.M{"_id": bson.M{"$eq": idKey}}
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()
	filter := bson.M{"_id": urnHash}
	opts := options.FindOne()
	document := collection.FindOne(ctx, filter, opts)

	attestationState := attestation.State{}
	if err := document.Decode(&attestationState); err != nil {
		return nil, err
	}

	return &attestationState, nil
}

// ClearState will clear the state (drop collection)
func (c *Connection) ClearState() error {
	collection := c.Database(databaseName).Collection("identityState")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()
	return collection.Drop(ctx)
	// TODO: Clear attestationState
}

// GetDocs gets a number of documents for a given collection
func (c *Connection) GetDocs(collectionName string, limit int64, skip int64, filter bson.M) ([]bmap.Tx, error) {
	collection := c.Database(databaseName).Collection(collectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer func() {
		cancel()
	}()
	cur, err := collection.Find(ctx, filter, &options.FindOptions{
		Skip:  &skip,
		Limit: &limit,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = cur.Close(ctx)
	}()
	var txs []bmap.Tx
	for cur.Next(ctx) {
		// To decode into a bmap.Tx
		bmapTx := bmap.Tx{}
		if err = cur.Decode(&bmapTx); err != nil {
			return nil, err
		}

		txs = append(txs, bmapTx)
	}
	if err = cur.Err(); err != nil {
		return nil, err
	}
	return txs, nil
}

// GetStateDocs gets a number of documents for a given state collection
func (c *Connection) GetStateDocs(collectionName string, limit int64, skip int64, filter bson.M) ([]bson.M, error) {
	collection := c.Database(databaseName).Collection(collectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer func() {
		cancel()
	}()
	cur, err := collection.Find(ctx, filter, &options.FindOptions{
		Skip:  &skip,
		Limit: &limit,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = cur.Close(ctx)
	}()
	var txs []bson.M
	for cur.Next(ctx) {
		// To decode into a bmap.Tx
		var record bson.M
		if err = cur.Decode(&record); err != nil {
			return nil, err
		}

		txs = append(txs, record)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return txs, nil
}

// InsertOne connects and inserts the provided data into the provided collection
func (c *Connection) InsertOne(collectionName string, data bson.M) (interface{}, error) {

	collection := c.Database(databaseName).Collection(collectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer func() {
		cancel()
	}()
	res, err := collection.InsertOne(ctx, data)
	if err != nil {
		return 0, err
	}

	return res.InsertedID, nil
}

// Update connects and updates the provided data into the provided collection
// NOTE: This function can update multiple records if the filter is not restrictive
func (c *Connection) Update(collectionName string, filter interface{}, update bson.M) (interface{}, error) {

	collection := c.Database(databaseName).Collection(collectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer func() {
		cancel()
	}()
	res, err := collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, err
	}

	return res, nil
}

// UpsertOne connects and updates the provided data into the provided collection given the filter
func (c *Connection) UpsertOne(collectionName string, filter interface{}, data bson.M) (interface{}, error) {

	collection := c.Database(databaseName).Collection(collectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer func() {
		cancel()
	}()
	opts := options.Update().SetUpsert(true)

	update := bson.M{"$set": data}

	res, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return 0, err
	}

	return res.UpsertedID, nil
}

// Upsert connects and updates the provided data into the provided collection given the filter
func (c *Connection) Upsert(collectionName string, filter interface{}, update bson.M) (interface{}, error) {

	collection := c.Database(databaseName).Collection(collectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer func() {
		cancel()
	}()
	opts := options.Update().SetUpsert(true)

	res, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return 0, err
	}

	return res.UpsertedID, nil
}

// CountCollectionDocs returns the number of records in a given collection
func (c *Connection) CountCollectionDocs(collectionName string, filter bson.M) (int64, error) {
	collection := c.Database(databaseName).Collection(collectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer func() {
		cancel()
	}()
	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}

	return count, nil
}
