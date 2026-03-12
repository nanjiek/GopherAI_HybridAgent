package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"gophermind/internal/config"
)

// NewClusterClient 初始化 Redis Cluster 客户端。
func NewClusterClient(cfg config.RedisConfig) *redis.ClusterClient {
	return redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        cfg.Addrs,
		Username:     cfg.Username,
		Password:     cfg.Password,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  100 * time.Millisecond,
		ReadTimeout:  150 * time.Millisecond,
		WriteTimeout: 150 * time.Millisecond,
	})
}

// Ping 用于探测 Redis 可用性。
func Ping(ctx context.Context, cli *redis.ClusterClient) error {
	_, err := cli.Ping(ctx).Result()
	return err
}
