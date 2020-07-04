package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rohenaz/bap-planaria-go/bap"
	"github.com/rohenaz/bap-planaria-go/bmap"
	"github.com/rohenaz/bap-planaria-go/bob"
	"github.com/rohenaz/bap-planaria-go/database"
	"github.com/rohenaz/bap-planaria-go/state"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
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
	req.Header.Set("token", "eyJhbGciOiJFUzI1NksiLCJ0eXAiOiJKV1QifQ.eyJzdWIiOiIxNGozTGNMQlJoZU1aOHBRWnh3UEw3a013Y2NXYWZQSnNiIiwiaXNzdWVyIjoiZ2VuZXJpYy1iaXRhdXRoIn0.SUpqeTdRMEtEbGVlRlRHZkc1d1BwTDlzY2NaRjk5eG93ZHU5S09CaGEzQTNRMEpBd2t2RVc2eTJwd0Y3RjBua0MwYXROZ3ZjNjRmVnViMVpaKzdmRDNZPQ")
	resp, err := client.Do(req)
	defer resp.Body.Close()
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	conn, err := database.Connect(ctx)
	if err != nil {
		return
	}
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

		// Write to DB
		_, err = conn.InsertOne(bmapData.BAP.Type,
			bson.M{
				"tx":  bobData.Tx,
				"in":  bobData.In,
				"out": bobData.Out,
				"blk": bobData.Blk,
				"BAP": bmapData.BAP,
				"MAP": bmapData.MAP,
			})
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
      "find": { "out.tape.cell.s": "` + bap.BapPrefix + `" },
      "sort": { "blk.i": 1 }
    }
  }`)
	fmt.Printf("current block? %d", currentBlock)
	crawl(q, currentBlock)

	state.Build()
	// time.Sleep(10 * time.Second)
	// main()
}
