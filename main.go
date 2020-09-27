package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/tonicpow/bap-planaria-go/config"
	"github.com/tonicpow/bap-planaria-go/crawler"
	"github.com/tonicpow/bap-planaria-go/persist"
	"github.com/tonicpow/bap-planaria-go/router"
	"github.com/tonicpow/bap-planaria-go/state"
)

func syncWorker(currentBlock int, wait bool) {
	if wait {
		time.Sleep(30 * time.Second)
	}

	// crawl
	newBlock := crawler.SyncBlocks(currentBlock)

	// if we've indexed some new txs to bring into the state
	if newBlock > currentBlock {
		newBlock = state.SyncState(currentBlock)
	} else {
		fmt.Println("everything up-to-date")
	}

	go syncWorker(newBlock, true)
}

func main() {

	var currentBlock int

	// load persisted block to continue from
	if err := persist.Load("./block.tmp", &currentBlock); err != nil {
		log.Println(err, "Starting from default block.")
		currentBlock = config.FromBlock
	}

	// blocks only the first time, then runs as a go func
	syncWorker(currentBlock, false)

	// First time through we start the server once synchronized
	startServer()
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
