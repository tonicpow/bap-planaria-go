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
