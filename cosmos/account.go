package cosmos

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"regexp"
	"strings"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type User struct {
	Address string `json:"address"`
	Denom   string `json:"denom"`
}

// GetBalance fetches the current balance for a specific account address and denom.
func (user *User) GetBalance(ctx context.Context, denom, grpcAddr string) (sdkmath.Int, error) {
	params := &bankTypes.QueryBalanceRequest{Address: user.Address, Denom: denom}
	conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return sdkmath.Int{}, err
	}
	defer conn.Close()

	queryClient := bankTypes.NewQueryClient(conn)
	res, err := queryClient.Balance(ctx, params)

	if err != nil {
		return sdkmath.Int{}, err
	}

	return res.Balance.Amount, nil
}

func (user *User) GetFaucet(api string) {
	// Data to send in the POST request
	data := map[string]string{
		"address": user.Address,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	// Create a new POST request
	req, err := http.NewRequest("POST", api, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	// Set the request header to indicate that we're sending JSON data
	req.Header.Set("Content-Type", "application/json")

	// Create an HTTP client and send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	fmt.Println("Response Status:", resp.Status)
	fmt.Println("Response Body:", string(body))
}

func GetEvmAddressFromAnyFormatAddress(addrs ...string) (evmAddrs []common.Address, err error) {
	for _, addr := range addrs {
		normalizedAddr := strings.ToLower(addr)

		if regexp.MustCompile(`^(0x)?[a-f\d]{40}$`).MatchString(normalizedAddr) {
			evmAddrs = append(evmAddrs, common.HexToAddress(normalizedAddr))
		} else if regexp.MustCompile(`^(0x)?[a-f\d]{64}$`).MatchString(normalizedAddr) {
			err = fmt.Errorf("ERR: invalid address format: %s", normalizedAddr)
			return
		} else { // bech32
			spl := strings.Split(normalizedAddr, "1")
			if len(spl) != 2 || len(spl[0]) < 1 || len(spl[1]) < 1 {
				err = fmt.Errorf("ERR: invalid bech32 address: %s", normalizedAddr)
				return
			}

			var bz []byte
			bz, err = sdk.GetFromBech32(normalizedAddr, spl[0])
			if err != nil {
				err = fmt.Errorf("ERR: failed to decode bech32 address %s: %s", normalizedAddr, err)
				return
			}

			if len(bz) != 20 {
				err = fmt.Errorf("ERR: bech32 address %s has invalid length, must be 20 bytes, got %s %d bytes", normalizedAddr, hex.EncodeToString(bz), len(bz))
				return
			}

			evmAddrs = append(evmAddrs, common.BytesToAddress(bz))
		}
	}

	return
}

func (user *User) GetERC20Balance(jsonrpc, erc20Contract string) (*big.Int, error) {
	var height int64
	contextHeight := big.NewInt(height)

	evmAddrs, err := GetEvmAddressFromAnyFormatAddress(user.Address, erc20Contract)
	fmt.Println(err)
	accountAddr := evmAddrs[0]
	contractAddr := evmAddrs[1]

	fmt.Println("Account", accountAddr)

	ethClient8545, err := ethclient.Dial(jsonrpc)
	if err != nil {
		fmt.Println("Failed to connect to EVM Json-RPC:", err)
		return big.NewInt(0), err
	}
	bz, err := ethClient8545.CallContract(context.Background(), ethereum.CallMsg{
		To:   &contractAddr,
		Data: append([]byte{0x70, 0xa0, 0x82, 0x31}, common.BytesToHash(accountAddr.Bytes()).Bytes()...), // balanceOf(address)
	}, contextHeight)
	if err != nil {
		return big.NewInt(0), err
	}

	tokenBalance := new(big.Int).SetBytes(bz)

	return tokenBalance, nil
}
