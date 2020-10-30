package attestation

import (
	"github.com/bitcoinschema/go-bap"
	"github.com/bitcoinschema/go-bob"
)

type State struct {
	bap.Bap
	Blk bob.Blk
	Tx  bob.Tx
}
