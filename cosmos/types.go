package cosmos

import (
	types1 "github.com/tendermint/tendermint/abci/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/codec/types"
)

type TxResponse struct {
	// The block height
	Height string `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
	// The transaction hash.
	TxHash string `protobuf:"bytes,2,opt,name=txhash,proto3" json:"txhash,omitempty"`
	// Namespace for the Code
	Codespace string `protobuf:"bytes,3,opt,name=codespace,proto3" json:"codespace,omitempty"`
	// Response code.
	Code uint32 `protobuf:"varint,4,opt,name=code,proto3" json:"code,omitempty"`
	// Result bytes, if any.
	Data string `protobuf:"bytes,5,opt,name=data,proto3" json:"data,omitempty"`
	// The output of the application's logger (raw string). May be
	// non-deterministic.
	RawLog string `protobuf:"bytes,6,opt,name=raw_log,json=rawLog,proto3" json:"raw_log,omitempty"`
	// The output of the application's logger (typed). May be non-deterministic.
	Logs cosmostypes.ABCIMessageLogs `protobuf:"bytes,7,rep,name=logs,proto3,castrepeated=ABCIMessageLogs" json:"logs"`
	// Additional information. May be non-deterministic.
	Info string `protobuf:"bytes,8,opt,name=info,proto3" json:"info,omitempty"`
	// Amount of gas requested for transaction.
	GasWanted string `protobuf:"varint,9,opt,name=gas_wanted,json=gasWanted,proto3" json:"gas_wanted,omitempty"`
	// Amount of gas consumed by transaction.
	GasUsed string `protobuf:"varint,10,opt,name=gas_used,json=gasUsed,proto3" json:"gas_used,omitempty"`
	// The request transaction bytes.
	Tx *types.Any `protobuf:"bytes,11,opt,name=tx,proto3" json:"tx,omitempty"`
	// Time of the previous block. For heights > 1, it's the weighted median of
	// the timestamps of the valid votes in the block.LastCommit. For height == 1,
	// it's genesis time.
	Timestamp string `protobuf:"bytes,12,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	// Events defines all the events emitted by processing a transaction. Note,
	// these events include those emitted by processing all the messages and those
	// emitted from the ante. Whereas Logs contains the events, with
	// additional metadata, emitted only by processing the messages.
	//
	// Since: cosmos-sdk 0.42.11, 0.44.5, 0.45
	Events []types1.Event `protobuf:"bytes,13,rep,name=events,proto3" json:"events"`
}