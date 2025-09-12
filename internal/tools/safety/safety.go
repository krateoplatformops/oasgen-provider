package safety

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// RecursionGuard manages safe recursion settings
type RecursionGuard struct {
	maxDepth  int
	maxNodes  int32
	timeout   time.Duration
	nodeCount int32
}

func NewRecursionGuard(maxDepth int, maxNodes int32, timeout time.Duration) *RecursionGuard {
	return &RecursionGuard{
		maxDepth: maxDepth,
		maxNodes: maxNodes,
		timeout:  timeout,
	}
}

// WithContext creates a cancellable context for recursion
func (rg *RecursionGuard) WithContext() (context.Context, context.CancelFunc) {
	atomic.StoreInt32(&rg.nodeCount, 0)
	return context.WithTimeout(context.Background(), rg.timeout)
}

// Check verifies recursion constraints at each step
func (rg *RecursionGuard) Check(ctx context.Context, depth int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if depth > rg.maxDepth {
		return ErrMaxDepth
	}

	if atomic.AddInt32(&rg.nodeCount, 1) > rg.maxNodes {
		return ErrMaxNodes
	}

	return nil
}

var (
	ErrMaxDepth = fmt.Errorf("maximum recursion depth exceeded")
	ErrMaxNodes = fmt.Errorf("maximum recursion nodes exceeded")
)
