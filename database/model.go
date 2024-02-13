package database

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Block struct {
	ID string `bson:"_id"` //blockhash

	Height   int32 `bson:"height"` // should be indexed
	IsOrphan bool  `bson:"is_orphan"`

	PreviousBlock string `bson:"previous_block"` // indexed
	Version       int32  `bson:"version"`
	Nonce         uint32 `bson:"nonce"`
	Timestamp     int64  `bson:"timestamp"` // time stamp indexed
	Bits          uint32 `bson:"bits"`
	MerkleRoot    string `bson:"merkle_root"`
}

type Transaction struct {
	ID string `bson:"_id,omitempty"` //txhash

	LockTime uint32 `bson:"lock_time"`
	Version  int32  `bson:"version"`
	Safe     bool   `bson:"safe"`

	BlockHash  string `bson:"block_hash"`
	BlockIndex int32  `bson:"block_index"`
}

type OutPoint struct {
	ID primitive.ObjectID `bson:"_id,omitempty"`

	SpendingTxHash  string `bson:"spending_tx_hash"` // indexed
	SpendingTxIndex uint32 `bson:"spending_tx_index"`
	Sequence        uint32 `bson:"sequence"`
	SignatureScript string `bson:"signature_script"`
	Witness         string `bson:"witness"`

	FundingTxHash  string `bson:"funding_tx_hash"`  // indexed
	FundingTxIndex uint32 `bson:"funding_tx_index"` // index
	PkScript       string `bson:"pk_script"`
	Value          int64  `bson:"value"`
	Spender        string `bson:"spender"`
	Type           string `bson:"type"`
}

type Transactions []Transaction
