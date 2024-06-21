package cosmos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"

	sdkmath "cosmossdk.io/math"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
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

func (user *User) GetERC20Balance(jsonrpc, erc20Contract string) error {

	command := []string{
		"q", "balance", user.Address, erc20Contract, "--rpc", jsonrpc,
	}

	// Create the command
	cmd := exec.Command("devd", command...)
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
