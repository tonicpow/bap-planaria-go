## BAP Transaction Indexer & State Machine

A Planaria-like indexer and transaction processor. Uses Bitbus to ingest all Bitcoin Attestation Protocol transactions. Once sync is complete it replays history to create a state collection that can then be queried.

## Dependencies

- Mongo DB
- Bitbus - Get a token [here](https://token.planaria.network/)
- [BMAP](https://github.com/bitcoinschema/go-bmap) - BOB Transaction parser
  - [AIP](https://github.com/bitcoinschema/go-aip) - Author Identity Protocol library
  - [BAP](https://github.com/bitcoinschema/go-bap) - Bitcoin Attestation Protocol library
  - [MAP](https://github.com/bitcoinschema/go-map) - Magic Attribute Protocol library

## Installation

Set the required environmental variables:

BAP_MONGO_URL

    A valid mongodb connection string.
    Example: "mongodb://localhost:27017/bap"

PLANARIA_TOKEN

    A token from https://token.planaria.network

## Start indexer & api server

API server will start once sync is complete.

```
go run main.go
```

## Persisted sync progress

A `./block.tmp` file is created containing the latest synchronized block height. If this is set to 0 or any initial block height the sync will begin from this point.

## Clear everything and start over

Delete `./block.tmp`, (or set to the block height you want to sync from) and drop the bap db.

## Trust mode

Toggle this on to trust the blockchain data coming from Bitbus as accurate & authentic. This makes sync extremely fast at the expense of personal verification.

Toggle this off to contact a miner for each tx that gets ingested into the state to make sure it is in a Bitcoin block.

```go
// Build starts the state builder, starting from the given block
func Build(fromBlock int, trust bool) {
```

## Query API

### Endpoints

Identity State: `/find/identityState`

#### Parameters:

- query
- limit
- skip

### Example

You can query using a mongo style find query like this:

```json
{ "Tx.h": "ef1f414c51aabbf5d2f02dd448baf7926b1f5b492d9359a7a0b533d35e14d0f5" }
```

base64 encode that query:

`eyJUeC5oIjoiZWYxZjQxNGM1MWFhYmJmNWQyZjAyZGQ0NDhiYWY3OTI2YjFmNWI0OTJkOTM1OWE3YTBiNTMzZDM1ZTE0ZDBmNSJ9`

and send it as the `query` parameter

```
/find/identityState?limit=100&skip=0&query=eyJUeC5oIjoiZWYxZjQxNGM1MWFhYmJmNWQyZjAyZGQ0NDhiYWY3OTI2YjFmNWI0OTJkOTM1OWE3YTBiNTMzZDM1ZTE0ZDBmNSJ9
```

and get a response like:

```json
[
  {
    "IDControlAddress": "1PSiEkLGqxL6FFpsohy1BG4mpPyhP9ChK5",
    "IDHistory": [
      {
        "address": "1DjVDse2n9FCFwT4214Qsh7WRSSfLN5eD9",
        "firstSeen": 590194,
        "lastSeen": 0
      }
    ],
    "IDKey": "66a32aaafc3fa84f72a01bb49a93b71087f2d72afe874e6ef81a15cc5fa90517",
    "Tx": {
      "h": "ef1f414c51aabbf5d2f02dd448baf7926b1f5b492d9359a7a0b533d35e14d0f5"
    },
    "_id": "5f6f66dc56de7760019b6342"
  }
]
```
