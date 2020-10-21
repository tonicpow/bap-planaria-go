package identity

// Identity refers to a users identity as it relates to an id key
type Identity struct {
	Tx        string     `json:"tx" bson:"tx"`
	Address   string     `json:"address" bson:"address"`
	FirstSeen uint32     `json:"firstSeen" bson:"firstSeen"`
	LastSeen  uint32     `json:"lastSeen" bson:"lastSeen"`
}

// State is the state object representing an identity key
type State struct {
	IDKey            string     `bson:"_id,omitempty"`
	IDControlAddress string     `json:"controlAddress" bson:"controlAddress"`
	IDCurrentAddress string     `json:"currentAddress" bson:"currentAddress"`
	IDHistory        []Identity `json:"history" bson:"history"`
}
