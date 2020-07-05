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

## Start web server

```
todo
```
