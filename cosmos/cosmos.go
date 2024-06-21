package cosmos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/decentrio/rollup-e2e-testing/dymension"
	"github.com/decentrio/rollup-e2e-testing/ibc"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	libclient "github.com/tendermint/tendermint/rpc/jsonrpc/client"
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
) {
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
		return
	}

	// Print the output
	fmt.Println(string(output))
}

func (c *CosmosChain) Height(ctx context.Context) (uint64, error) {
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
					time.Sleep(2 * time.Second)
					continue
				} else {
					return false, fmt.Errorf("error querying rollapp state: %v", err)
				}
			}

			for _, bd := range rollappState.StateInfo.BlockDescriptors.BD {
				height, err := strconv.ParseUint(bd.Height, 10, 64)
				fmt.Println("height:", height)
				if err != nil {
					continue
				}
				if height == targetHeight {
					return true, nil
				}
			}

			if time.Since(startTime)+2*time.Second < timeout {
				time.Sleep(2 * time.Second)
			} else {
				return false, fmt.Errorf("specified rollapp height %d not found within the timeout", targetHeight)
			}
		}
	}
}
