package database

import (
	"btc-indexer/pkg/logger"
	"context"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type store struct {
	blocks *mongo.Collection
	txs    *mongo.Collection
	out    *mongo.Collection
}

type Store interface {
	GetBlockByHeight(height int32) (Block, error)
	GetBlockHashByHeight(height int32) (string, error)

	GetLatestBlockHeight() (int32, error)
	GetLatestBlockHash() (*chainhash.Hash, error)

	InitGenesisBlock(block *wire.MsgBlock) error

	PutBlock(*wire.MsgBlock) error

	// PutRandBLock() error
}

func NewStore(blocks, txs, outpoints *mongo.Collection) Store {
	return &store{
		blocks: blocks,
		txs:    txs,
		out:    outpoints,
	}
}

func (s *store) GetBlockByHeight(height int32) (Block, error) {
	var block Block
	err := s.blocks.FindOne(context.TODO(), bson.D{{Key: "height", Value: height}}).Decode(&block)
	return block, err
}

func (s *store) GetBlockHashByHeight(height int32) (string, error) {
	logger.NewDefaultLogger().Info(fmt.Sprintf("GetBlockHashByHeight: %d", height))
	var BlockHash struct {
		ID string `bson:"_id"`
	}
	err := s.blocks.FindOne(context.TODO(), bson.D{{Key: "height", Value: height}}, options.FindOne().SetProjection(bson.M{"_id": 1})).Decode(&BlockHash)
	return BlockHash.ID, err
}

func (s *store) GetLatestBlockHeight() (int32, error) {
	var block struct {
		Height int32 `bson:"height"`
	}
	err := s.blocks.FindOne(context.TODO(), bson.D{}, options.FindOne().SetSort(bson.D{{Key: "height", Value: -1}}).SetProjection(bson.M{"height": 1})).Decode(&block)
	return block.Height, err
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

func (s *store) PutBlock(block *wire.MsgBlock) error {
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
	return err
}
