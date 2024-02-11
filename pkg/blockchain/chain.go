package blockchain

import (
	"btc-indexer/database"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

type BlockLocator []*chainhash.Hash

type chain struct {
	store    database.Store
	checkpts []chaincfg.Checkpoint
}

func NewChain(store database.Store, checkpts []chaincfg.Checkpoint) Chain {
	return &chain{
		store:    store,
		checkpts: checkpts,
	}
}

func (c *chain) getBlockLocator(height int32) ([]*chainhash.Hash, error) {
	var maxEntries uint8
	if height <= 12 {
		maxEntries = uint8(height) + 1
	} else {
		adjustedHeight := uint32(height) - 10
		maxEntries = 12 + fastLog2Floor(adjustedHeight)
	}
	locator := make(BlockLocator, 0, maxEntries)
	if height < 0 {
		return locator, nil
	}

	step := int32(1)

	for height >= 0 {
		blockHash, err := c.store.GetBlockHashByHeight(height)
		if err != nil {
			return nil, err
		}

		chainhash, err := chainhash.NewHashFromStr(blockHash)
		if err != nil {
			return nil, err
		}

		locator = append(locator, chainhash)

		if height == 0 {
			break
		}

		height := height - step
		if height < 0 {
			height = 0
		}

		if len(locator) > 10 {
			step *= 2
		}
	}

	return locator, nil
}

func (c *chain) findNextHeaderCheckpoint(height int32) *chaincfg.Checkpoint {
	checkpoints := c.checkpts
	if len(checkpoints) == 0 {
		return nil
	}

	finalCheckpoint := &checkpoints[len(checkpoints)-1]
	if height >= finalCheckpoint.Height {
		return nil
	}

	nextCheckpoint := finalCheckpoint
	for i := len(checkpoints) - 2; i >= 0; i-- {
		if height >= checkpoints[i].Height {
			break
		}
		nextCheckpoint = &checkpoints[i]
	}
	return nextCheckpoint
}

var log2FloorMasks = []uint32{0xffff0000, 0xff00, 0xf0, 0xc, 0x2}

func fastLog2Floor(n uint32) uint8 {
	rv := uint8(0)
	exponent := uint8(16)
	for i := 0; i < 5; i++ {
		if n&log2FloorMasks[i] != 0 {
			rv += exponent
			n >>= exponent
		}
		exponent >>= 1
	}
	return rv
}
