package attestation

import (
	"github.com/rohenaz/go-bap"
	"github.com/rohenaz/go-bob"
)

type State struct {
	bap.Data
	Blk bob.Blk
	Tx  bob.Tx
}
