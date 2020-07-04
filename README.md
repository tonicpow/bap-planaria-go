## BAP Transaction Indexer & State Machine

A Planaria-like indexer and transaction processor. Uses Bitbus to ingest all Bitcoin Attestation Protocol transactions. Once sync is complete it replays history to create a state collection that can then be queried.

## Dependencies

- Mongo DB
- BMAP - BOB Transaction parser
  - AIP - Author Identity Protocol library
  - BAP - Bitcoin Attestation Protocol library
  - MAP - Magic Attribute Protocol library

## Environmental Variables

BAP_MONGO_URL

    A valid mongodb connection string.
    Example: "mongodb://localhost:27017/bap"

## Usage

```
go run main.go
```
