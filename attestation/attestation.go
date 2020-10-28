package attestation

import (
	"github.com/bitcoinschema/go-bap"
	"github.com/rohenaz/go-bob"
)

type State struct {
	bap.Data
	Blk bob.Blk
	Tx  bob.BobTx
}
