package redis

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"gophermind/internal/config"
)

func TestSessionCache_FallbackMode(t *testing.T) {
	cache := NewSessionCache(config.RedisConfig{
		Addrs: []string{"127.0.0.1:6399"},
	}, zap.NewNop())
	require.True(t, cache.IsDegraded())

	ctx := context.Background()
	require.NoError(t, cache.SetSummary(ctx, "u1", "s1", "summary", time.Minute))
	got, ok, err := cache.GetSummary(ctx, "u1", "s1")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "summary", got)

	require.NoError(t, cache.MarkIdempotent(ctx, "c1", "m1", time.Minute))
	dup, err := cache.IsIdempotent(ctx, "c1", "m1")
	require.NoError(t, err)
	require.True(t, dup)
}
