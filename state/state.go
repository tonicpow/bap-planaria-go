package state

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rohenaz/go-bap"
	"github.com/rohenaz/go-bitcoin"
	"github.com/rohenaz/go-bmap"
	"github.com/tonicpow/bap-planaria-go/database"
	"github.com/tonicpow/bap-planaria-go/identity"
	"go.mongodb.org/mongo-driver/bson"
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

func validateIDTx(idTx bmap.Tx) (valid bool) {

	// Make sure BAP Address is a valid Bitcoin address
	addressValid, _ := bitcoin.ValidA58([]byte(idTx.BAP.Address))

	// Make sude Id Key is a valid length
	return len(idTx.BAP.IDKey) == 64 && idTx.AIP.Validate() && addressValid
}

// Build starts the state builder
func Build(currentBlock int) {

	var numPerPass int = 100
	// Query x records at a time in a loop
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	conn, err := database.Connect(ctx)
	if err != nil {
		return
	}
	defer conn.Disconnect(ctx)

	// Clear old state
	if currentBlock == 0 {
		conn.ClearState()
	}

	// Make Identity State first
	bmapIdTxs, err := conn.GetDocs(string(bap.ID), 1000, 0)
	if err != nil {
		log.Println("Error: 2", err)
		return
	}

	for _, idTx := range bmapIdTxs {

		if validateIDTx(idTx) {

			idHistory := []identity.Identity{}

			// Check if ID key exists
			idState, _ := conn.GetIdentityState(idTx.BAP.IDKey)

			updatedIdentity := bson.M{
				"IDControlAddress": idTx.AIP.Address,
				"IDKey":            idTx.BAP.IDKey,
				"IDHistory":        idHistory,
				"Tx":               idTx.Tx,
			}

			// If its a record for the same ID key (excluding the very same tx)
			if idState != nil && idState.IDKey == idTx.BAP.IDKey && idState.Tx.H != idTx.Tx.H && len(idState.IDHistory) > 0 {

				// ok, hard part...
				// update identity history

				// TODO: validate if this is a valid chain of signatures

				// Get prev address
				prevAddress := idState.IDHistory[len(idState.IDHistory)-1].Address
				if idTx.AIP.Address == idState.IDControlAddress {
					// - when a key is replaced it is signed with the previous address/key
					updatedIdentity["IDHistory"] = append(idHistory, identity.Identity{
						Address:   idState.IDControlAddress,
						FirstSeen: idState.IDHistory[len(idState.IDHistory)-1].FirstSeen,
						LastSeen:  idTx.Blk.I,
					})

					updatedIdentity["IDControlAddress"] = idTx.BAP.Address
				} else {
					err = fmt.Errorf("Must use control address to change an identity %s %s %+v", prevAddress, idTx.AIP.Address, idState)
					// Upsert as identity state document
					filter := bson.M{"Tx.h": bson.M{"$eq": idTx.Tx.H}}
					conn.UpsertOne("stateErrors", filter, bson.M{
						"Tx":    idTx.Tx,
						"Error": err.Error(),
					})
				}

			} else {
				// Brand new identity key
				updatedIdentity["IDHistory"] = append(idHistory, identity.Identity{
					Address:   idTx.BAP.Address,
					FirstSeen: idTx.Blk.I,
					LastSeen:  0,
				})
			}

			// Upsert as identity state document
			filter := bson.M{"IDKey": bson.M{"$eq": idTx.BAP.IDKey}}
			conn.UpsertOne("identityState", filter, updatedIdentity)
		}
	}

	// Find number of passes - should get this from state not raw
	numIdentities, err := conn.CountCollectionDocs(string(bap.ID), bson.M{})
	if err != nil {
		log.Println("Error", err)
	}

	for i := 0; i < (int(numIdentities)/numPerPass)+1; i++ {
		// log.Println("Page", i)
		skip := i * numPerPass
		attestationTxs, err := conn.GetDocs(string(bap.ATTEST), int64(numPerPass), int64(skip))
		if err != nil {
			log.Println("Error: 3", err)
			return
		}

		// var identities []Identity
		for _, tx := range attestationTxs {
			// log.Printf("Got Doc! %+v %s", tx.BAP, identities)
			if tx.AIP.Validate() {

				// 1. Look up related Identity (find an identity with the AIP address in history)
				// log.Printf("Find id %+v", tx.AIP.Address)
				idState, err := conn.GetIdentityStateFromAddress(tx.AIP.Address)
				if err != nil {
					log.Println("Error: 4", err)
					return
				}

				firstSeen := int(idState.IDHistory[0].FirstSeen)
				// lastSeen := int(idState.IDHistory[0].LastSeen)

				// log.Printf("Last seen %d currentBlock %d", lastSeen, tx.Blk.I)

				// 2. TODO: Check that current block is between the firstSeen and lastSeen
				if int(tx.Blk.I) > firstSeen {
					// log.Println("Valid ID state!", idState)

					conn.UpsertOne("attestationState", bson.M{"Tx.h": tx.Tx.H}, bson.M{
						"urnHash":  tx.BAP.URNHash,
						"Tx":       tx.Tx,
						"sequence": tx.BAP.Sequence,
						"Blk":      tx.Blk,
					})
				}

			}

		}

		skip = 0
		revokeTxs, err := conn.GetDocs(string(bap.REVOKE), int64(numPerPass), int64(skip))
		if err != nil {
			log.Println("Error: 3", err)
			return
		}

		// var identities []Identity
		for _, tx := range revokeTxs {

			// Get the attestation we are revoking, sorted by most recent
			attestationState, err := conn.GetAttestationState(tx.BAP.URNHash)
			if err != nil {
				log.Println("Error revoking attestation", err)
			}
			// check its block height against this
			if tx.Blk.I > attestationState.Blk.I {

				// revoke is newer
				log.Println("Revoke is newer")

			}
		}
	}
}

// query
// - lookup the id of the org related to each attestation
