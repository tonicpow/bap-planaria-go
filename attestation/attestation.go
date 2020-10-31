package attestation

import (
	"github.com/bitcoinschema/go-bap"
	"github.com/bitcoinschema/go-bob"
)

// State is the main chain state struct for BAP
type State struct {
	bap.Bap
	Blk bob.Blk
	Tx  bob.Tx
}
