package safety

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRecursionGuard_Check(t *testing.T) {
	t.Run("should return no error when limits are not exceeded", func(t *testing.T) {
		guard := NewRecursionGuard(10, 100, 1*time.Second)
		ctx, cancel := guard.WithContext()
		defer cancel()

		err := guard.Check(ctx, 5)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), guard.nodeCount)
	})

	t.Run("should return ErrMaxDepth when depth is exceeded", func(t *testing.T) {
		guard := NewRecursionGuard(5, 100, 1*time.Second)
		ctx, cancel := guard.WithContext()
		defer cancel()

		err := guard.Check(ctx, 6)
		assert.Error(t, err)
		assert.Equal(t, ErrMaxDepth, err)
	})

	t.Run("should return ErrMaxNodes when node count is exceeded", func(t *testing.T) {
		guard := NewRecursionGuard(10, 5, 1*time.Second)
		ctx, cancel := guard.WithContext()
		defer cancel()

		for i := 0; i < 5; i++ {
			assert.NoError(t, guard.Check(ctx, 1))
		}

		err := guard.Check(ctx, 1)
		assert.Error(t, err)
		assert.Equal(t, ErrMaxNodes, err)
		assert.Equal(t, int32(6), guard.nodeCount)
	})

	t.Run("should return context deadline exceeded when timeout occurs", func(t *testing.T) {
		guard := NewRecursionGuard(10, 100, 10*time.Millisecond)
		ctx, cancel := guard.WithContext()
		defer cancel()

		time.Sleep(20 * time.Millisecond) // Wait for the context to timeout

		err := guard.Check(ctx, 1)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})

	t.Run("should handle zero values gracefully", func(t *testing.T) {
		guard := NewRecursionGuard(0, 0, 1*time.Second)
		ctx, cancel := guard.WithContext()
		defer cancel()

		// First check should fail on max nodes (1 > 0)
		err := guard.Check(ctx, 0)
		assert.ErrorIs(t, err, ErrMaxNodes)

		// Second check should fail on max depth
		err = guard.Check(ctx, 1)
		assert.ErrorIs(t, err, ErrMaxDepth)
	})

	t.Run("WithContext should reset node count", func(t *testing.T) {
		guard := NewRecursionGuard(10, 100, 1*time.Second)

		// First run
		ctx1, cancel1 := guard.WithContext()
		guard.Check(ctx1, 1)
		guard.Check(ctx1, 1)
		assert.Equal(t, int32(2), guard.nodeCount)
		cancel1()

		// Second run should have a reset counter
		ctx2, cancel2 := guard.WithContext()
		guard.Check(ctx2, 1)
		assert.Equal(t, int32(1), guard.nodeCount)
		cancel2()
	})
}
