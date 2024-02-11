package database

import (
	"context"

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
	err := s.blocks.FindOne(context.TODO(), Block{Height: height}).Decode(&block)
	return block, err
}

func (s *store) GetBlockHashByHeight(height int32) (string, error) {
	var BlockHash struct {
		ID string `bson:"_id"`
	}
	err := s.blocks.FindOne(context.TODO(), Block{Height: height}, options.FindOne().SetProjection(bson.M{"_id": 1})).Decode(&BlockHash)
	return BlockHash.ID, err
}

func (s *store) GetLatestBlockHeight() (int32, error) {
	var block struct {
		Height int32 `bson:"height"`
	}
	err := s.blocks.FindOne(context.TODO(), Block{Height: -1}, options.FindOne().SetProjection(bson.M{"height": 1})).Decode(&block)
	return block.Height, err
}

// func (s *store) PutRandBLock() error {
// 	_, err := s.blocks.InsertOne(context.TODO(), Block{
// 		Height:   0,
// 		IsOrphan: true,
// 	})
// 	return err
// }
