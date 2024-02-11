package main

import (
	"btc-indexer/config"
	"btc-indexer/database"
	path "btc-indexer/internal"
	"btc-indexer/pkg/blockchain"
	"btc-indexer/pkg/logger"
	"context"
)

func main() {

	// load config
	config, err := config.LoadConfig(path.DefaultConfigPath)
	if err != nil {
		panic(err)
	}

	logger := logger.NewLoggerWithOptions(config.Logger.Level, &logger.Options{
		LogBackTraceEnabled: config.Logger.LogBackTraceEnabled,
	})

	logger.Info("Logger Setup Complete")

	mi, err := database.NewMongoDBConnection(config.DB.URI)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	defer func() {
		mi.Client.Disconnect(context.TODO())
	}()

	mi, err = mi.SetupIndexerClient(context.TODO(), config.DB.Database)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	store := database.NewStore(
		mi.BlocksCol,
		mi.TxCol,
		mi.OutCol,
	)

	logger.Info("MongoDB Setup Complete")

	indexer := blockchain.NewIndexer(blockchain.ModeFull, blockchain.Mainnet, store)
	indexer.Start()
	// start indexerr [go routines]
	// load server
	// run server
}
