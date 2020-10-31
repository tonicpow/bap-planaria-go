package crawler

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/bitcoinschema/go-bap"
	"github.com/bitcoinschema/go-bmap"
	"github.com/bitcoinschema/go-bob"
	"github.com/tidwall/sjson"
	"github.com/tonicpow/bap-planaria-go/database"
	"go.mongodb.org/mongo-driver/bson"
)

// SyncBlocks will crawl from a specific height and return the newest block
func SyncBlocks(height int) (newBlock int) {

	// Start tracking total crawl time
	crawlTimeStart := time.Now()

	// Bitbus Query
	q := []byte(`
		{
			"q": {
				"find": { "out.tape.cell.s": "` + bap.Prefix + `" },
				"sort": { "blk.i": 1 }
			}
		}
	`)

	// Crawl will mutate currentBlock
	newBlock = Crawl(q, height)

	// Crawl complete
	diff := time.Since(crawlTimeStart).Seconds()

	// Todo: remove logging or use go-logger
	log.Printf("bitbus sync complete in %fs\nblock height: %d\n", diff, height)

	return
}

// Crawl loops over the new bap transactions since the given block height
func Crawl(query []byte, height int) (newHeight int) {

	// Create a new client
	client := http.Client{}
	// todo: set defaults

	// Create a timestamped query by applying the "$gt" (greater then) operator with the height
	nJSON, _ := sjson.Set(string(query), `q.find.blk\.i.$gt`, height)
	// todo: test for error?

	bJSON := []byte(nJSON)

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"https://bob.bitbus.network/block",
		bytes.NewBuffer(bJSON),
	)
	if err != nil {
		log.Println(err)
		return
	}

	// Set the headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("token", os.Getenv("PLANARIA_TOKEN"))

	// Fire the request
	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		log.Println(err)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Logging
	log.Printf("initializing from block %d\n", height)

	// Create a DB connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer func() {
		cancel()
	}()
	var conn *database.Connection
	if conn, err = database.Connect(ctx); err != nil {
		log.Println(err)
		return
	}
	defer func() {
		_ = conn.Disconnect(ctx)
	}()

	// Read the body
	reader := bufio.NewReader(resp.Body)

	// Split NDJSON stream by line
	for {
		var line []byte
		line, err = reader.ReadBytes('\n')
		if errors.Is(err, io.EOF) {
			break
		}

		var bobData *bob.Tx
		if bobData, err = bob.NewFromBytes(line); err != nil {
			fmt.Println(err)
			return
		}

		if int(bobData.Blk.I) > height {
			newHeight = int(bobData.Blk.I)
		}

		// Transform from BOB to BMAP
		var bmapData *bmap.Tx
		if bmapData, err = bmap.NewFromBob(bobData); err != nil {
			log.Println(err)
			return
		}

		bsonData := bson.M{
			"_id": bobData.Tx.H,
			"tx":  bobData.Tx,
			"in":  bobData.In,
			"out": bobData.Out,
			"blk": bobData.Blk,
		}

		if bmapData.AIP != nil {
			bsonData["AIP"] = bmapData.AIP
		}

		if bmapData.BAP != nil {
			bsonData["BAP"] = bmapData.BAP
		}

		if bmapData.MAP != nil {
			bsonData["MAP"] = bmapData.MAP
		}

		collectionName := bmapData.BAP.Type
		filter := bson.M{"tx.h": bson.M{"$eq": bmapData.Tx.H}}

		// Write to DB
		if _, err = conn.UpsertOne(string(collectionName), filter, bsonData); err != nil {
			log.Println(err)
			return
		}
	}

	return
}
