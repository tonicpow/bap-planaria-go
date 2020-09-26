## BAP Transaction Indexer & State Machine

A Planaria-like indexer and transaction processor. Uses Bitbus to ingest all Bitcoin Attestation Protocol transactions. Once sync is complete it replays history to create a state collection that can then be queried.

## Dependencies

- Mongo DB
- Bitbus - Get a token [here](https://token.planaria.network/)
- [BMAP](https://github.com/rohenaz/go-bmap) - BOB Transaction parser
  - [AIP](https://github.com/rohenaz/go-aip) - Author Identity Protocol library
  - [BAP](https://github.com/rohenaz/go-bap) - Bitcoin Attestation Protocol library
  - [MAP](https://github.com/rohenaz/go-map) - Magic Attribute Protocol library

## Installation

Set the required environmental variables:

BAP_MONGO_URL

    A valid mongodb connection string.
    Example: "mongodb://localhost:27017/bap"

PLANARIA_TOKEN

    A token from https://token.planaria.network

## Start indexer

```
go run main.go
```

## Persisted sync progress

A `./block.tmp` file is created containing the latest synchronized block height. If this is set to 0 or any initial block height the sync will begin from this point.

## Clear everything and start over

Delete `./block.tmp`, (or set to the block height you want to sync from) and drop the bap db.

## Trust mode

Toggle this on to trust the data coming from Bitbus. This makes sync extremely fast. Individual txs can be validated later.

Toggle this off to contact a miner for each tx that gets ingested into the state to make sure it is in fact in a Bitcoin block.

```go
// Build starts the state builder, starting from the given block
func Build(fromBlock int, trust bool) {
```

## Start web server

```
todo
```
