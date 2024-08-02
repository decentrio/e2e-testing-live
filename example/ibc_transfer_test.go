package example

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	"github.com/decentrio/e2e-testing-live/cosmos"
	"github.com/decentrio/e2e-testing-live/testutil"
	"github.com/decentrio/rollup-e2e-testing/ibc"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	channelIDDymRollappX = "channel-17"
	channelIDDymRollappY = "channel-22"
	channelIDRollappXDym = "channel-0"
	channelIDRollappYDym = "channel-0"
	dymFee               = "6000000000000000adym"
	rolxFee              = "10000000000000arolx"
	rolyFee              = "2000000000000000aroly"
	erc20Addr            = "rolx1glht96kr2rseywuvhhay894qw7ekuc4q4d4qs2"
	erc20IBCDenom        = "ibc/FECACB927EB3102CCCB240FFB3B6FCCEEB8D944C6FEA8DFF079650FEFF59781D"
)

func TestIBCTransfer(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.Background()

	hub := cosmos.CosmosChain{
		RPCAddr:       "rpc-blumbus.mzonder.com:443",
		GrpcAddr:      "grpc-blumbus.mzonder.com:9090",
		ChainID:       "blumbus_111-1",
		Bin:           "dymd",
		GasPrices:     "1000adym",
		GasAdjustment: "1.1",
		Denom:         "adym",
	}

	rollappX := cosmos.CosmosChain{
		RPCAddr:       "rpc.rolxtwo.evm.ra.blumbus.noisnemyd.xyz:443",
		GrpcAddr:      "3.123.185.77:9090",
		ChainID:       "rolx_100004-1",
		Bin:           "rollapp-evm",
		GasPrices:     "0.0arolx",
		GasAdjustment: "1.1",
		Denom:         "arolx",
	}

	rollappY := cosmos.CosmosChain{
		RPCAddr:       "rpc.roly.wasm.ra.blumbus.noisnemyd.xyz:443",
		GrpcAddr:      "18.153.150.111:9090",
		ChainID:       "rollappy_700002-1",
		Bin:           "rollapp-wasm",
		GasPrices:     "0.0aroly",
		GasAdjustment: "1.1",
		Denom:         "aroly",
	}

	dymensionUser, err := hub.CreateUser("dym1")
	require.NoError(t, err)
	rollappXUser, err := rollappX.CreateUser("rolx1")
	require.NoError(t, err)
	rollappYUser, err := rollappY.CreateUser("roly1")
	require.NoError(t, err)

	err = hub.NewClient("https://" + hub.RPCAddr)
	require.NoError(t, err)

	err = rollappX.NewClient("https://" + rollappX.RPCAddr)
	require.NoError(t, err)

	err = rollappY.NewClient("https://" + rollappY.RPCAddr)
	require.NoError(t, err)

	dymensionUser.GetFaucet("http://18.184.170.181:3000/api/get-dym")
	rollappXUser.GetFaucet("http://18.184.170.181:3000/api/get-rollx")
	rollappYUser.GetFaucet("http://18.184.170.181:3000/api/get-rolly")

	// Wait for blocks
	testutil.WaitForBlocks(ctx, 5, hub)

	// Get the IBC denom
	rollappTokenDenom := transfertypes.GetPrefixedDenom("transfer", channelIDDymRollappX, rollappXUser.Denom)
	rollappIBCDenom := transfertypes.ParseDenomTrace(rollappTokenDenom).IBCDenom()

	hubTokenDenom := transfertypes.GetPrefixedDenom("transfer", channelIDRollappXDym, dymensionUser.Denom)
	hubIBCDenom := transfertypes.ParseDenomTrace(hubTokenDenom).IBCDenom()

	dymensionOrigBal, err := dymensionUser.GetBalance(ctx, dymensionUser.Denom, hub.GrpcAddr)
	require.NoError(t, err)
	fmt.Println(dymensionOrigBal)

	rollappXOrigBal, err := rollappXUser.GetBalance(ctx, rollappXUser.Denom, rollappX.GrpcAddr)
	require.NoError(t, err)
	fmt.Println(rollappXOrigBal)

	rollappYOrigBal, err := rollappYUser.GetBalance(ctx, rollappYUser.Denom, rollappY.GrpcAddr)
	require.NoError(t, err)
	fmt.Println(rollappYOrigBal)

	erc20_OrigBal, err := GetERC20Balance(ctx, erc20IBCDenom, rollappX.GrpcAddr)
	require.NoError(t, err)
	fmt.Println(erc20_OrigBal)

	transferAmount := math.NewInt(1_000_000)

	// Compose an IBC transfer and send from dymension -> rollapp
	transferData := ibc.WalletData{
		Address: rollappXUser.Address,
		Denom:   dymensionUser.Denom,
		Amount:  transferAmount,
	}

	testutil.WaitForBlocks(ctx, 3, hub)

	cosmos.SendIBCTransfer(hub, channelIDDymRollappX, dymensionUser.Address, transferData, dymFee, ibc.TransferOptions{})
	require.NoError(t, err)

	testutil.WaitForBlocks(ctx, 3, hub)
	// Compose an IBC transfer and send from rollapp -> hub
	transferData = ibc.WalletData{
		Address: dymensionUser.Address,
		Denom:   rollappXUser.Denom,
		Amount:  transferAmount,
	}

	multiplier := math.NewInt(10)
	eibcFee := transferAmount.Quo(multiplier) // transferAmount * 0.1

	var options ibc.TransferOptions
	// set eIBC specific memo
	options.Memo = BuildEIbcMemo(eibcFee)
	cosmos.SendIBCTransfer(rollappX, channelIDRollappXDym, rollappXUser.Address, transferData, rolxFee, options)
	require.NoError(t, err)

	testutil.WaitForBlocks(ctx, 10, hub)

	erc20_Bal, err := GetERC20Balance(ctx, hubIBCDenom, rollappX.GrpcAddr)
	require.NoError(t, err)
	fmt.Println(erc20_Bal)
	fmt.Println(rollappIBCDenom)
	testutil.AssertBalance(t, ctx, dymensionUser, rollappIBCDenom, hub.GrpcAddr, transferAmount.Sub(eibcFee))
	require.Equal(t, erc20_OrigBal.Add(transferAmount), erc20_Bal)
}

func BuildEIbcMemo(eibcFee math.Int) string {
	return fmt.Sprintf(`{"eibc": {"fee": "%s"}}`, eibcFee.String())
}
func GetERC20Balance(ctx context.Context, denom, grpcAddr string) (sdkmath.Int, error) {
	params := &bankTypes.QueryBalanceRequest{Address: erc20Addr, Denom: denom}
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
