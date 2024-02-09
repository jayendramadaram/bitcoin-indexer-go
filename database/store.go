package database

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

type store struct {
	blocks *mongo.Collection
	txs    *mongo.Collection
	out    *mongo.Collection
}

type Store interface {
	GetBlockByHeight(height int32) (Block, error)
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
