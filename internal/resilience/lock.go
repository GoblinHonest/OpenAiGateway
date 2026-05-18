package resilience

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type DistributedLock struct {
	redis   *redis.Client
	key     string
	value   string
	ttl     time.Duration
	cancel  context.CancelFunc
	mu      sync.Mutex
	held    bool
}

func NewDistributedLock(redis *redis.Client, key string, ttl time.Duration) *DistributedLock {
	return &DistributedLock{
		redis: redis,
		key:   key,
		value: uuid.New().String(),
		ttl:   ttl,
	}
}

func (l *DistributedLock) Acquire(ctx context.Context) (bool, error) {
	if l.redis == nil {
		l.mu.Lock()
		l.held = true
		l.mu.Unlock()
		return true, nil
	}

	success, err := l.redis.SetNX(ctx, l.key, l.value, l.ttl).Result()
	if err != nil {
		return false, err
	}

	if success {
		l.mu.Lock()
		l.held = true
		l.mu.Unlock()
		l.startAutoRenew(ctx)
	}

	return success, nil
}

func (l *DistributedLock) startAutoRenew(ctx context.Context) {
	ctx, l.cancel = context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(l.ttl / 3)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if l.redis == nil {
					return
				}
				script := redis.NewScript(`
					if redis.call("get", KEYS[1]) == ARGV[1] then
						return redis.call("pexpire", KEYS[1], ARGV[2])
					else
						return 0
					end
				`)
				script.Run(ctx, l.redis, []string{l.key}, l.value, l.ttl.Milliseconds())
			}
		}
	}()
}

func (l *DistributedLock) Release(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.held {
		return nil
	}
	l.held = false

	if l.cancel != nil {
		l.cancel()
	}

	if l.redis == nil {
		return nil
	}

	script := redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`)

	result, err := script.Run(ctx, l.redis, []string{l.key}, l.value).Result()
	if err != nil {
		return err
	}

	if result.(int64) == 0 {
		return errors.New("lock not held by this process")
	}

	return nil
}

type LocalLock struct {
	mu    sync.Mutex
	owner string
}

func NewLocalLock() *LocalLock {
	return &LocalLock{owner: uuid.New().String()}
}

func (l *LocalLock) Lock() {
	l.mu.Lock()
}

func (l *LocalLock) Unlock() {
	l.mu.Unlock()
}

func LockKey(parts ...string) string {
	key := ""
	for i, p := range parts {
		if i > 0 {
			key += ":"
		}
		key += p
	}
	return fmt.Sprintf("lock:%s", key)
}
