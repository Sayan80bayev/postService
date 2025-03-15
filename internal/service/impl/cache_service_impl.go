package impl

import (
	"context"
	"github.com/redis/go-redis/v9"
	"time"
)

type CacheServiceImpl struct {
	client *redis.Client
}

func NewCacheService(client *redis.Client) *CacheServiceImpl {
	return &CacheServiceImpl{client: client}
}

func (c *CacheServiceImpl) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

func (c *CacheServiceImpl) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

func (c *CacheServiceImpl) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *CacheServiceImpl) Publish(ctx context.Context, channel, message string) error {
	return c.client.Publish(ctx, channel, message).Err()
}

func (c *CacheServiceImpl) Exists(ctx context.Context, key string) (bool, error) {
	res, err := c.client.Exists(ctx, key).Result()
	return res > 0, err
}

func (c *CacheServiceImpl) Subscribe(ctx context.Context, channel string) *redis.PubSub {
	return c.client.Subscribe(ctx, channel)
}
