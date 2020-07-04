package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rohenaz/go-bmap"
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
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(bapMongoURL))
	if err != nil {
		fmt.Println("Failed", err)
		return nil, err
	}

	return &Connection{client}, nil
}

// GetDocs gets a number of documents for a given collection
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

// CountCollectionDocs returns the number of records in a given colletion
func (c *Connection) CountCollectionDocs(collectionName string) (int64, error) {
	collection := c.Database(databaseName).Collection(collectionName)
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	count, err := collection.CountDocuments(ctx, bson.D{})
	if err != nil {
		return 0, err
	}

	return count, nil
}
