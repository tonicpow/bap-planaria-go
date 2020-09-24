package state

import (
	"context"
	"log"
	"time"

	"github.com/rohenaz/go-bap"
	"github.com/rohenaz/go-bmap"
	"github.com/tonicpow/bap-planaria-go/database"
)

type Attestation struct {
	TxID            string
	IDKey           string
	AttestationHash string
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

func validateIdTx(idTx bmap.Tx) (valid bool) {
	// Make sude Id Key is a valid length
	return len(idTx.BAP.IDKey) == 64 && idTx.AIP.Validate()

	// TODO: Makle sure idTx.BAP.Address is a valid address

	// See if this ID key is already in the state
}

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

	// Make Identity State first
	bmapIdTxs, err := conn.GetDocs(string(bap.ID), 1000, 0)
	if err != nil {
		log.Println("Error:", err)
		return
	}

	for _, idTx := range bmapIdTxs {

		if validateIdTx(idTx) {
			// Check if ID key exists
			bmapTx, err := conn.GetIdentityState(idTx.BAP.IDKey)
			if err != nil {
				log.Println("Error:", err)
				return
			}

			log.Printf("Yo %+v", bmapTx)

			// if bmapTx {

			// } else {
			// }

		}

	}

	// Find number of passes - should get this from state not raw
	numIdentities, err := conn.CountCollectionDocs(string(bap.ID))
	if err != nil {
		log.Println("Error", err)
	}

	for i := 0; i < (int(numIdentities)/numPerPass)+1; i++ {
		log.Println("Page", i)
		skip := i * numPerPass
		bmapTxs, err := conn.GetDocs(string(bap.ATTEST), int64(numPerPass), int64(skip))
		if err != nil {
			log.Println("Error:", err)
			return
		}

		// var identities []Identity
		for _, tx := range bmapTxs {
			// log.Printf("Got Doc! %+v %s", tx.BAP, identities)
			if tx.AIP.Validate() {

				// 1. Look up related Identity (find an idetity with the AIP address in history?)
				// 2. Check that current block is between the firstSeen and lastSeen

				// Save to state collection

				switch tx.BAP.Type {
				case bap.ATTEST:
					log.Printf("Attestation! Attestor: %s Hash: %s", tx.AIP.Address, tx.BAP.URNHash)
					break
				case bap.REVOKE:
					log.Println("Revocation")
					break
				case bap.ID:
					log.Println("ID!")
					break
				}
			}

		}
		// Find a previous record with the same identity

	}
}

// query
// - lookup the id of the org related to each attestation
