package example

import (
	"context"
	"fmt"
	"testing"

	"github.com/decentrio/e2e-testing-live/cosmos"
	"github.com/stretchr/testify/require"
)

var (
	address  string
	denom    string
	grpcAddr string
)

func TestIBCTransfer(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.Background()

	address = "dym139mq752delxv78jvtmwxhasyrycufsvrw4aka9"
	denom = "adym"
	grpcAddr = "0.0.0.0:8090"
	amount, err := cosmos.GetBalance(ctx, address, denom, grpcAddr)
	require.NoError(t, err)
	fmt.Println(amount)
	fmt.Println(err)
}
