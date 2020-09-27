package crawler

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/rohenaz/go-bap"
	"github.com/rohenaz/go-bmap"
	"github.com/rohenaz/go-bob"
	"github.com/tidwall/sjson"
	"github.com/tonicpow/bap-planaria-go/database"
	"go.mongodb.org/mongo-driver/bson"
)

func SyncBlocks(height int) (newBlock int) {
	// Setup crawl timer
	crawlStart := time.Now()

	// Bitbus Query
	q := []byte(`
		{
			"q": {
				"find": { "out.tape.cell.s": "` + bap.Prefix + `" },
				"sort": { "blk.i": 1 }
			}
		}`)

	// Crawl will mutate currentBlock
	newBlock = Crawl(q, height)

	// Crawl complete
	diff := time.Now().Sub(crawlStart).Seconds()
	fmt.Printf("Bitbus sync complete in %fs\nBlock height: %d\n", diff, height)
	return
}

// Crawl loops over the new bap transactions since the given block height
func Crawl(query []byte, height int) (newHeight int) {

	client := http.Client{}
	// Create a timestamped query by applying the "$gt" (greater then) operator with the height
	njson, _ := sjson.Set(string(query), `q.find.blk\.i.$gt`, height)
	bjson := []byte(njson)
	req, err := http.NewRequest("POST", "https://bob.bitbus.network/block", bytes.NewBuffer(bjson))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("token", os.Getenv("PLANARIA_TOKEN"))
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	conn, err := database.Connect(ctx)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Printf("Initializing from block %d\n", height)

	defer conn.Disconnect(ctx)
	reader := bufio.NewReader(resp.Body)
	// Split NDJSON stream by line
	for {
		line, err := reader.ReadBytes('\n')
		if err == io.EOF {
			break
		}

		bobData := bob.New()
		err = bobData.FromBytes(line)
		if err != nil {
			fmt.Println("Error: 1", err)
			return
		}

		if int(bobData.Blk.I) > height {
			newHeight = int(bobData.Blk.I)
		}
		// Transform from BOB to BMAP
		bmapData := bmap.New()
		err = bmapData.FromBob(bobData)
		if err != nil {
			log.Println("Error", err)
		}

		bsonData := bson.M{
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
		_, err = conn.UpsertOne(string(collectionName), filter, bsonData)
	}

	// Print tx line to stdout
	if err != nil {
		fmt.Println(err)
	}

	return
}
