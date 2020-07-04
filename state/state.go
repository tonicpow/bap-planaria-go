package state

import (
	"context"
	"log"
	"time"

	"github.com/rohenaz/bap-planaria-go/bap"
	"github.com/rohenaz/bap-planaria-go/database"
)

// Build starts the state builder
func Build() {

	var numPerPass int = 100
	// Query x records at a time in a loop
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	conn, err := database.Connect(ctx)
	if err != nil {
		return
	}

	defer conn.Disconnect(ctx)
	// Find number of passes
	numIdentities, err := conn.CountCollectionDocs(bap.ID)
	if err != nil {
		log.Println("Error", err)
	}

	for i := 0; i < (int(numIdentities)/numPerPass)+1; i++ {
		log.Println("Page", i)
		skip := i * numPerPass
		bmapTxs, err := conn.GetDocs(bap.ATTEST, int64(numPerPass), int64(skip))
		if err != nil {
			log.Println("Error:", err)
			return
		}
		log.Printf("Got Docs! %+v", bmapTxs[0].BAP)
	}
}
