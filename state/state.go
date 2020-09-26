package state

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/rohenaz/go-bap"
	"github.com/rohenaz/go-bitcoin"
	"github.com/rohenaz/go-bmap"
	"github.com/tonicpow/bap-planaria-go/database"
	"github.com/tonicpow/bap-planaria-go/identity"
	"github.com/tonicpow/bap-planaria-go/persist"
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
		log.Println("Clearing state")
		conn.ClearState()
	}

	// Make Identity State first
	bmapIdTxs, err := conn.GetDocs(string(bap.ID), 1000, 0, bson.M{"blk.i": bson.M{"$gt": currentBlock}})
	if err != nil {
		log.Println("Error: 2", err)
		return
	}

	for _, idTx := range bmapIdTxs {

		if validateIDTx(idTx) {

			idHistory := []identity.Identity{}

			// See if we have this tx already, if so skip it
			numTxs, _ := conn.CountCollectionDocs("identityState", bson.M{"Tx.h": idTx.Tx.H})
			if numTxs != 0 {
				// We already have this one
				log.Println("This tx is already in the state", idTx.Tx.H)
				continue
			}

			// make sure the tx exists in the blockchain
			ok, err := verifyExistence(idTx.Tx.H)
			if err != nil || !ok {
				// This tx does not exist in the blockchain!
				fmt.Println("Either this tx does not exist on the blockchain, or there was an error checking!", err, idTx.Tx.H)
				continue
			} else {
				fmt.Println("Found in block", idTx.Tx.H)
			}

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

			// persist our progress to disk
			if err := persist.Save("./block.tmp", currentBlock); err != nil {
				log.Fatalln(err)
			}
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
		attestationTxs, err := conn.GetDocs(string(bap.ATTEST), int64(numPerPass), int64(skip), bson.M{})
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
					log.Println("No identity found for address", tx.AIP.Address, tx.Tx.H)
					continue
				}

				firstSeen := int(idState.IDHistory[0].FirstSeen)
				lastSeen := int(idState.IDHistory[0].LastSeen)

				// log.Printf("Last seen %d currentBlock %d", lastSeen, tx.Blk.I)

				// 2. TODO: Check that current block is between the firstSeen and lastSeen?
				if int(tx.Blk.I) > firstSeen && int(tx.Blk.I) > lastSeen {
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
		revokeTxs, err := conn.GetDocs(string(bap.REVOKE), int64(numPerPass), int64(skip), bson.M{})
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
				continue
			}
			// check its block height against this
			if tx.Blk.I > attestationState.Blk.I {
				// revoke is newer
				log.Println("Revoke is newer")

			}
		}
	}
}

type MapiTxStatus struct {
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
	PublicKey string `json:"publicKey"`
	Encoding  string `json:"encoding"`
	MimeType  string `json:"mimetype"`
}

type MapiStatusPayload struct {
	APIVersion            string `json:"apiVersion"`
	BlockHash             string `json:"blockHash"`
	BlockHeight           uint32 `json:"blockHeight"`
	Confirmations         uint32 `json:"confirmations"`
	MinerID               string `json:"minerId"`
	ResultDescription     string `json:"resultDescription"`
	ReturnResult          string `json:"returnResult"`
	Timestamp             string `json:"timestamp"`
	TxSecondMempoolExpiry int    `json:"txSecondMempoolExpiry"`
}

// Mempool
// {
// 	"payload":"{\"apiVersion\":\"0.1.0\",\"timestamp\":\"2020-09-26T18:12:24.318Z\",\"returnResult\":\"success\",\"resultDescription\":\"\",\"blockHash\":\"0000000000000000003110de9dc837ccf2e4930d925da49dc7a2201884fad266\",\"blockHeight\":590194,\"confirmations\":64112,\"minerId\":null,\"txSecondMempoolExpiry\":0}",
// 	"signature":null,
// 	"publicKey":null,
// 	"encoding":"UTF-8",
// 	"mimetype":"applicaton/json"
// }

// Taal
// {
// 	"payload": "{\"apiVersion\":\"0.1.0\",\"timestamp\":\"2020-09-26T17:39:47.065Z\",\"returnResult\":\"failure\",\"resultDescription\":\"ERROR: No such mempool or blockchain transaction. Use gettransaction for wallet transactions.\",\"blockHash\":null,\"blockHeight\":null,\"confirmations\":0,\"minerId\":\"03e92d3e5c3f7bd945dfbf48e7a99393b1bfb3f11f380ae30d286e7ff2aec5a270\",\"txSecondMempoolExpiry\":0}",
// 	"signature": "3045022100e77cb90a4e8f6e1e9eb02e839b50dba023deb769720ea3b2270bb31a3a5808f10220165ed53b5cd62ab7a0df6c4fac208604f616d4e0b706445892c92938312f749e",
// 	"publicKey": "03e92d3e5c3f7bd945dfbf48e7a99393b1bfb3f11f380ae30d286e7ff2aec5a270",
// 	"encoding": "UTF-8",
// 	"mimetype": "application/json"
// }

func verifyExistence(tx string) (bool, error) {

	// check w a miner that it is in fact in the blockchain
	url := "https://www.ddpurse.com/openapi/mapi/tx/" + tx // "https://merchantapi.taal.com/mapi/tx/" + tx
	payload := strings.NewReader("")

	// resp, err := http.Get(url)

	client := &http.Client{}
	request, err := http.NewRequest(http.MethodGet, url, payload)
	if err != nil {
		log.Println("Error", err)
		return false, err
	}

	request.Header.Add("token", "561b756d12572020ea9a104c3441b71790acbbce95a6ddbf7e0630971af9424b")
	request.Header.Add("Content-Type", "application/json")

	var res *http.Response
	if res, err = client.Do(request); err != nil {
		log.Println("Error", err)
		return false, err
	}

	defer func() {
		_ = res.Body.Close()
	}()

	var body []byte
	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("Error", err)
		return false, err
	}

	// fmt.Println("get:\n", string(body))

	txStatus := &MapiTxStatus{}
	err = json.Unmarshal(body, txStatus)
	if err != nil {
		log.Println("Error 999", err)
		return false, err
	}

	pl := &MapiStatusPayload{}
	err = json.Unmarshal([]byte(txStatus.Payload), pl)
	if err != nil {
		log.Println("Error 999", err)
		return false, err
	}

	var inBlock bool
	if pl.BlockHeight != 0 {
		inBlock = true
	}

	return inBlock, nil
}
