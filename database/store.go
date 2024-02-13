package database

import (
	"btc-indexer/pkg/logger"
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type store struct {
	blocks *mongo.Collection
	txs    *mongo.Collection
	out    *mongo.Collection

	latestHeight int32
	chainParams  *chaincfg.Params

	mu     sync.Mutex
	logger *logger.CustomLogger
}

type Store interface {
	GetBlockByHeight(height int32) (Block, error)
	GetBlockByHash(hash string) (Block, error)
	GetBlockHashByHeight(height int32) (string, error)

	GetLatestBlockHeight() (int32, error)
	GetLatestBlockHash() (*chainhash.Hash, error)

	GetLatestTxHash() (*chainhash.Hash, error)

	PutBlock(*wire.MsgBlock) error
	PutTx(*wire.MsgTx, string, int32) error

	InitGenesisBlock(block *wire.MsgBlock) error
	InitCoinBaseTx() error

	SetChainCfg(chainParams *chaincfg.Params)

	// PutRandBLock() error
}

func NewStore(blocks, txs, outpoints *mongo.Collection) (Store, error) {
	var block struct {
		Height int32 `bson:"height"`
	}

	err := blocks.FindOne(context.TODO(), bson.D{}, options.FindOne().SetSort(bson.D{{Key: "height", Value: -1}}).SetProjection(bson.M{"height": 1})).Decode(&block)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			block.Height = -1
		} else {
			return nil, err
		}
	}

	return &store{
		blocks:       blocks,
		txs:          txs,
		out:          outpoints,
		latestHeight: block.Height,
		logger:       logger.NewDefaultLogger(),
		mu:           sync.Mutex{},
	}, nil
}

func (s *store) SetChainCfg(chainParams *chaincfg.Params) {
	s.chainParams = chainParams
}

func (s *store) GetBlockByHeight(height int32) (Block, error) {
	var block Block
	err := s.blocks.FindOne(context.TODO(), bson.D{{Key: "height", Value: height}}).Decode(&block)
	return block, err
}

func (s *store) GetBlockByHash(hash string) (Block, error) {
	var block Block
	err := s.blocks.FindOne(context.TODO(), bson.D{{Key: "_id", Value: hash}}).Decode(&block)
	return block, err
}

func (s *store) GetBlockHashByHeight(height int32) (string, error) {
	// s.logger.Info(fmt.Sprintf("GetBlockHashByHeight: %d", height))
	var BlockHash struct {
		ID string `bson:"_id"`
	}
	err := s.blocks.FindOne(context.TODO(), bson.D{{Key: "height", Value: height}}, options.FindOne().SetProjection(bson.M{"_id": 1})).Decode(&BlockHash)
	return BlockHash.ID, err
}

func (s *store) GetLatestBlockHeight() (int32, error) {
	return s.latestHeight, nil
}

func (s *store) GetLatestBlockHash() (*chainhash.Hash, error) {
	var block struct {
		Hash string `bson:"_id"`
	}
	err := s.blocks.FindOne(context.TODO(), bson.D{}, options.FindOne().SetSort(bson.D{{Key: "height", Value: -1}}).SetProjection(bson.M{"_id": 1})).Decode(&block)
	if err != nil {
		return nil, err
	}
	return chainhash.NewHashFromStr(block.Hash)
}

func (s *store) GetLatestTxHash() (*chainhash.Hash, error) {
	var tx struct {
		Hash string `bson:"_id"`
	}
	err := s.txs.FindOne(context.TODO(), bson.D{}, options.FindOne().SetSort(bson.D{{Key: "height", Value: -1}}).SetProjection(bson.M{"_id": 1})).Decode(&tx)
	if err != nil {
		return nil, err
	}
	return chainhash.NewHashFromStr(tx.Hash)
}

func (s *store) PutBlock(block *wire.MsgBlock) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// if incoming block num is less than latest consider it has orphan
	// if incoming block is same as latest then get previous block
	// recursively set all its parents to not orphan and corresponding non orphan blocks to orphan
	// else if parent is not orphan consider as bestChain and continue indexing blocks
	// finally update latestBlock Height in store
	prevBlock, err := s.GetBlockByHash(block.Header.PrevBlock.String())
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil
		}
		s.logger.Error(err.Error())
		return err
	}

	// latest best chain is longer than incoming block then incoming block is orphan
	if s.latestHeight-1 > prevBlock.Height {
		bl := Block{
			ID:            block.BlockHash().String(),
			Height:        prevBlock.Height + 1,
			IsOrphan:      true,
			PreviousBlock: block.Header.PrevBlock.String(),
			Version:       block.Header.Version,
			Nonce:         block.Header.Nonce,
			Timestamp:     block.Header.Timestamp.Unix(),
			Bits:          block.Header.Bits,
			MerkleRoot:    block.Header.MerkleRoot.String(),
		}
		_, err := s.blocks.InsertOne(context.TODO(), bl)
		if err != nil {
			if mongo.IsDuplicateKeyError(err) {
				s.logger.Warn(fmt.Sprintf("Block %s already exists", block.BlockHash().String()))
				return nil
			}
			s.logger.Error(err.Error())
			return err
		}
		s.processTxs(block.Transactions, block.BlockHash().String(), prevBlock.Height+1)
		return nil
	}

	// redefine bestChain
	if s.latestHeight == prevBlock.Height || (s.latestHeight-1) == prevBlock.Height {
		parentBlock := prevBlock
		for parentBlock.IsOrphan {
			// update corresponding non orphan block to orphan
			_, err := s.blocks.UpdateOne(context.TODO(), bson.D{{Key: "height", Value: parentBlock.Height}, {Key: "is_orphan", Value: false}}, bson.D{{Key: "$set", Value: bson.D{{Key: "is_orphan", Value: true}}}})
			if err != nil {
				s.logger.Error(err.Error())
				return err
			}

			// make sure parent is not orphan
			_, err = s.blocks.UpdateOne(context.TODO(), bson.D{{Key: "_id", Value: parentBlock.ID}}, bson.D{{Key: "$set", Value: bson.D{{Key: "is_orphan", Value: false}}}})
			if err != nil {
				s.logger.Error(err.Error())
				return err
			}

			parentBlock, err = s.GetBlockByHash(parentBlock.PreviousBlock)
			if err != nil {
				s.logger.Error(err.Error())
				return err
			}
		}
	}

	bl := Block{
		ID:            block.BlockHash().String(),
		Height:        prevBlock.Height + 1,
		IsOrphan:      false,
		PreviousBlock: block.Header.PrevBlock.String(),
		Version:       block.Header.Version,
		Nonce:         block.Header.Nonce,
		Timestamp:     block.Header.Timestamp.Unix(),
		Bits:          block.Header.Bits,
		MerkleRoot:    block.Header.MerkleRoot.String(),
	}
	_, err = s.blocks.InsertOne(context.TODO(), bl)
	if err != nil {
		s.logger.Error(err.Error())
		return err
	}

	s.processTxs(block.Transactions, block.BlockHash().String(), prevBlock.Height+1)

	s.latestHeight = prevBlock.Height + 1
	s.logger.Info(fmt.Sprintf("Height: %d", s.latestHeight))
	return nil
}

func (s *store) processTxs(txs []*wire.MsgTx, blockhash string, blockIndex int32) {
	wg := new(sync.WaitGroup)
	for _, tx := range txs {
		go func(tx *wire.MsgTx) {
			defer wg.Done()
			err := s.PutTx(tx, blockhash, blockIndex)
			if err != nil {
				s.logger.Error(err.Error())
			}
		}(tx)
		wg.Add(1)
	}
	wg.Wait()
}

func (s *store) PutTx(tx *wire.MsgTx, blockhash string, blockIndex int32) error {
	transaction := Transaction{
		ID:         tx.TxHash().String(),
		LockTime:   tx.LockTime,
		Version:    tx.Version,
		Safe:       true,
		BlockHash:  blockhash,
		BlockIndex: blockIndex,
	}
	_, err := s.txs.InsertOne(context.TODO(), transaction)
	if err != nil {
		return err
	}

	// process inputs and outputs
	// for all outputs create them as new fundingTxs
	// for all inputs link it to its previous outPoint i e; fundingTx and inputs are spendingTx

	for i, out := range tx.TxOut {
		spenderAddress := ""

		pkScript, err := txscript.ParsePkScript(out.PkScript)
		if err == nil {
			addr, err := pkScript.Address(s.chainParams)
			if err != nil {
				return err
			}
			spenderAddress = addr.EncodeAddress()
		}

		outPoint := OutPoint{
			FundingTxHash:  tx.TxHash().String(),
			FundingTxIndex: uint32(i),
			PkScript:       hex.EncodeToString(out.PkScript),
			Value:          out.Value,
			Spender:        spenderAddress,
			Type:           pkScript.Class().String(),
		}
		_, err = s.out.InsertOne(context.TODO(), outPoint)
		if err != nil {
			return err
		}
	}

	for i, txIn := range tx.TxIn {
		witness := make([]string, len(txIn.Witness))
		for i, w := range txIn.Witness {
			witness[i] = hex.EncodeToString(w)
		}
		witnessToHex := strings.Join(witness, ",")

		// get previous txOut
		var outPoint OutPoint
		err = s.out.FindOne(context.TODO(), bson.D{{Key: "funding_tx_hash", Value: txIn.PreviousOutPoint.Hash.String()}, {Key: "funding_tx_index", Value: txIn.PreviousOutPoint.Index}}).Decode(&outPoint)
		if err != nil {
			s.logger.Error(fmt.Sprintf("Error: %s fundingTx %v index %d", err.Error(), txIn.PreviousOutPoint.Hash.String(), txIn.PreviousOutPoint.Index))
			return err
		}

		outPoint.SpendingTxHash = tx.TxHash().String()
		outPoint.SpendingTxIndex = uint32(i)
		outPoint.Sequence = txIn.Sequence
		outPoint.SignatureScript = hex.EncodeToString(txIn.SignatureScript)
		outPoint.Witness = witnessToHex
		_, err = s.out.UpdateOne(context.TODO(), bson.D{{Key: "_id", Value: outPoint.ID}}, bson.D{{Key: "$set", Value: outPoint}})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *store) InitGenesisBlock(block *wire.MsgBlock) error {
	bl := Block{
		ID:            block.BlockHash().String(),
		Height:        0,
		IsOrphan:      false,
		PreviousBlock: block.Header.PrevBlock.String(),
		Version:       block.Header.Version,
		Nonce:         block.Header.Nonce,
		Timestamp:     block.Header.Timestamp.Unix(),
		Bits:          block.Header.Bits,
		MerkleRoot:    block.Header.MerkleRoot.String(),
	}
	_, err := s.blocks.InsertOne(context.TODO(), bl)
	s.latestHeight = 0
	return err
}

func (s *store) InitCoinBaseTx() error {
	tx := OutPoint{
		FundingTxHash:  "0000000000000000000000000000000000000000000000000000000000000000",
		FundingTxIndex: 4294967295,
	}
	_, err := s.out.InsertOne(context.TODO(), tx)
	return err
}
