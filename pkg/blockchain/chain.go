package blockchain

import (
	"btc-indexer/database"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

type BlockLocator []*chainhash.Hash

type chain struct {
	store database.Store
}

func NewChain(store database.Store) Chain {
	return &chain{
		store: store,
	}
}

func (c *chain) getBlockLocator(height int32) (BlockLocator, error) {

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
		//Todo: Get this hash from Db for block height for not orphan block
		block, err := c.store.GetBlockByHeight(height)
		if err != nil {
			return nil, err
		}

		chainhash, err := chainhash.NewHashFromStr(block.ID)
		if err != nil {
			return nil, err
		}

		locator = append(locator, chainhash)

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
