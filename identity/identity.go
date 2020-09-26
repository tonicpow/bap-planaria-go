package identity

import "github.com/rohenaz/go-bob"

// Identity refers to a users identity as it relates to an id key
type Identity struct {
	Address   string `json:"address" bson:"address"`
	FirstSeen uint32 `json:"firstSeen" bson:"firstSeen"`
	LastSeen  uint32 `json:"lastSeen" bson:"lastSeen"`
}

// State is the state object represending an identity key
type State struct {
	MongoID          string     `bson:"_id,omitempty"`
	IDControlAddress string     `json:"idControlAddress" bson:"IDControlAddress"`
	IDKey            string     `json:"idKey" bson:"IDKey"`
	IDHistory        []Identity `json:"idHistory" bson:"IDHistory"`
	Tx               bob.TxInfo `json:"tx" bson:"tx"`
}
