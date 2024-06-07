package testutil

import (
	"context"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/decentrio/e2e-testing-live/cosmos"
	"github.com/stretchr/testify/require"
)

func AssertBalance(t *testing.T, ctx context.Context, user cosmos.User, denom, grpc string, expectedBalance sdkmath.Int) {
	balance, err := user.GetBalance(ctx, denom, grpc)
	require.NoError(t, err)
	require.Equal(t, expectedBalance.String(), balance.String())
}
