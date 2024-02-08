package blockchain

import (
	"btc-indexer/internal/network"
	"btc-indexer/pkg/logger"
	"strconv"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
)

type Mode string
type Chain string

const (
	ModeFull  Mode = "full"
	ModeLight Mode = "light"
)

const (
	Mainnet Chain = "btc"
	Testnet Chain = "btct"
	Regtest Chain = "btcrt"
	Signet  Chain = "btcs"
)

type indexer struct {
	mode        Mode
	chainParams *chaincfg.Params
	logger      *logger.CustomLogger
}

func NewIndexer(mode Mode, chain Chain) *indexer {
	var chainParams *chaincfg.Params
	switch chain {
	case Mainnet:
		chainParams = &chaincfg.MainNetParams
	case Testnet:
		chainParams = &chaincfg.TestNet3Params
	case Regtest:
		chainParams = &chaincfg.RegressionNetParams
	case Signet:
		chainParams = &chaincfg.SimNetParams
	}
	return &indexer{
		mode:        mode,
		chainParams: chainParams,
		logger:      logger.NewDefaultLogger(),
	}
}

func (i *indexer) Start() {
	peerIpChan := make(chan *wire.NetAddressV2)
	defaultPeerPort, err := strconv.Atoi(i.chainParams.DefaultPort)
	if err != nil {
		i.logger.Error(err.Error())
	}
	go network.LookUpPeers(i.chainParams.DNSSeeds, uint16(defaultPeerPort), peerIpChan)

	for peer := range peerIpChan {
		i.logger.Info("Found Peer: " + peer.Addr.String())
	}
	// Get Peer Ips
	// pick best peer based on checkpoints
	// design handlers
	// request blocks
	// process and store it in db
}
