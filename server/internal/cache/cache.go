package cache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"small-merchant-ops-hub-server/internal/config"
	"github.com/redis/go-redis/v9"
)

type Store interface {
	Ping(ctx context.Context) error
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Close() error
}

func New(cfg config.Config) (Store, error) {
	switch cfg.CacheMode {
	case "local":
		return newLocalStore(), nil
	case "redis":
		return newRedisStore(cfg.RedisURL)
	default:
		return nil, errors.New("unsupported cache mode")
	}
}

type localStore struct {
	mu   sync.RWMutex
	data map[string]string
}

func newLocalStore() *localStore {
	return &localStore{data: map[string]string{}}
}

func (l *localStore) Ping(context.Context) error {
	return nil
}

func (l *localStore) Get(_ context.Context, key string) (string, bool, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	value, ok := l.data[key]
	return value, ok, nil
}

func (l *localStore) Set(_ context.Context, key, value string, _ time.Duration) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.data[key] = value
	return nil
}

func (l *localStore) Close() error {
	return nil
}

type redisStore struct {
	client *redis.Client
}

func newRedisStore(redisURL string) (*redisStore, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	return &redisStore{client: redis.NewClient(opt)}, nil
}

func (r *redisStore) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *redisStore) Get(ctx context.Context, key string) (string, bool, error) {
	value, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

func (r *redisStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *redisStore) Close() error {
	return r.client.Close()
}
