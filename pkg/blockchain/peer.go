package blockchain

import (
	"btc-indexer/pkg/logger"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/peer"
	"github.com/btcsuite/btcd/wire"
)

func newPeerConfig(params *chaincfg.Params, pr *peerListeners) *peer.Config {
	return &peer.Config{
		Listeners: peer.MessageListeners{
			OnVersion: pr.OnVersion,
			OnHeaders: pr.OnHeaders,
			OnBlock:   pr.OnBlock,
			OnInv:     pr.OnInv,
			// OnVerAck:  pr.OnVerAck,
			// OnMemPool:      sp.OnMemPool,
			// OnTx:           sp.OnTx,
			// OnHeaders:      sp.OnHeaders,
			// OnGetData:      sp.OnGetData,
			// OnGetBlocks:    sp.OnGetBlocks,
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
		ChainParams:         params,
		Services:            wire.SFNodeWitness,
		ProtocolVersion:     peer.MaxProtocolVersion,
		DisableStallHandler: false,
		AllowSelfConns:      true,
	}
}

type peerListeners struct {
	logger     *logger.CustomLogger
	validPeers chan *peer.Peer
	CanSend    bool

	msgChan    chan interface{}
	InvMsgChan chan int

	done chan struct{}
}

func newPeerListeners(logger *logger.CustomLogger, validPeers chan *peer.Peer, msgChan chan interface{}, done chan struct{}, InvMsgChan chan int) *peerListeners {
	return &peerListeners{
		logger:     logger,
		validPeers: validPeers,
		CanSend:    true,
		msgChan:    msgChan,
		done:       done,
		InvMsgChan: InvMsgChan,
	}
}

func (pr *peerListeners) DisableSend() {
	pr.CanSend = false
}

func (pr *peerListeners) OnVersion(p *peer.Peer, msg *wire.MsgVersion) *wire.MsgReject {
	if p.Services()&wire.SFNodeWitness == wire.SFNodeWitness {
		if pr.CanSend {
			pr.validPeers <- p
			return nil
		}
	}
	return nil
}

func (pr *peerListeners) OnHeaders(p *peer.Peer, msg *wire.MsgHeaders) {
	pr.logger.Debug(fmt.Sprintf("Headers: %d", len(msg.Headers)))
	for _, hdr := range msg.Headers {
		pr.logger.Debug(fmt.Sprintf("Header: %s", hdr.Timestamp))
	}
}

func (pr *peerListeners) OnInv(p *peer.Peer, msg *wire.MsgInv) {
	if msg.InvList[0].Type != wire.InvTypeBlock {
		return
	}
	pr.logger.Debug(fmt.Sprintf("Inv: %d of type %d", len(msg.InvList), msg.InvList[0].Type))
	sendMsg := wire.NewMsgGetData()
	for _, inv := range msg.InvList {
		sendMsg.AddInvVect(inv)
	}
	p.QueueMessage(sendMsg, pr.done)
	if msg.InvList[0].Type == wire.InvTypeBlock {
		pr.InvMsgChan <- len(msg.InvList)
	}
}

func (pr *peerListeners) OnBlock(p *peer.Peer, msg *wire.MsgBlock, buf []byte) {
	pr.msgChan <- msg
}
