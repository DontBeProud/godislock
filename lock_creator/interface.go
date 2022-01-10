package lock_creator

import (
	"context"
	"github.com/DontBeProud/godislock/lock_creator/redis_lock"
	"github.com/go-redis/redis/v8"
	"time"
)

type LockCreatorFactoryInterface interface {
	CheckConnection(ctx context.Context) error
	Acquire(ctx context.Context, lockName string, uuid string, ttl time.Duration, timeOut time.Duration) error
	Refresh(ctx context.Context, lockName string, uuid string, ttl time.Duration) error
	Release(ctx context.Context, lockName string, uuid string) error
}

type LockCreatorRedis struct {
	LockCreatorFactoryInterface
	Rdb *redis.Client
}

func (r LockCreatorRedis) CheckConnection(ctx context.Context) error{
	return redis_lock.CheckConnection(ctx, r.Rdb)
}

func (r LockCreatorRedis) Acquire(ctx context.Context, lockName string, uuid string, ttl time.Duration, timeOut time.Duration) error{
	return redis_lock.Acquire(ctx, r.Rdb, lockName, uuid, ttl, timeOut)
}

func (r LockCreatorRedis) Refresh(ctx context.Context, lockName string, uuid string, ttl time.Duration) error{
	return redis_lock.Refresh(ctx, r.Rdb, lockName, uuid, ttl)
}

func (r LockCreatorRedis) Release(ctx context.Context, lockName string, uuid string) error{
	return redis_lock.Release(ctx, r.Rdb, lockName, uuid)
}
