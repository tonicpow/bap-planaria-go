package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/rohenaz/go-bap"
	"github.com/tonicpow/bap-planaria-go/persist"
	"github.com/tonicpow/bap-planaria-go/router"
	"github.com/tonicpow/bap-planaria-go/state"
)

// Constants
var currentBlock = 590000
var fromBlock = currentBlock
var stateBlock = 0

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
	crawler.crawl(q, currentBlock)

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
