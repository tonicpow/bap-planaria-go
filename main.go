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

	"github.com/mrz1836/go-logger"
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

// Matter cloud block hashes
// {
// "status": 200,
// "errors": [],
// "result": [
// 	{
// 		"height": 650000,
// 		"hash": "00000000000000000310c17bbb4f3f8e5371a41ec2cee36a39876042019b725b",
// 		"size": 430268,
// 		"version": 566484992,
// 		"merkleroot": "58b0688e0590a9b1302e243afe9a0900728afc950f94f1080d2e8e8f7a2acd2f",
// 		"time": 1598573981,
// 		"nonce": 367658839,
// 		"bits": "1804b6eb",
// 		"difficulty": "233214426358.1377",
// 		"previousblockhash": "0000000000000000029031f1c09178d252d4d34147a3992b93001dc3e16d74a2",
// 		"nextblockhash": "000000000000000004653fdb4b2358770c8a1e80d2540b696d2ce879d2173ca0",
// 		"coinbaseinfo": "0310eb09049a4d485f485a2f48756f42692ffabe6d6dd3fc78f2c7a896830ab39509e983c7c4d1a3a3cb9fed8fcdd38594959b9fda930200000090fed21806a8ada70500000000000000",
// 		"coinbasetxid": "9834daa6d34690981888f7db4c1c36686ebb9b685d37115abc38e0e75f9cd98d",
// 		"chainwork": "00000000000000000000000000000000000000000117fbcf44cf317be90c16ab"
// 	}]
// }

// Matter latest block
// {"status":200,"errors":[],"result":[{"height":654018,"hash":"0000000000000000009e3ba1ea2518c0a73212e0a38e3dc3aee2e7df3d883d7c","size":10554327,"version":549453824,"merkleroot":"640043cfb875916170a57d1f9ba60118422dd67075927471688767a485e5eb96","time":1600981747,"nonce":2013154616,"bits":"1804444b","difficulty":"257687900404.1103","previousblockhash":"000000000000000001531b9571684285d0b64cba3b08556c7451a6601a4d00e4","nextblockhash":"","coinbaseinfo":"03c2fa0904f40a6d5f626a2f4254432e636f6d2ffabe6d6d33c952d467a829a68681f889154a99ac11cd71ab369c697f6c128c2bd777133502000000020bb56c04ab3c0053fe052eef320000","coinbasetxid":"00eb4e6a607f8b2089e237dad9e71e17b18f08c779a2f96a985a9e6bf70be3cf","chainwork":"0000000000000000000000000000000000000000011bae229bd17d4c9df55688"}]}

type Result struct {
	Height        uint64 `json:"height"`
	Hash          string `json:"hash"`
	Size          int    `json:"size"`
	Version       int    `jsong:"version"`
	MerkleRoot    string `json:"merkleroot"`
	Time          uint64 `json:"time"`
	Nonce         uint64 `json:"nonce"`
	Bits          string `json:"bits"`
	Difficulty    string `json:"difficulty"`
	NextBlockHash string `json:"nextblockhash"`
	CoinbaseInfo  string `json:"coinbaseinfo"`
	CoinbaseTxID  string `json:"coinbasetxid"`
	Chainwork     string `json:"chainwork"`
}
type MatterCloudBlockResult struct {
	Status int      `json:"status"`
	Errors []string `json:"errors"`
	Result []Result `json:"result"`
}

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

		// bobGjsonResult := gjson.Get(string(line), "*")
		// // Update the current_block height when a tx with new block is discovered
		// if int(bobGjsonResult.Get("blk.i").Int()) > currentBlock {
		// 	currentBlock = int(bobGjsonResult.Get("blk.i").Int())
		// 	fmt.Printf("Crawling block: %d\n", currentBlock)
		// }

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

	// matter.getBlockHeaders()

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
	// Load the server
	logger.Data(2, logger.DEBUG, "starting Go web server...", logger.MakeParameter("port", 8888))
	srv := &http.Server{
		Addr:         ":8888",
		Handler:      router.Handlers(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	logger.Fatalln(srv.ListenAndServe())
}
