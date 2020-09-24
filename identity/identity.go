package identity

// Identity refers to a users identity as it relates to an id key
type Identity struct {
	Address   string `json:"address"`
	FirstSeen uint32 `json:"firstSeen"`
	LastSeen  uint32 `json:"lastSeen"`
}

// State is the state object represending an identity key
type State struct {
	IDControlAddress string     `json:"idControlAddress"`
	IDKey            string     `json:"idKey"`
	IDHistory        []Identity `json:"idHistory"`
}
