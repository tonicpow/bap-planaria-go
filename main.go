package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/rohenaz/go-bap"
	"github.com/rohenaz/go-bmap"
	"github.com/rohenaz/go-bob"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/tonicpow/bap-planaria-go/database"
	"github.com/tonicpow/bap-planaria-go/state"
	"go.mongodb.org/mongo-driver/bson"
)

// Constants
var currentBlock = 625000

func crawl(query []byte, height int) {
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
	fmt.Printf("current block? %d", currentBlock)

	defer conn.Disconnect(ctx)
	reader := bufio.NewReader(resp.Body)
	// Split NDJSON stream by line
	for {
		line, err := reader.ReadBytes('\n')
		if err == io.EOF {
			break
		}

		bobGjsonResult := gjson.Get(string(line), "*")
		// Update the current_block height when a tx with new block is discovered
		if int(bobGjsonResult.Get("blk.i").Int()) > currentBlock {
			currentBlock = int(bobGjsonResult.Get("blk.i").Int())
			fmt.Printf("Crawling block: %d\n", currentBlock)
		}

		bobData := bob.New()
		if err := json.Unmarshal(line, &bobData); err != nil {
			fmt.Println("Error:", err)
			return
		}

		// Transform from BOB to BMAP
		bmapData := bmap.New()
		bmapData.FromBob(bobData)

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
		// Write to DB
		_, err = conn.InsertOne(collectionName, bsonData)
		// log.Println("Inserted")
	}

	// Print tx line to stdout
	if err != nil {
		fmt.Println(err)
	}
}

func main() {

	q := []byte(`
  {
    "q": {
      "find": { "out.tape.cell.s": "` + bap.Prefix + `" },
      "sort": { "blk.i": 1 }
    }
  }`)
	crawl(q, currentBlock)

	state.Build()
	// time.Sleep(10 * time.Second)
	// main()
}
