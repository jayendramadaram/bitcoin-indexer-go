package main

import "btc-indexer/pkg/blockchain"

func main() {
	indexer := blockchain.NewIndexer(blockchain.ModeFull, blockchain.Mainnet)
	indexer.Start()
}
