package godislock

import (
	"context"
	. "github.com/DontBeProud/godislock/lock_creator"
	"github.com/go-redis/redis/v8"
	"time"
)

type LockInterface interface {
	AutoRefresh(ctx context.Context)
	Release(ctx context.Context)
}

type LockCreatorInterface interface {
	Acquire(ctx context.Context, ttl time.Duration, waitTimeOut time.Duration) (LockInterface, error)
}

func CreateRedisLockCreator(ctx context.Context, lockName string, rDb *redis.Client) (LockCreatorInterface, error){
	return generateLockCreator(ctx, lockName, &LockCreatorRedis{
		Rdb: rDb,
	})
}
