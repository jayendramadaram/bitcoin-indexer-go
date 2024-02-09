package blockchain

import (
	"btc-indexer/database"
	"btc-indexer/internal/network"
	"btc-indexer/pkg/logger"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/peer"
	"github.com/btcsuite/btcd/wire"
)

type Mode string
type ChainType string

const (
	ModeFull  Mode = "full"
	ModeLight Mode = "light"
)

const (
	Mainnet ChainType = "btc"
	Testnet ChainType = "btct"
	Regtest ChainType = "btcrt"
	Signet  ChainType = "btcs"
)

type indexer struct {
	mode           Mode
	chainParams    *chaincfg.Params
	logger         *logger.CustomLogger
	currentPeer    *peer.Peer
	availablePeers []*peer.Peer

	chain Chain
	store database.Store
}

type Chain interface {
	getBlockLocator(height int32) (BlockLocator, error)
}

func NewIndexer(mode Mode, chainType ChainType, store database.Store) *indexer {
	var chainParams *chaincfg.Params
	switch chainType {
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

		chain: NewChain(store),
		store: store,
	}
}

type state struct {
	Height         int32
	Hash           string
	requestedBlock map[chainhash.Hash]struct{}
}

func (i *indexer) Start() {

	peerIpChan := make(chan *wire.NetAddressV2)

	defaultPeerPort, err := strconv.Atoi(i.chainParams.DefaultPort)
	if err != nil {
		i.logger.Error(err.Error())
	}

	go network.LookUpPeers(i.chainParams.DNSSeeds, uint16(defaultPeerPort), peerIpChan)
	validPeers := make(chan *peer.Peer)

	wg := new(sync.WaitGroup)
	for peerAddr := range peerIpChan {
		go func(peerAddr *wire.NetAddressV2) {
			defer wg.Done()
			if peerAddr.ToLegacy().IP.To4() == nil {
				return
			}
			peerIp := peerAddr.Addr.String() + fmt.Sprintf(":%s", i.chainParams.DefaultPort)
			peer, err := peer.NewOutboundPeer(newPeerConfig(i.chainParams, newPeerListeners(i.logger, validPeers)), peerIp)
			if err != nil {
				i.logger.Warn(err.Error())
				return
			}
			conn, err := net.DialTimeout("tcp", peer.Addr(), 2*time.Second)
			if err != nil {
				i.logger.Warn(err.Error())
				return
			}
			peer.AssociateConnection(conn)
		}(peerAddr)
		wg.Add(1)
	}

	go func() {
		wg.Wait()
		close(validPeers)
	}()

	if i.currentPeer != nil {
		return
	}

	i.currentPeer = <-validPeers
	for validPeer := range validPeers {
		// i.currentPeer = validPeer
		i.availablePeers = append(i.availablePeers, validPeer)
		if i.currentPeer.LastBlock() < validPeer.LastBlock() {
			i.currentPeer.Disconnect()
			i.currentPeer = validPeer
			continue
		}
		validPeer.Disconnect()
	}

	i.logger.Info("Current Peer: " + i.currentPeer.Addr())

	// Get Peer Ips
	// pick best peer based on checkpoints
	// design handlers
	// request blocks
	// process and store it in db
}
