package database

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoInstance struct {
	Client    *mongo.Client
	BlocksCol *mongo.Collection
	TxCol     *mongo.Collection
	OutCol    *mongo.Collection
}

func NewMongoDBConnection(dbUri string) (*mongoInstance, error) {
	ctx := context.Background()
	clientOptions := options.Client().ApplyURI(dbUri).SetMaxPoolSize(20)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &mongoInstance{Client: client}, nil
}

func (mi *mongoInstance) SetupIndexerClient(ctx context.Context, dbName string) (*mongoInstance, error) {
	db := mi.Client.Database(dbName)

	blocksCol := db.Collection("Blocks")
	TxCol := db.Collection("Transactions")
	OutPointCol := db.Collection("OutPoints")

	heightIndex := mongo.IndexModel{
		Keys:    bson.D{{"height", 1}},
		Options: options.Index().SetUnique(true),
	}

	prevBlockIndex := mongo.IndexModel{
		Keys:    bson.D{{"previous_block", 1}},
		Options: options.Index().SetUnique(false),
	}

	_, err := blocksCol.Indexes().CreateMany(ctx, []mongo.IndexModel{heightIndex, prevBlockIndex})
	if err != nil {
		return mi, err
	}

	spendingTxIndex := mongo.IndexModel{
		Keys:    bson.D{{"spending_tx_id", 1}},
		Options: options.Index().SetUnique(false),
	}

	spendingTxhashIndex := mongo.IndexModel{
		Keys:    bson.D{{"spending_tx_hash", 1}},
		Options: options.Index().SetUnique(false),
	}

	fundigTxIndex := mongo.IndexModel{
		Keys:    bson.D{{"funding_tx_id", 1}},
		Options: options.Index().SetUnique(false),
	}

	fundingTxhashIndex := mongo.IndexModel{
		Keys:    bson.D{{"funding_tx_hash", 1}},
		Options: options.Index().SetUnique(false),
	}

	_, err = TxCol.Indexes().CreateMany(ctx, []mongo.IndexModel{spendingTxIndex, spendingTxhashIndex, fundigTxIndex, fundingTxhashIndex})
	if err != nil {
		return mi, err
	}

	return &mongoInstance{
		Client:    mi.Client,
		BlocksCol: blocksCol,
		TxCol:     TxCol,
		OutCol:    OutPointCol,
	}, nil
}
