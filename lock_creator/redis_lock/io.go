package redis_lock

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"time"
)

func Acquire(ctx context.Context, rdb *redis.Client, lockName string, uuid string, ttl time.Duration, timeOut time.Duration) error{
	// try to acquire lock
	success, err := rdb.SetNX(ctx, generateLockName(&lockName), uuid, ttl).Result()
	if success{
		return nil
	}

	// fatal error
	if err != nil {
		return errors.New(LockErrorRedisLockAcquireFail.Error() + err.Error())
	}

	// subscribe redis and wait for notification
	sub := rdb.Subscribe(ctx, uuid)
	if sub == nil{
		return LockErrorRedisSubScribeError
	}
	defer sub.Close()

	// queue up
	clock := time.Tick(timeOut)
	deadline := time.Now().Add(timeOut)
	if err = queueUp(ctx, rdb, lockName, uuid, deadline); err != nil{
		return err
	}

	// clear
	defer func() {
		if err != nil{
			rdb.LRem(ctx, generateLockQueueName(&lockName), 0, uuid)
			rdb.Del(ctx, uuid)
		}
	}()

	for{
		select {
		case <- clock:
			return LockErrorRedisTimeOut
		case msg := <- sub.Channel():
			if msg.Payload != uuid{
				continue
			}
			// 再次获取锁
			if err = reAcquire(ctx, rdb, lockName, uuid, ttl, deadline); err != LockErrorRedisLockBlocked {
				return err
			}
		}
	}
}

func Refresh(ctx context.Context, rdb *redis.Client, lockName string, uuid string, ttl time.Duration) error{
	code, err := scriptRenewalLock.Run(ctx, rdb, []string{generateLockName(&lockName)}, uuid, ttl.Milliseconds()).Int64()
	if err != nil{
		return LockErrorRedisRenewalProcedure
	}
	switch code {
	case 0:
		return LockErrorRedisRenewalUnknownError
	case -1:
		return LockErrorRedisTokenNotMatch
	}

	return nil
}

func Release(ctx context.Context, rdb *redis.Client, lockName string, uuid string) error{
	code, err := scriptReleaseLock.Run(ctx, rdb, []string{generateLockName(&lockName), generateLockQueueName(&lockName)}, uuid).Int64()
	if err != nil{
		return errors.New(LockErrorRedisReleaseProcedure.Error() + err.Error())
	}
	switch code {
	case -1:
		return LockErrorRedisTokenNotMatch
	case -2:
		return LockErrorRedisReleaseDeleteFail
	}
	return nil
}

func CheckConnection(ctx context.Context, rdb *redis.Client) error{
	if rdb == nil{
		return LockErrorInvalidRedisClient
	}

	if _, err := rdb.Ping(ctx).Result(); err != nil{
		return LockErrorRedisCanNotConnect
	}
	return nil
}

func reAcquire(ctx context.Context, rdb *redis.Client, lockName string, uuid string, ttl time.Duration, deadline time.Time) error{
	code, err := scriptReJoin.Run(ctx, rdb, []string{generateLockName(&lockName), generateLockQueueName(&lockName)}, uuid, ttl.Milliseconds(), deadline.UnixMilli() - time.Now().UnixMilli()).Int64()
	if err != nil {
		return errors.New(LockErrorRedisLockAcquireFail.Error() + err.Error())
	}
	switch code {
	case 0:
		return LockErrorRedisLockBlocked
	case -1:
		return errors.New(LockErrorRedisQueueUpFail.Error() + "reset fail")
	case -2:
		return errors.New(LockErrorRedisQueueUpFail.Error() + "rejoin fail")
	}
	return nil
}

func queueUp(ctx context.Context, rdb *redis.Client, lockName string, uuid string, deadline time.Time) error{
	code, err := scriptQueueUp.Run(ctx, rdb, []string{uuid, generateLockQueueName(&lockName)}, deadline.UnixMilli() - time.Now().UnixMilli()).Int64()
	if err != nil{
		return errors.New(LockErrorRedisQueueUpFail.Error() + err.Error())
	}
	switch code {
	case 0:
		return LockErrorRedisQueueUpFail
	case -1:
		return errors.New(LockErrorRedisQueueUpFail.Error() + "set fail")
	}
	return nil
}

