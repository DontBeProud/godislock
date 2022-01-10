package godislock

import (
	"context"
	"github.com/go-redis/redis/v8"
	"testing"
	"time"
)

const (
	lockNameInvalid	= ""
	lockName		= "testLockName"

	redisCon    	= "localhost:6379"
	redisPsw    	= ""
	redisDb     	= 7
)

var (
	ctx = context.Background()
	rdb = redis.NewClient(&redis.Options{
		Addr:     redisCon,
		Password: redisPsw,
		DB:       redisDb,
	})
)

func TestCreateRedisLockCreatorWithInvalidParam(t *testing.T) {
	_, err := CreateRedisLockCreator(ctx, lockNameInvalid, rdb)
	if err == nil{
		t.Error("lock name length error")
	}
}

func TestCreateRedisLockCreator(t *testing.T) {
	clear()
	creator, err := CreateRedisLockCreator(ctx, lockName, rdb)
	if err != nil{
		t.Error(err.Error())
		return
	}

	lock, err := creator.Acquire(ctx, time.Duration(30) * time.Second, time.Duration(3) * time.Second)
	if err != nil{
		t.Error(err.Error())
		return
	}
	defer lock.Release(ctx)
	go lock.AutoRefresh(ctx)

	// do something
	// xxxxxxxxxxxxxxxxxxxxxxxxxxxx

	clear()
}

func clear(){
	rdb.Del(ctx, "DistributedLockQueue_testLockName")
	rdb.Del(ctx, "DistributedLock_testLockName")
}