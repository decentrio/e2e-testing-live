package cosmos

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/types"
	chanTypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	"github.com/decentrio/rollup-e2e-testing/blockdb"
	"github.com/decentrio/rollup-e2e-testing/dymension"
	"github.com/decentrio/rollup-e2e-testing/ibc"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	libclient "github.com/tendermint/tendermint/rpc/jsonrpc/client"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type CosmosChain struct {
	RPCAddr       string `json:"rpc_addr"`
	JsonRPCAddr   string `json:"json_rpc_addr"`
	GrpcAddr      string `json:"grpc_addr"`
	ChainID       string `json:"chain_id"`
	Bin           string `json:"bin"`
	GasPrices     string `json:"gas_prices"`
	GasAdjustment string `json:"gas_adjustment"`
	Denom         string `json:"denom"`
	Client        rpcclient.Client
}

// NewClient creates and assigns a new Tendermint RPC client to the Node
func (c *CosmosChain) NewClient(addr string) error {
	httpClient, err := libclient.DefaultHTTPClient(addr)
	if err != nil {
		return err
	}

	httpClient.Timeout = 10 * time.Second
	rpcClient, err := rpchttp.NewWithClient(addr, "/websocket", httpClient)
	if err != nil {
		return err
	}

	c.Client = rpcClient
	return nil
}

func SendIBCTransfer(
	srcChain CosmosChain,
	channelID string,
	keyName string,
	toWallet ibc.WalletData,
	fees string,
	options ibc.TransferOptions,
) (*TxResponse, error) {
	command := []string{
		"ibc-transfer", "transfer", "transfer", channelID,
		toWallet.Address, fmt.Sprintf("%s%s", toWallet.Amount.String(), toWallet.Denom),
		"--fees", fees, "--node", "https://" + srcChain.RPCAddr,
	}
	if options.Timeout != nil {
		if options.Timeout.NanoSeconds > 0 {
			command = append(command, "--packet-timeout-timestamp", fmt.Sprint(options.Timeout.NanoSeconds))
		} else if options.Timeout.Height > 0 {
			command = append(command, "--packet-timeout-height", fmt.Sprintf("0-%d", options.Timeout.Height))
		}
	}
	if options.Memo != "" {
		command = append(command, "--memo", options.Memo)
	}

	command = append([]string{"tx"}, command...)
	// var gasPriceFound, gasAdjustmentFound = false, false
	// for i := 0; i < len(command); i++ {
	// 	if command[i] == "--gas-prices" {
	// 		gasPriceFound = true
	// 	}
	// 	if command[i] == "--gas-adjustment" {
	// 		gasAdjustmentFound = true
	// 	}
	// }

	// if !gasPriceFound {
	// 	command = append(command, "--gas-prices", srcChain.GasPrices)
	// }
	// if !gasAdjustmentFound {
	// 	command = append(command, "--gas-adjustment", srcChain.GasAdjustment)
	// }

	command = append(command,
		"--chain-id", srcChain.ChainID,
		"--gas", "auto",
		"--gas-adjustment", "1.5",
		"--from", keyName,
		"--keyring-backend", keyring.BackendTest,
		"--output", "json",
		"--broadcast-mode", "block",
		"-y")

	// Create the command
	cmd := exec.Command(srcChain.Bin, command...)
	fmt.Println(cmd)
	// Run the command and get the output
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error executing command:", err)
		return nil, err
	}

	txResponse := TxResponse{}
	err = json.Unmarshal(output, &txResponse)
	if err != nil {
		return nil, err
	}

	return &txResponse, nil
}

// TODO: refactor this to dym_hub
func FullfillDemandOrder(
	dymHub *CosmosChain,
	orderId string,
	keyName string,
	fees string,
) (*TxResponse, error) {
	command := []string{
		"eibc", "fulfill-order", orderId,
		"--fees", fees, "--node", "https://" + dymHub.RPCAddr,
	}

	command = append([]string{"tx"}, command...)

	command = append(command,
		"--chain-id", dymHub.ChainID,
		"--gas", "auto",
		"--gas-adjustment", "1.5",
		"--from", keyName,
		"--keyring-backend", keyring.BackendTest,
		"--output", "json",
		"--broadcast-mode", "block",
		"-y")

	// Create the command
	cmd := exec.Command(dymHub.Bin, command...)
	fmt.Println(cmd)
	// Run the command and get the output
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error executing command:", err)
		return nil, err
	}

	txResponse := TxResponse{}
	err = json.Unmarshal(output, &txResponse)
	if err != nil {
		return nil, err
	}

	return &txResponse, nil
}

func GetIbcTxFromTxResponse(txResp TxResponse) (tx ibc.Tx, _ error) {
	height, err := strconv.ParseUint(txResp.Height, 10, 64)
	if err != nil {
		return tx, err
	}
	tx.Height = height
	tx.TxHash = txResp.TxHash
	// In cosmos, user is charged for entire gas requested, not the actual gas used.
	tx.GasSpent = txResp.GasWanted

	const evType = "send_packet"
	events := txResp.Events

	var (
		seq, _           = AttributeValue(events, evType, "packet_sequence")
		srcPort, _       = AttributeValue(events, evType, "packet_src_port")
		srcChan, _       = AttributeValue(events, evType, "packet_src_channel")
		dstPort, _       = AttributeValue(events, evType, "packet_dst_port")
		dstChan, _       = AttributeValue(events, evType, "packet_dst_channel")
		timeoutHeight, _ = AttributeValue(events, evType, "packet_timeout_height")
		timeoutTs, _     = AttributeValue(events, evType, "packet_timeout_timestamp")
		data, _          = AttributeValue(events, evType, "packet_data")
	)
	tx.Packet.SourcePort = srcPort
	tx.Packet.SourceChannel = srcChan
	tx.Packet.DestPort = dstPort
	tx.Packet.DestChannel = dstChan
	tx.Packet.TimeoutHeight = timeoutHeight
	tx.Packet.Data = []byte(data)

	seqNum, err := strconv.Atoi(seq)
	if err != nil {
		return tx, fmt.Errorf("invalid packet sequence from events %s: %w", seq, err)
	}
	tx.Packet.Sequence = uint64(seqNum)

	timeoutNano, err := strconv.ParseUint(timeoutTs, 10, 64)
	if err != nil {
		return tx, fmt.Errorf("invalid packet timestamp timeout %s: %w", timeoutTs, err)
	}
	tx.Packet.TimeoutTimestamp = ibc.Nanoseconds(timeoutNano)

	return tx, nil
}

func (c CosmosChain) Height(ctx context.Context) (uint64, error) {
	res, err := c.Client.Status(ctx)
	if err != nil {
		return 0, fmt.Errorf("tendermint rpc client status: %w", err)
	}
	height := res.SyncInfo.LatestBlockHeight
	return uint64(height), nil
}

func (c *CosmosChain) CreateUser(keyName string) (User, error) {
	user := User{}
	if err := c.CreateKey(keyName); err != nil {
		return user, err
	}
	addr, err := c.KeyBech32(keyName)
	if err != nil {
		return user, err
	}

	user = User{
		Address: addr,
		Denom:   c.Denom,
	}
	return user, nil
}

func (c *CosmosChain) CreateKey(name string) error {

	command := []string{
		"keys", "add", name, "--keyring-backend", keyring.BackendTest,
	}

	// Create the command
	cmd := exec.Command(c.Bin, command...)
	fmt.Println(cmd)
	// Run the command and get the output
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error executing command:", err)
		return err
	}

	// Print the output
	fmt.Println(string(output))
	return err
}

func (c *CosmosChain) KeyBech32(name string) (string, error) {
	command := []string{"keys", "show", "--address", name,
		"--keyring-backend", keyring.BackendTest,
	}

	// Create the command
	cmd := exec.Command(c.Bin, command...)
	fmt.Println(cmd)
	// Run the command and get the output
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error executing command:", err)
		return "", err
	}

	// Print the output
	fmt.Println(string(output))

	return string(bytes.TrimSuffix(output, []byte("\n"))), nil
}

// Acknowledgements implements ibc.Chain, returning all acknowledgments in block at height
func (c CosmosChain) Acknowledgements(ctx context.Context, interfaceRegistry codectypes.InterfaceRegistry, height uint64) ([]ibc.PacketAcknowledgement, error) {
	var acks []*chanTypes.MsgAcknowledgement

	err := rangeBlockMessages(ctx, interfaceRegistry, c.Client, height, func(msg types.Msg) bool {
		found, ok := msg.(*chanTypes.MsgAcknowledgement)
		if ok {
			acks = append(acks, found)
		}
		return false
	})
	if err != nil {
		return nil, fmt.Errorf("find acknowledgements at height %d: %w", height, err)
	}
	ibcAcks := make([]ibc.PacketAcknowledgement, len(acks))
	for i, ack := range acks {
		ack := ack
		ibcAcks[i] = ibc.PacketAcknowledgement{
			Acknowledgement: ack.Acknowledgement,
			Packet: ibc.Packet{
				Sequence:         ack.Packet.Sequence,
				SourcePort:       ack.Packet.SourcePort,
				SourceChannel:    ack.Packet.SourceChannel,
				DestPort:         ack.Packet.DestinationPort,
				DestChannel:      ack.Packet.DestinationChannel,
				Data:             ack.Packet.Data,
				TimeoutHeight:    ack.Packet.TimeoutHeight.String(),
				TimeoutTimestamp: ibc.Nanoseconds(ack.Packet.TimeoutTimestamp),
			},
		}
	}
	return ibcAcks, nil
}

func (c CosmosChain) FindTxs(ctx context.Context, height uint64, interfaceRegistry codectypes.InterfaceRegistry) ([]blockdb.Tx, error) {
	h := int64(height)
	var eg errgroup.Group
	var blockRes *coretypes.ResultBlockResults
	var block *coretypes.ResultBlock
	eg.Go(func() (err error) {
		blockRes, err = c.Client.BlockResults(ctx, &h)
		return err
	})
	eg.Go(func() (err error) {
		block, err = c.Client.Block(ctx, &h)
		return err
	})
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	txs := make([]blockdb.Tx, 0, len(block.Block.Txs)+2)
	for i, tx := range block.Block.Txs {
		var newTx blockdb.Tx
		newTx.Data = []byte(fmt.Sprintf(`{"data":"%s"}`, hex.EncodeToString(tx)))

		sdkTx, err := decodeTX(interfaceRegistry, tx)
		if err != nil {
			fmt.Println("Failed to decode tx", zap.Uint64("height", height), zap.Error(err))
			continue
		}
		b, err := encodeTxToJSON(interfaceRegistry, sdkTx)
		if err != nil {
			fmt.Println("Failed to marshal tx to json", zap.Uint64("height", height), zap.Error(err))
			continue
		}
		newTx.Data = b

		rTx := blockRes.TxsResults[i]

		newTx.Events = make([]blockdb.Event, len(rTx.Events))
		for j, e := range rTx.Events {
			attrs := make([]blockdb.EventAttribute, len(e.Attributes))
			for k, attr := range e.Attributes {
				attrs[k] = blockdb.EventAttribute{
					Key:   string(attr.Key),
					Value: string(attr.Value),
				}
			}
			newTx.Events[j] = blockdb.Event{
				Type:       e.Type,
				Attributes: attrs,
			}
		}
		txs = append(txs, newTx)
	}
	if len(blockRes.BeginBlockEvents) > 0 {
		beginBlockTx := blockdb.Tx{
			Data: []byte(`{"data":"begin_block","note":"this is a transaction artificially created for debugging purposes"}`),
		}
		beginBlockTx.Events = make([]blockdb.Event, len(blockRes.BeginBlockEvents))
		for i, e := range blockRes.BeginBlockEvents {
			attrs := make([]blockdb.EventAttribute, len(e.Attributes))
			for j, attr := range e.Attributes {
				attrs[j] = blockdb.EventAttribute{
					Key:   string(attr.Key),
					Value: string(attr.Value),
				}
			}
			beginBlockTx.Events[i] = blockdb.Event{
				Type:       e.Type,
				Attributes: attrs,
			}
		}
		txs = append(txs, beginBlockTx)
	}
	if len(blockRes.EndBlockEvents) > 0 {
		endBlockTx := blockdb.Tx{
			Data: []byte(`{"data":"end_block","note":"this is a transaction artificially created for debugging purposes"}`),
		}
		endBlockTx.Events = make([]blockdb.Event, len(blockRes.EndBlockEvents))
		for i, e := range blockRes.EndBlockEvents {
			attrs := make([]blockdb.EventAttribute, len(e.Attributes))
			for j, attr := range e.Attributes {
				attrs[j] = blockdb.EventAttribute{
					Key:   string(attr.Key),
					Value: string(attr.Value),
				}
			}
			endBlockTx.Events[i] = blockdb.Event{
				Type:       e.Type,
				Attributes: attrs,
			}
		}
		txs = append(txs, endBlockTx)
	}

	return txs, nil
}

func (c *CosmosChain) QueryRollappState(rollappName string, onlyFinalized bool) (*dymension.RollappState, error) {

	command := []string{
		"q", "rollapp", "state", rollappName, "--node", "https://" + c.RPCAddr, "--output", "json"}

	if onlyFinalized {
		command = append(command, "--finalized")
	}

	// Create the command
	cmd := exec.Command(c.Bin, command...)
	fmt.Println(cmd)
	// Run the command and get the output
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error executing command:", err)
		return nil, err
	}

	// Print the output
	fmt.Println(string(output))
	var rollappState dymension.RollappState
	err = json.Unmarshal(output, &rollappState)
	if err != nil {
		return nil, err
	}
	return &rollappState, nil
}

func (c *CosmosChain) FinalizedRollappStateHeight(rollappName string) (uint64, error) {
	rollappState, err := c.QueryRollappState(rollappName, true)
	if err != nil {
		return 0, err
	}

	if len(rollappState.StateInfo.BlockDescriptors.BD) == 0 {
		return 0, fmt.Errorf("no block descriptors found for rollapp %s", rollappName)
	}

	lastBD := rollappState.StateInfo.BlockDescriptors.BD[len(rollappState.StateInfo.BlockDescriptors.BD)-1]
	parsedHeight, err := strconv.ParseUint(lastBD.Height, 10, 64)
	if err != nil {
		return 0, err
	}
	return parsedHeight, nil
}

func (c *CosmosChain) FinalizedRollappDymHeight(rollappName string) (uint64, error) {
	rollappState, err := c.QueryRollappState(rollappName, true)
	if err != nil {
		return 0, err
	}

	parsedHeight, err := strconv.ParseUint(rollappState.StateInfo.CreationHeight, 10, 64)
	if err != nil {
		return 0, err
	}
	return parsedHeight, nil
}

func (c *CosmosChain) WaitUntilRollappHeightIsFinalized(ctx context.Context, rollappChainID string, targetHeight uint64, timeoutSecs int) (bool, error) {
	startTime := time.Now()
	timeout := time.Duration(timeoutSecs) * time.Second

	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(timeout):
			return false, fmt.Errorf("specified rollapp height %d not found within the timeout", targetHeight)
		default:
			rollappState, err := c.QueryRollappState(rollappChainID, true)
			if err != nil {
				if time.Since(startTime) < timeout {
					time.Sleep(10 * time.Second)
					continue
				} else {
					return false, fmt.Errorf("error querying rollapp state: %v", err)
				}
			}

			for _, bd := range rollappState.StateInfo.BlockDescriptors.BD {
				height, err := strconv.ParseUint(bd.Height, 10, 64)
				if err != nil {
					continue
				}
				if height == targetHeight {
					return true, nil
				}
			}

			if time.Since(startTime)+2*time.Second < timeout {
				time.Sleep(10 * time.Second)
			} else {
				return false, fmt.Errorf("specified rollapp height %d not found within the timeout", targetHeight)
			}
		}
	}
}
