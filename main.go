package main

import (
	"log"
	"net/http"
	"time"

	"github.com/tonicpow/bap-planaria-go/crawler"
	"github.com/tonicpow/bap-planaria-go/persist"
	"github.com/tonicpow/bap-planaria-go/router"
	"github.com/tonicpow/bap-planaria-go/state"
)

func syncWorker(currentBlock int) {
	// crawl
	newBlock := crawler.SyncBlocks(currentBlock)
	state.SyncState(currentBlock)

	time.Sleep(30 * time.Second)
	go syncWorker(newBlock)
}

func main() {
	var currentBlock int

	// load persisted block to continue from
	if err := persist.Load("./block.tmp", &currentBlock); err != nil {
		log.Println(err, "Starting from default block.")
	}

	// blocks only the first time, then runs as a go func
	syncWorker(currentBlock)

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
