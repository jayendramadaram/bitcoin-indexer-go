package blockchain

import (
	"btc-indexer/database"
	"btc-indexer/internal/network"
	"btc-indexer/pkg/logger"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/peer"
	"github.com/btcsuite/btcd/wire"
	"go.mongodb.org/mongo-driver/mongo"
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
	state          state

	chain Chain
	store database.Store
}

type Chain interface {
	getBlockLocator(height int32) ([]*chainhash.Hash, error)
	findNextHeaderCheckpoint(height int32) *chaincfg.Checkpoint
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

		chain: NewChain(store, chainParams.Checkpoints),
		store: store,
	}
}

type state struct {
	LastHeight int32
	LastHash   string
}

func (i *indexer) Start() {

	LatestBlockHeight, err := i.store.GetLatestBlockHeight()
	if err != nil {
		if err == mongo.ErrNoDocuments {
			LatestBlockHeight = -1
		} else {
			i.logger.Error(err.Error())
		}
	}

	i.logger.Info(fmt.Sprintf("Latest Block Height: %d", LatestBlockHeight))

	i.state = state{
		LastHeight: LatestBlockHeight,
		LastHash:   "",
	}

	peerIpChan := make(chan *wire.NetAddressV2)

	defaultPeerPort, err := strconv.Atoi(i.chainParams.DefaultPort)
	if err != nil {
		i.logger.Error(err.Error())
	}

	go network.LookUpPeers(i.chainParams.DNSSeeds, uint16(defaultPeerPort), peerIpChan)
	validPeers := make(chan *peer.Peer)

	wg := new(sync.WaitGroup)
	listeners := newPeerListeners(i.logger, validPeers)
	for peerAddr := range peerIpChan {
		go func(peerAddr *wire.NetAddressV2) {
			defer wg.Done()
			if peerAddr.ToLegacy().IP.To4() == nil {
				return
			}
			peerIp := peerAddr.Addr.String() + fmt.Sprintf(":%s", i.chainParams.DefaultPort)
			peer, err := peer.NewOutboundPeer(newPeerConfig(i.chainParams, listeners), peerIp)
			if err != nil {
				i.logger.Warn(err.Error())
				return
			}
			conn, err := net.DialTimeout("tcp", peer.Addr(), 2*time.Second)
			if err != nil {
				return
			}
			peer.AssociateConnection(conn)
		}(peerAddr)
		wg.Add(1)
	}

	go func() {
		wg.Wait()
		listeners.DisableSend()
		close(validPeers)
	}()

	if i.currentPeer != nil {
		return
	}

	for validPeer := range validPeers {
		// i.currentPeer = validPeer
		i.availablePeers = append(i.availablePeers, validPeer)
		// if i.currentPeer.LastBlock() < validPeer.LastBlock() {
		// i.currentPeer.Disconnect()
		// 	i.currentPeer = validPeer
		// 	continue
		// }
		validPeer.Disconnect()
	}

	var peerDoneChan = make(chan struct{})
	i.startSync(peerDoneChan)

	go func() {
		for {
			<-peerDoneChan
			i.startSync(peerDoneChan)
		}
	}()

	wg = &sync.WaitGroup{}
	wg.Add(1)

	// Now wait for the WaitGroup to be done, effectively blocking here until Done is called.
	wg.Wait()

	// Get Peer Ips
	// pick best peer based on checkpoints
	// design handlers
	// request blocks
	// process and store it in db
}

func (i *indexer) startSync(peerDone chan struct{}) {
	i.currentPeer = i.GetRandPeer()
	conn, err := net.DialTimeout("tcp", i.currentPeer.Addr(), 2*time.Second)
	if err != nil {
		return
	}
	i.currentPeer.AssociateConnection(conn)
	go func() {
		<-time.After(10 * time.Second)
		i.logger.Info("Peer Disconnected: " + i.currentPeer.Addr())
		peerDone <- struct{}{}
	}()

	locator, err := i.chain.getBlockLocator(i.state.LastHeight)
	if err != nil {
		i.logger.Error(err.Error())
	}

	i.logger.Info("Syning From Peer: " + i.currentPeer.Addr())

	nextCheckPoint := i.chain.findNextHeaderCheckpoint(i.state.LastHeight)
	if nextCheckPoint == nil {
		i.logger.Info("No Checkpoint Found")
		return
	}

	if err := i.currentPeer.PushGetHeadersMsg(locator, nextCheckPoint.Hash); err != nil {
		i.logger.Error(err.Error())
	}

	i.logger.Info(fmt.Sprintf("Downloading Headers from %d to %d", i.state.LastHeight, nextCheckPoint.Height))
}

func (i *indexer) GetRandPeer() *peer.Peer {
	index := rand.Intn(len(i.availablePeers))
	i.logger.Info(fmt.Sprintf("Random Peer: %s for index %d", i.availablePeers[index].Addr(), index))
	return i.availablePeers[index]
}
