package blockchain

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/peer"
	"github.com/btcsuite/btcd/wire"
)

func newPeerConfig(params chaincfg.Params) *peer.Config {
	return &peer.Config{
		Listeners: peer.MessageListeners{
			// OnVersion:      sp.OnVersion,
			// OnVerAck:       sp.OnVerAck,
			// OnMemPool:      sp.OnMemPool,
			// OnTx:           sp.OnTx,
			// OnBlock:        sp.OnBlock,
			// OnInv:          sp.OnInv,
			// OnHeaders:      sp.OnHeaders,
			// OnGetData:      sp.OnGetData,
			// OnGetBlocks:    sp.OnGetBlocks,
			// OnGetHeaders:   sp.OnGetHeaders,
			// OnGetCFilters:  sp.OnGetCFilters,
			// OnGetCFHeaders: sp.OnGetCFHeaders,
			// OnGetCFCheckpt: sp.OnGetCFCheckpt,
			// OnFeeFilter:    sp.OnFeeFilter,
			// OnFilterAdd:    sp.OnFilterAdd,
			// OnFilterClear:  sp.OnFilterClear,
			// OnFilterLoad:   sp.OnFilterLoad,
			// OnGetAddr:      sp.OnGetAddr,
			// OnAddr:         sp.OnAddr,
			// OnAddrV2:       sp.OnAddrV2,
			// OnRead:         sp.OnRead,
			// OnWrite:        sp.OnWrite,
			// OnNotFound:     sp.OnNotFound,

			// Note: The reference client currently bans peers that send alerts
			// not signed with its key.  We could verify against their key, but
			// since the reference client is currently unwilling to support
			// other implementations' alert messages, we will not relay theirs.
			OnAlert: nil,
		},
		NewestBlock:         nil,
		UserAgentName:       "peer",
		UserAgentVersion:    "1.0.0",
		ChainParams:         &params,
		Services:            wire.SFNodeWitness,
		ProtocolVersion:     peer.MaxProtocolVersion,
		DisableStallHandler: true,
	}
}
