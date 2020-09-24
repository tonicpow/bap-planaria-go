package state

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/rohenaz/go-bap"
	"github.com/tonicpow/bap-planaria-go/database"
)

// TODO
type Identity struct {
	Thing string
}

type IdentityState struct {
	IDKey     string `json:"idKey"`
	IDHistory []Identity
}

// {
//   idKey: kjasfkjasfjkbasf,
//   IDHistory: [
//     {
//       address: 1jkasfjafsjhf76576576,
//       firstSeen: 700001,
//     },{
//       address: 1jknsdfgjkndsgyut767u786,
//       firstSeen: 60000,
//       lastSeen: 700000
//     }
//   ]
// }

// Attestation
// {
//   txid: 6c6e52da3f16f6a03a9ee5bfd68dd6a9fb7fce16fc66f137a265a4bf7cbb4cba
//   IDKey: f10e4e49d7d024821818452fb57a9ec5b6c4f5168f8a8a48fb3dd69a918effef, <- this is a lookup of attestation AIP address from ID collection, and timing makes it valid at time of signing
//   attestationHash: 9b7d5b90c2aca598f2990bb06dc2e5dfd6db21c138d96b3a32dba25d4f75ef1c
// }

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

		var identities []Identity
		for _, tx := range bmapTxs {
			log.Printf("Got Doc! %+v %s", tx.BAP, identities)
			valid := tx.AIP.Validate()
			log.Println("AIP Valid?", strings.Join(tx.AIP.Data, ""), valid)
		}
		// Find a previous record with the same identity

	}
}

// query
// - lookup the id of the org related to each attestation
