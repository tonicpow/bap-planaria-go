package state

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bitcoinschema/go-aip"
	"github.com/bitcoinschema/go-bap"
	"github.com/bitcoinschema/go-bitcoin"
	"github.com/bitcoinschema/go-bmap"
	"github.com/tonicpow/bap-planaria-go/attestation"
	"github.com/tonicpow/bap-planaria-go/config"
	"github.com/tonicpow/bap-planaria-go/database"
	"github.com/tonicpow/bap-planaria-go/identity"
	"github.com/tonicpow/bap-planaria-go/miner"
	"github.com/tonicpow/bap-planaria-go/persist"
	"go.mongodb.org/mongo-driver/bson"
)

// Attestation is the data for an attestation
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
//   IDKey: f10e4e49d7d024821818452fb57a9ec5b6c4f5168f8a8a48fb3dd69a918effef, <- this is a lookup of attestation
//   AIP address from ID collection, and timing makes it valid at time of signing
//   attestationHash: 9b7d5b90c2aca598f2990bb06dc2e5dfd6db21c138d96b3a32dba25d4f75ef1c
// }

// SaveProgress persists the block height to ./block.tmp
func SaveProgress(height uint32) {
	if height > 0 {
		if config.UseDBForState {
			// persist our progress to the database
			// TODO save height to _state collection
			// { _id: 'height', value: height }
		} else {
			// persist our progress to disk
			if err := persist.Save("./block.tmp", height); err != nil {
				log.Fatalln(err)
				// todo: return an error?
			}
		}
	}

}

func validateIDTx(idTx bmap.Tx) (valid bool) {

	// Make sure BAP Address is a valid Bitcoin address
	addressValid, _ := bitcoin.ValidA58([]byte(idTx.BAP.Address))
	// todo: log or return error?

	// Make sure Id Key is a valid length
	return len(idTx.BAP.IDKey) == 64 && idTx.AIP.Validate() && addressValid
}

// Build starts the state builder
func build(fromBlock int, trust bool) (stateBlock int) {

	// if there are no txs to process, return the same thing we sent in
	stateBlock = fromBlock

	var numPerPass = 100
	var err error

	// Create a DB connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer func() {
		cancel()
	}()
	var conn *database.Connection
	if conn, err = database.Connect(ctx); err != nil {
		log.Println(err.Error())
		return
	}
	defer func() {
		_ = conn.Disconnect(ctx)
	}()

	// Clear old state
	if fromBlock == 0 {
		// todo: replace with logging package
		log.Println("clearing state")
		if err = conn.ClearState(); err != nil {
			log.Println(err.Error())
			// return
		}
	}

	// TODO: This is fixed ingesting only 1000 ids
	// Make Identity State first
	var bmapIDTxs []bmap.Tx
	if bmapIDTxs, err = conn.GetDocs(
		string(bap.ID),
		1000,
		0,
		bson.M{"blk.i": bson.M{"$gt": fromBlock}},
	); err != nil {
		log.Println(err)
		return
	}

	// Loop on map txs
	for idx, idTx := range bmapIDTxs {

		// Validate
		if validateIDTx(idTx) {

			var idHistory []identity.Identity

			// See if we have this tx already, if so skip it
			numTxs, _ := conn.CountCollectionDocs("identityState", bson.M{"Tx.h": idTx.Tx.H})
			// todo: handle error?
			if numTxs != 0 {
				// We already have this one
				log.Println("this tx is already in the state", idTx.Tx.H, idTx.Blk.I)
				SaveProgress(idTx.Blk.I)
				continue
			}

			if !trust {
				// make sure the tx exists in the blockchain
				var foundInBlock uint32
				if foundInBlock, err = miner.VerifyExistence(idTx.Tx.H); err != nil || foundInBlock == 0 {
					// This tx does not exist in the blockchain!
					log.Println("either this tx does not exist on the blockchain, or there was an error checking!", err, idTx.Tx.H)
					continue
				} else {
					pct := idx * 100 / len(bmapIDTxs)
					log.Printf("mempool confirms %s in block %d %d\n", idTx.Tx.H, foundInBlock, pct)
				}
			}

			// Check if ID key exists
			idState, _ := conn.GetIdentityState(idTx.BAP.IDKey)
			// todo: check error?

			if idState == nil {
				// This has to be the first time this ID is seen on-chain, otherwise the
				// IDControlAddress will not be correct
				_, err = conn.InsertOne("identityState", bson.M{
					"_id":            idTx.BAP.IDKey,
					"controlAddress": idTx.AIP.AlgorithmSigningComponent,
					"currentAddress": idTx.BAP.Address,
					"idKey":          idTx.BAP.IDKey,
					"history": append(idHistory, identity.Identity{
						Tx:        idTx.Tx.H,
						Address:   idTx.BAP.Address,
						FirstSeen: idTx.Blk.I,
						LastSeen:  0,
					}),
				})
				if err != nil {
					log.Println("insert failed", err.Error())
					// return
				}
			} else {
				// update identity history

				// TODO: validate if this is a valid chain of signatures

				// Get prev address
				// TODO it might be better to search for the last array element with LastSeen = 0 ??
				prevAddress := idState.IDHistory[len(idState.IDHistory)-1].Address
				if idTx.AIP.AlgorithmSigningComponent == prevAddress {
					// - when a key is replaced it is signed with the previous address/key

					// TODO We should try to do this in 1 transaction
					filter := bson.M{"_id": idState.IDKey}
					_, err = conn.Update("identityState", filter, bson.M{
						"$addToSet": identity.Identity{
							Tx:        idTx.Tx.H,
							Address:   idTx.BAP.Address,
							FirstSeen: idTx.Blk.I,
							LastSeen:  0,
						},
					})
					if err != nil {
						log.Println("update failed", err.Error())
						// return
					}

					filterSet := bson.M{"_id": idState.IDKey, "IDHistory.Address": prevAddress}
					_, err = conn.Update("identityState", filterSet, bson.M{
						"$set": bson.M{
							"currentAddress":     idTx.BAP.Address,
							"history.$.lastSeen": idTx.Blk.I,
						},
					})
					if err != nil {
						log.Println("update failed", err.Error())
						// return
					}
				} else {
					err = fmt.Errorf("must use previous address to change an identity "+
						"address: %s %s %+v", prevAddress, idTx.AIP.AlgorithmSigningComponent, idState)

					// Upsert as identity state document
					filter := bson.M{"Tx.h": bson.M{"$eq": idTx.Tx.H}}
					_, err = conn.UpsertOne("stateErrors", filter, bson.M{
						"Tx":    idTx.Tx,
						"Error": err.Error(),
					})
					if err != nil {
						log.Println("upsert one failed", err.Error())
						// return
					}
				}
			}

			// persist our progress to disk
			SaveProgress(idTx.Blk.I)
			stateBlock = int(idTx.Blk.I)
		}
	}

	// Find number of passes - should get this from state not raw
	numIdentities, err := conn.CountCollectionDocs(string(bap.ID), bson.M{})
	if err != nil {
		log.Println("Error", err)
	}

	for i := 0; i < (int(numIdentities)/numPerPass)+1; i++ {

		skip := i * numPerPass
		var attestationTxs []bmap.Tx
		attestationTxs, err = conn.GetDocs(string(bap.ATTEST), int64(numPerPass), int64(skip), bson.M{})
		if err != nil {
			log.Println(err)
			return
		}

		// var identities []Identity
		for _, tx := range attestationTxs {

			if tx.AIP.Validate() {

				// 1. Look up related Identity (find an identity with the AIP address in history)
				var idState *identity.State
				if idState, err = conn.GetIdentityStateFromAddress(tx.AIP.AlgorithmSigningComponent); err != nil {
					log.Println("no identity found for address", tx.AIP.AlgorithmSigningComponent, tx.Tx.H)
					continue
				}

				// TODO: look for the IDHistory element with the address, not 0
				firstSeen := int(idState.IDHistory[0].FirstSeen)
				lastSeen := int(idState.IDHistory[0].LastSeen)

				// 2. TODO: Check that current block is between the firstSeen and lastSeen?
				if int(tx.Blk.I) > firstSeen && int(tx.Blk.I) > lastSeen {

					var addr string
					if tx.AIP.Algorithm == aip.Paymail {
						addr = tx.AIP.AlgorithmSigningComponent
					} else if tx.AIP.Algorithm == aip.BitcoinECDSA {
						addr = tx.BAP.Address
					}
					_, err = conn.Upsert("attestationState", bson.M{
						"_id": tx.BAP.URNHash,
					}, bson.M{
						"$setOnInsert": bson.M{
							"_id": tx.BAP.URNHash,
						},
						"$addToSet": bson.M{
							"attestations": bson.M{
								"idKey":     idState.IDKey,
								"address":   addr,
								"signature": tx.AIP.Signature,
								"tx":        tx.Tx.H,
								"sequence":  tx.BAP.Sequence,
								"blk":       tx.Blk.I,
							},
						},
					})
					if err != nil {
						log.Println("upsert failed", err)
						// return
					}
				}

			}

		}

		skip = 0
		var revokeTxs []bmap.Tx
		revokeTxs, err = conn.GetDocs(string(bap.REVOKE), int64(numPerPass), int64(skip), bson.M{})
		if err != nil {
			log.Println(err)
			return
		}

		// var identities []Identity
		for _, tx := range revokeTxs {

			// Get the attestation we are revoking, sorted by most recent
			var attestationState *attestation.State
			if attestationState, err = conn.GetAttestationState(tx.BAP.URNHash); err != nil {
				log.Println("error revoking attestation", err)
				continue
			}

			// TODO find the attestation for the idKey
			// attestation := attestationState.attestations.find(...)
			// check its block height against this
			if tx.Blk.I > attestationState.Blk.I {
				// revoke is newer
				log.Println("revoke is newer")
			}
		}
	}
	return
}

// SyncState will sync from a given block
func SyncState(fromBlock int) (newBlock int) {
	// Set up timer for state sync
	stateStart := time.Now()

	// set tru to trust planaria, false to verify every tx with a miner
	newBlock = build(fromBlock, config.TrustPlanaria)

	// Difference in time
	diff := time.Since(stateStart).Seconds()

	log.Printf("State sync complete to block height %d in %fs\n", newBlock, diff)

	// update the state block counter
	SaveProgress(uint32(newBlock))

	return
}

// Mempool
// {
// 	"payload":"{\"apiVersion\":\"0.1.0\",\"timestamp\":\"2020-09-26T18:12:24.318Z\",\"returnResult\":\"success\",
// 	\"resultDescription\":\"\",\"blockHash\":\"0000000000000000003110de9dc837ccf2e4930d925da49dc7a2201884fad266\",
// 	\"blockHeight\":590194,\"confirmations\":64112,\"minerId\":null,\"txSecondMempoolExpiry\":0}",
// 	"signature":null,
// 	"publicKey":null,
// 	"encoding":"UTF-8",
// 	"mimetype":"application/json"
// }

// Taal
// {
// 	"payload": "{\"apiVersion\":\"0.1.0\",\"timestamp\":\"2020-09-26T17:39:47.065Z\",\"returnResult\":\"failure\",
// 	\"resultDescription\":\"ERROR: No such mempool or blockchain transaction. Use gettransaction for
// 	wallet transactions.\",\"blockHash\":null,\"blockHeight\":null,\"confirmations\":0,\"minerId\":
// 	\"03e92d3e5c3f7bd945dfbf48e7a99393b1bfb3f11f380ae30d286e7ff2aec5a270\",\"txSecondMempoolExpiry\":0}",
// 	"signature": "3045022100e77cb90a4e8f6e1e9eb02e839b50dba023deb769720ea3b2270bb31a3a5808f10220165ed53b5cd62ab7a0df6c
// 	4fac208604f616d4e0b706445892c92938312f749e",
// 	"publicKey": "03e92d3e5c3f7bd945dfbf48e7a99393b1bfb3f11f380ae30d286e7ff2aec5a270",
// 	"encoding": "UTF-8",
// 	"mimetype": "application/json"
// }
