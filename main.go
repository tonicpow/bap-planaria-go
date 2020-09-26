package main

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
	"github.com/tonicpow/bap-planaria-go/persist"
	"github.com/tonicpow/bap-planaria-go/router"
	"github.com/tonicpow/bap-planaria-go/state"
	"go.mongodb.org/mongo-driver/bson"
)

// Constants
var currentBlock = 590000
var fromBlock = currentBlock
var stateBlock = 0

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
	fmt.Printf("Initializing from block %d\n", currentBlock)

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

		if int(bobData.Blk.I) > currentBlock {
			currentBlock = int(bobData.Blk.I)
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
}

func main() {

	// load persisted block to continue from
	if err := persist.Load("./block.tmp", &currentBlock); err != nil {
		log.Println("Cant load it for some reason", err)
	} else {
		stateBlock = currentBlock
	}

	then := time.Now()
	q := []byte(`
	{
	  "q": {
	    "find": { "out.tape.cell.s": "` + bap.Prefix + `" },
	    "sort": { "blk.i": 1 }
	  }
	}`)

	// crawl will mutate currentBlock as it crawls forward from the given block height
	crawl(q, currentBlock)

	diff := time.Now().Sub(then).Seconds()
	fmt.Printf("Bitbus sync complete in %fs\nBlock height: %d\nSync height: %d\n", diff, currentBlock, stateBlock)

	then = time.Now()

	// if we've indexed some new txs to bring into the state
	if currentBlock > stateBlock {

		// set tru to trust planaria, false to verify every tx with a miner
		state.Build(stateBlock, true)
		diff = time.Now().Sub(then).Seconds()
		fmt.Printf("State sync complete in %fs\n", diff)
	} else {
		fmt.Println("everything up-to-date")
	}

	// First time through we start the server once synchronized
	if stateBlock == 0 {
		go startServer()
	}

	// update the state block clounter
	stateBlock = currentBlock
	state.SaveProgress(uint32(stateBlock))
	time.Sleep(30 * time.Second)
	main()
}

func startServer() {
	// TODO: Make configurable
	// Load the server
	log.Println("starting Go web server on http://localhost:8888")
	srv := &http.Server{
		Addr:         ":8888",
		Handler:      router.Handlers(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	log.Fatalln(srv.ListenAndServe())
}
