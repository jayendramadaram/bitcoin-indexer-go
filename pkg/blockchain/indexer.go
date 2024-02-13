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
	mode        Mode
	chainParams *chaincfg.Params
	logger      *logger.CustomLogger

	currentPeer    *peer.Peer
	availablePeers []string
	state          state

	headersFirstMode bool

	chain Chain
	store database.Store

	requestedBlocks int
	processedBlocks int

	stallPeerTicker *time.Ticker
}

type Chain interface {
	getBlockLocator(height int32) ([]*chainhash.Hash, error)
	findNextHeaderCheckpoint(height int32) *chaincfg.Checkpoint
}

func NewIndexer(mode Mode, chainType ChainType, headersFirst bool, store database.Store) *indexer {
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

		headersFirstMode: headersFirst,

		chain: NewChain(store, chainParams.Checkpoints),
		store: store,

		stallPeerTicker: time.NewTicker(15 * time.Second),
	}
}

type state struct {
	LastHeight int32
	LastHash   *chainhash.Hash
}

func (i *indexer) Start() {
	i.store.SetChainCfg(i.chainParams)
	LastHeight := i.GetInitialBlockHeight()
	LastHash, err := i.store.GetLatestBlockHash()
	if err != nil {
		i.logger.Error(err.Error())
	}

	txhash, err := i.store.GetLatestTxHash()
	if err != nil {
		if err == mongo.ErrNoDocuments {
			err = i.store.InitCoinBaseTx()
			if err != nil {
				i.logger.Error(err.Error())
			}
		} else {
			i.logger.Error(err.Error())
		}
	}
	fmt.Println("Latest TxHash: ", txhash)

	i.state = state{
		LastHeight: LastHeight,
		LastHash:   LastHash,
	}

	validPeers := make(chan *peer.Peer)
	i.FilterPeers(validPeers)

	// i.currentPeer = <-validPeers
	for validPeer := range validPeers {
		i.availablePeers = append(i.availablePeers, validPeer.Addr())
		// if i.currentPeer.LastBlock() < validPeer.LastBlock() {
		// 	// i.currentPeer.Disconnect()
		// 	// i.currentPeer = validPeer
		// 	continue
		// }
		validPeer.Disconnect()
	}

	var peerDoneChan = make(chan struct{})
	i.startSync(peerDoneChan)
	for range peerDoneChan {
		i.startSync(peerDoneChan)
	}
}

// gets a new peer if not set
// and starts syncing
func (i *indexer) startSync(peerDone chan struct{}) {
	i.logger.Info("Start Syncing from Peer")
	InvDoneChan := make(chan struct{})
	msgChan := make(chan interface{})
	invMsgCountChan := make(chan int)
	processDoneChan := make(chan struct{})

	peer, err := i.GetRandPeer(InvDoneChan, msgChan, invMsgCountChan)
	if err != nil {
		i.logger.Error(err.Error())
		return
	}
	i.currentPeer = peer
	i.logger.Info("Peer Connected: " + i.currentPeer.Addr())

	go i.msgHandler(msgChan, processDoneChan)

	var progresslogDoneChan = make(chan struct{})

	go func() {
		i.currentPeer.WaitForDisconnect()
		i.logger.Warn("Peer Disconnected: " + i.currentPeer.Addr())
		i.currentPeer = nil
		peerDone <- struct{}{}
		progresslogDoneChan <- struct{}{}
		return
	}()

	go func() {
		fmt.Printf("Start %s \n", time.Now())
		for {
			select {
			case <-i.stallPeerTicker.C:
				if i.currentPeer != nil {
					i.currentPeer.Disconnect()
					i.logger.Warn("Peer Stalled: " + i.currentPeer.Addr())
				}
				return
			case <-progresslogDoneChan:
				return
			case timestamp := <-time.After(60 * time.Second):
				fmt.Printf("Processed Blocks : %d [%s] \n", i.state.LastHeight, timestamp)
			}
		}
	}()

	for {
		i.processNext()

		<-InvDoneChan
		i.requestedBlocks = <-invMsgCountChan
		i.processedBlocks = 0
		<-processDoneChan

		i.stallPeerTicker.Reset(15 * time.Second)

		latestBlockHeight, err := i.store.GetLatestBlockHeight()
		if err != nil {
			i.logger.Error(err.Error())
			return
		}
		i.state.LastHeight = latestBlockHeight
		i.logger.Warn("received a done Msg")
	}

}

// returns a random Peer
func (i *indexer) GetRandPeer(invDoneChan chan struct{}, msgChan chan interface{}, invCountChan chan int) (*peer.Peer, error) {
	index := rand.Intn(len(i.availablePeers))
	i.logger.Info(fmt.Sprintf("Random Peer: %s for index %d", i.availablePeers[index], index))

	listeners := newPeerListeners(i.logger, nil, msgChan, invDoneChan, invCountChan)
	listeners.DisableSend()
	peer, err := peer.NewOutboundPeer(newPeerConfig(i.chainParams, listeners), i.availablePeers[index])
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTimeout("tcp", i.availablePeers[index], 2*time.Second)
	if err != nil {
		return nil, err
	}

	peer.AssociateConnection(conn)

	return peer, err
}

// getsLast Synced blockHeight
// if it is a fresh start, it will return insert genesis block and return 0
func (i *indexer) GetInitialBlockHeight() int32 {
	LatestBlockHeight, _ := i.store.GetLatestBlockHeight()
	// if err != nil {
	// 	if err == mongo.ErrNoDocuments {
	// 		if err := i.store.InitGenesisBlock(i.chainParams.GenesisBlock); err != nil {
	// 			i.logger.Error(err.Error())
	// 		}
	// 		LatestBlockHeight = 0
	// 	} else {
	// 		i.logger.Error(err.Error())
	// 	}
	// }
	if LatestBlockHeight == -1 {
		if err := i.store.InitGenesisBlock(i.chainParams.GenesisBlock); err != nil {
			i.logger.Error(err.Error())
		}
		LatestBlockHeight = 0
	}
	return LatestBlockHeight
}

// filters available peers and returns peers which support Segwit Upgrade
func (i *indexer) FilterPeers(validPeers chan *peer.Peer) {
	peerIpChan := make(chan *wire.NetAddressV2)
	defaultPeerPort, err := strconv.Atoi(i.chainParams.DefaultPort)
	if err != nil {
		i.logger.Error(err.Error())
	}

	go network.LookUpPeers(i.chainParams.DNSSeeds, uint16(defaultPeerPort), peerIpChan)

	wg := new(sync.WaitGroup)
	listeners := newPeerListeners(i.logger, validPeers, nil, nil, nil)
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
}

func (i *indexer) processNext() {
	locator, err := i.chain.getBlockLocator(i.state.LastHeight)
	if err != nil {
		i.logger.Error(err.Error())
	}

	// i.logger.Info("Syncing From Peer: " + i.currentPeer.Addr())

	if i.headersFirstMode {
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

	if err := i.currentPeer.PushGetBlocksMsg(locator, &chainhash.Hash{}); err != nil {
		i.logger.Error(err.Error())
	}

	i.logger.Info(fmt.Sprintf("Downloading Blocks from %d", i.state.LastHeight))

}

func (i *indexer) msgHandler(msgChan chan interface{}, processDoneChan chan struct{}) {
	for msg := range msgChan {
		switch msg := msg.(type) {
		case *wire.MsgBlock:
			if err := i.store.PutBlock(msg); err != nil {
				i.logger.Error(err.Error())
			}
			i.processedBlocks++
			if i.processedBlocks == i.requestedBlocks {
				i.logger.Info(fmt.Sprintf("Processed Blocks: %d", i.processedBlocks))
				processDoneChan <- struct{}{}
			}
		}
	}
}
