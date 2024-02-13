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
		Keys:    bson.D{{Key: "height", Value: 1}},
		Options: options.Index().SetUnique(false),
	}

	prevBlockIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "previous_block", Value: 1}},
		Options: options.Index().SetUnique(false),
	}

	_, err := blocksCol.Indexes().CreateMany(ctx, []mongo.IndexModel{heightIndex, prevBlockIndex})
	if err != nil {
		return mi, err
	}

	spendingTxhashIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "spending_tx_hash", Value: 1}},
		Options: options.Index().SetUnique(false),
	}

	fundingTxIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "funding_tx_hash", Value: 1}, {Key: "funding_tx_index", Value: 1}},
		Options: options.Index().SetUnique(false),
	}

	_, err = OutPointCol.Indexes().CreateOne(ctx, spendingTxhashIndex)
	if err != nil {
		return mi, err
	}

	_, err = OutPointCol.Indexes().CreateOne(ctx, fundingTxIndex)
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
