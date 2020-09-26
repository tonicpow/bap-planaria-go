package matter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/bitcoinsv/bsvutil"
)

// Matter cloud block hashes
// {
// "status": 200,
// "errors": [],
// "result": [
// 	{
// 		"height": 650000,
// 		"hash": "00000000000000000310c17bbb4f3f8e5371a41ec2cee36a39876042019b725b",
// 		"size": 430268,
// 		"version": 566484992,
// 		"merkleroot": "58b0688e0590a9b1302e243afe9a0900728afc950f94f1080d2e8e8f7a2acd2f",
// 		"time": 1598573981,
// 		"nonce": 367658839,
// 		"bits": "1804b6eb",
// 		"difficulty": "233214426358.1377",
// 		"previousblockhash": "0000000000000000029031f1c09178d252d4d34147a3992b93001dc3e16d74a2",
// 		"nextblockhash": "000000000000000004653fdb4b2358770c8a1e80d2540b696d2ce879d2173ca0",
// 		"coinbaseinfo": "0310eb09049a4d485f485a2f48756f42692ffabe6d6dd3fc78f2c7a896830ab39509e983c7c4d1a3a3cb9fed8fcdd38594959b9fda930200000090fed21806a8ada70500000000000000",
// 		"coinbasetxid": "9834daa6d34690981888f7db4c1c36686ebb9b685d37115abc38e0e75f9cd98d",
// 		"chainwork": "00000000000000000000000000000000000000000117fbcf44cf317be90c16ab"
// 	}]
// }

// Matter latest block
// {"status":200,"errors":[],"result":[{"height":654018,"hash":"0000000000000000009e3ba1ea2518c0a73212e0a38e3dc3aee2e7df3d883d7c","size":10554327,"version":549453824,"merkleroot":"640043cfb875916170a57d1f9ba60118422dd67075927471688767a485e5eb96","time":1600981747,"nonce":2013154616,"bits":"1804444b","difficulty":"257687900404.1103","previousblockhash":"000000000000000001531b9571684285d0b64cba3b08556c7451a6601a4d00e4","nextblockhash":"","coinbaseinfo":"03c2fa0904f40a6d5f626a2f4254432e636f6d2ffabe6d6d33c952d467a829a68681f889154a99ac11cd71ab369c697f6c128c2bd777133502000000020bb56c04ab3c0053fe052eef320000","coinbasetxid":"00eb4e6a607f8b2089e237dad9e71e17b18f08c779a2f96a985a9e6bf70be3cf","chainwork":"0000000000000000000000000000000000000000011bae229bd17d4c9df55688"}]}

type Result struct {
	Height        uint64 `json:"height"`
	Hash          string `json:"hash"`
	Size          int    `json:"size"`
	Version       int    `jsong:"version"`
	MerkleRoot    string `json:"merkleroot"`
	Time          uint64 `json:"time"`
	Nonce         uint64 `json:"nonce"`
	Bits          string `json:"bits"`
	Difficulty    string `json:"difficulty"`
	NextBlockHash string `json:"nextblockhash"`
	CoinbaseInfo  string `json:"coinbaseinfo"`
	CoinbaseTxID  string `json:"coinbasetxid"`
	Chainwork     string `json:"chainwork"`
}
type MatterCloudBlockResult struct {
	Status int      `json:"status"`
	Errors []string `json:"errors"`
	Result []Result `json:"result"`
}

func getBlockHeaders() {

	resp, err := http.Get("https://txdb.mattercloud.io/api/v1/blockheader/")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	response := &MatterCloudBlockResult{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		panic(err)
	}

	if len(response.Result) == 0 {
		panic("that just happened")
	}

	latestBlock := response.Result[0]

	fmt.Println("get:\n", string(latestBlock.Height))

	var readingBlockHeight = latestBlock.Height

	for readingBlockHeight > 0 {
		log.Println("Reading headers from block ", readingBlockHeight)
		resp, err = http.Get("https://txdb.mattercloud.io/api/v1/blockheader/" + strconv.FormatUint(readingBlockHeight, 10) + "?limit=100&order=desc%7Casc&pretty")
		if err != nil {
			panic(err)
		}

		defer resp.Body.Close()
		body, err = ioutil.ReadAll(resp.Body)

		response = &MatterCloudBlockResult{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			panic(err)
		}

		if len(response.Result) == 0 {
			panic("that just happened 2")
		}

		for _, block := range response.Result {
			log.Println(block.Hash)
			if block.Height < readingBlockHeight {
				readingBlockHeight = block.Height
			}
		}
	}

}

func getBlock(blockhash string) (err error, block *bsvutil.Block) {

	resp, err := http.Get("https://txdb.mattercloud.io/api/v1/blockheader/?limit=100&order=desc%7Casc&pretty")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	fmt.Println("get:\n", string(body))

	block = &bsvutil.Block{}

	return nil, block
}
