package miner

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/tonicpow/bap-planaria-go/config"
)

// MapiTxStatus is the mAPI status
type MapiTxStatus struct {
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
	PublicKey string `json:"publicKey"`
	Encoding  string `json:"encoding"`
	MimeType  string `json:"mimetype"`
}

// MapiStatusPayload is the payload from mAPI
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

// VerifyExistence checks with a miner that the given txid is in the blockchain
func VerifyExistence(tx string) (uint32, error) {

	// "https://merchantapi.taal.com/mapi/tx/" + tx
	url := config.MinerAPIEndpoint + tx

	// Create a request
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, strings.NewReader(""))
	if err != nil {
		return 0, err
	}

	// Add headers
	request.Header.Add("token", config.MempoolToken)
	request.Header.Add("Content-Type", "application/json")

	// Fire the request
	var res *http.Response
	client := &http.Client{}
	if res, err = client.Do(request); err != nil {
		return 0, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	// read the body
	var body []byte
	if body, err = ioutil.ReadAll(res.Body); err != nil {
		return 0, err
	}

	// Parse the status
	txStatus := &MapiTxStatus{}
	if err = json.Unmarshal(body, txStatus); err != nil {
		return 0, err
	}

	// Parse the payload
	pl := &MapiStatusPayload{}
	if err = json.Unmarshal([]byte(txStatus.Payload), pl); err != nil {
		return 0, err
	}

	return pl.BlockHeight, nil
}
