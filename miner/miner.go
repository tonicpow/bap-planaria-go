package miner

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/tonicpow/bap-planaria-go/config"
)

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

//VerifyExistence checks with a miner that the given txid is in the blockchain
func VerifyExistence(tx string) (uint32, error) {

	url := config.MinerAPIEndpoint + tx // "https://merchantapi.taal.com/mapi/tx/" + tx
	payload := strings.NewReader("")

	// resp, err := http.Get(url)

	client := &http.Client{}
	request, err := http.NewRequest(http.MethodGet, url, payload)
	if err != nil {
		log.Println("Error", err)
		return 0, err
	}

	request.Header.Add("token", config.MempoolToken)
	request.Header.Add("Content-Type", "application/json")

	var res *http.Response
	if res, err = client.Do(request); err != nil {
		log.Println("Error", err)
		return 0, err
	}

	defer func() {
		_ = res.Body.Close()
	}()

	var body []byte
	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("Error", err)
		return 0, err
	}

	// fmt.Println("get:\n", string(body))

	txStatus := &MapiTxStatus{}
	err = json.Unmarshal(body, txStatus)
	if err != nil {
		log.Println("Error 999", err)
		return 0, err
	}

	pl := &MapiStatusPayload{}
	err = json.Unmarshal([]byte(txStatus.Payload), pl)
	if err != nil {
		log.Println("Error 999", err)
		return 0, err
	}

	return pl.BlockHeight, nil
}
