package example

import (
	"context"
	"fmt"
	"testing"

	"github.com/decentrio/e2e-testing-live/cosmos"
	"github.com/stretchr/testify/require"
)

func TestIBCTransfer(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.Background()

	dymensionUser := cosmos.User{
		Address: "dym139mq752delxv78jvtmwxhasyrycufsvrw4aka9",
		Denom:   "adym",
	}
	rollappUser := cosmos.User{
		Address: "ethm1q5f3a8acs5yenfrxu9v49p49rn7gcy0nd94ruk",
		Denom:   "arax",
	}

	dymensionGrpcAddr := "0.0.0.0:8090"
	rollappGrpcAddr := "0.0.0.0:9090"

	dymensionOrigBal, err := cosmos.GetBalance(ctx, dymensionUser.Address, dymensionUser.Denom, dymensionGrpcAddr)
	require.NoError(t, err)
	fmt.Println(dymensionOrigBal)

	rollappOrigBal, err := cosmos.GetBalance(ctx, rollappUser.Address, rollappUser.Denom, rollappGrpcAddr)
	require.NoError(t, err)
	fmt.Println(rollappOrigBal)

}
