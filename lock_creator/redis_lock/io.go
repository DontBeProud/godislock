package redis_lock

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"sync"
	"time"
)

func Acquire(ctx context.Context, rdb *redis.Client, lockName string, uuid string, ttl time.Duration, timeOut time.Duration) error{

	// try to acquire lock
	if success, err := rdb.SetNX(ctx, generateLockName(&lockName), uuid, ttl).Result(); success{
		return nil
	}else if err != nil {
		return errors.New(LockErrorRedisLockAcquireFail.Error() + err.Error())			// fatal error
	}

	wg := sync.WaitGroup{}
	quitChan := make(chan bool)
	resChan := make(chan error)

	// timed execute
	go executeTimedTask(ctx, rdb, lockName, uuid, ttl, &wg, timeOut, quitChan, resChan)
	// subscribe
	go acquireBySubscribeQueue(ctx, rdb, lockName, uuid, ttl, &wg, timeOut, quitChan, resChan)

	res := <-resChan
	close(quitChan)
	wg.Wait()
	return res
}

func Refresh(ctx context.Context, rdb *redis.Client, lockName string, uuid string, ttl time.Duration) error{
	code, err := scriptRefreshLock.Run(ctx, rdb, []string{generateLockName(&lockName)}, uuid, ttl.Milliseconds()).Int64()
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
	code, err := scriptReleaseLock.Run(ctx, rdb, []string{generateLockQueueName(&lockName), generateLockName(&lockName)}, uuid).Int64()
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

func executeTimedTask(ctx context.Context, rdb *redis.Client, lockName string, uuid string, ttl time.Duration, wg *sync.WaitGroup, timeOut time.Duration, quitChan chan bool, resChan chan error){
	wg.Add(1)
	defer wg.Done()
	timeOutClock := time.Tick(timeOut)
	activeAcquireClock := time.Tick(300 * time.Millisecond)
	for{
		select {
		case <- quitChan:
			return
		case <- timeOutClock:
			rdb.Del(ctx, uuid)
			resChan <- LockErrorRedisTimeOut
			return
		case <-activeAcquireClock:
			err := activeAcquire(ctx, rdb, lockName, uuid, ttl)
			if err == nil || err.Error() != LockErrorRedisLockBlocked.Error() {
				resChan <- err
				return
			}
		}
	}
}

func acquireBySubscribeQueue(ctx context.Context, rdb *redis.Client, lockName string, uuid string, ttl time.Duration, wg *sync.WaitGroup, timeOut time.Duration, quitChan chan bool, resChan chan error){
	wg.Add(1)
	defer wg.Done()

	queueSyncChan := make(chan error)
	deadline := time.Now().Add(timeOut)
	go func() {
		if deadline.UnixNano() > time.Now().UnixNano(){
			success, err := rdb.SetNX(ctx, uuid, uuid, deadline.Sub(time.Now())).Result()
			if err != nil || success{
				queueSyncChan <- err
			}else {
				queueSyncChan <- LockErrorRedisQueueUpFail
			}
		}else {
			queueSyncChan <- LockErrorRedisTimeOut
		}
	}()

	// subscribe redis and wait for notification
	sub := rdb.Subscribe(ctx, uuid)
	if sub == nil{
		resChan <- LockErrorRedisSubScribeError
		return
	}
	defer sub.Close()

	if qErr := <- queueSyncChan; qErr != nil{
		resChan <- qErr
		return
	}

	if pErr := rdb.RPush(ctx, generateLockQueueName(&lockName), uuid).Err(); pErr != nil{
		resChan <- pErr
		return
	}

	for{
		select {
		case <- quitChan:
			return
		case <- sub.Channel():
			// 再次获取锁
			err := reAcquire(ctx, rdb, lockName, uuid, ttl, deadline)
			if err == nil || err.Error() != LockErrorRedisLockBlocked.Error() {
				resChan <- err
				return
			}
		}
	}
}

func reAcquire(ctx context.Context, rdb *redis.Client, lockName string, uuid string, ttl time.Duration, deadline time.Time) error{
	if deadline.UnixNano() <= time.Now().UnixNano(){
		return LockErrorRedisTimeOut
	}

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

func activeAcquire(ctx context.Context, rdb *redis.Client, lockName string, uuid string, ttl time.Duration) error{
	res, err := scriptActiveAcquire.Run(ctx, rdb, []string{generateLockName(&lockName)}, uuid, ttl.Milliseconds()).Int64()
	if err != nil{
		return errors.New(LockErrorRedisActiveAcquire.Error() + err.Error())
	}
	if res == 0{
		return LockErrorRedisLockBlocked
	}
	return nil
}
