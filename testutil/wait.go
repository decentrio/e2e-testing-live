package testutil

import (
	"context"

	"github.com/decentrio/e2e-testing-live/cosmos"
)

// WaitForBlocks blocks until all chains reach a block height delta equal to or greater than the delta argument.
// If a ChainHeighter does not monotonically increase the height, this function may block program execution indefinitely.
func WaitForBlocks(ctx context.Context, delta int, chain cosmos.CosmosChain) error {
	h := &height{Chain: chain}
	return h.WaitForDelta(ctx, delta)

}

type height struct {
	Chain cosmos.CosmosChain

	starting uint64
	current  uint64
}

func (h *height) WaitForDelta(ctx context.Context, delta int) error {
	for h.delta() < delta {
		cur, err := h.Chain.Height(ctx)
		if err != nil {
			return err
		}
		// We assume the chain will eventually return a non-zero height, otherwise
		// this may block indefinitely.
		if cur == 0 {
			continue
		}
		h.update(cur)
	}
	return nil
}

func (h *height) delta() int {
	if h.starting == 0 {
		return 0
	}
	return int(h.current - h.starting)
}

func (h *height) update(height uint64) {
	if h.starting == 0 {
		h.starting = height
	}
	h.current = height
}
