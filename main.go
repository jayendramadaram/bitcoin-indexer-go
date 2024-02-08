package main

import (
	"btc-indexer/config"
	"btc-indexer/database"
	path "btc-indexer/internal"
	"btc-indexer/pkg/logger"
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

	_, err = database.NewMongoDBConnection(config.DB.URI)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	logger.Info("MongoDB Setup Complete")
	// start indexerr [go routines]
	// load server
	// run server
}
