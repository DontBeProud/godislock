package redis_lock

import (
	"errors"
	"github.com/go-redis/redis/v8"
)

var(
	LockErrorInvalidRedisClient			= errors.New("godislock: redis is nil")
	LockErrorRedisCanNotConnect			= errors.New("godislock: connect redis fail")
	LockErrorRedisLockAcquireFail		= errors.New("godislock: acquire redis lock fail. ")
	LockErrorRedisQueueUpFail			= errors.New("godislock: redis queue up fail. ")
	LockErrorRedisLockBlocked			= errors.New("godislock: redis lock is in occupied state")
	LockErrorRedisSubScribeError 		= errors.New("godislock: subscribe redis channel fail")
	LockErrorRedisTokenNotMatch			= errors.New("godislock: token dos not match")
	LockErrorRedisReleaseProcedure		= errors.New("godislock: release redis lock fail. ")
	LockErrorRedisReleaseDeleteFail		= errors.New("godislock: delete key fail in the process of releasing the redis lock")
	LockErrorRedisRenewalProcedure		= errors.New("godislock: refresh redis lock fail")
	LockErrorRedisRenewalUnknownError	= errors.New("godislock: unknown error occurred in the process of releasing the redis lock")
	LockErrorRedisTimeOut				= errors.New("godislock: Waiting for redis lock timeout. ")
)

const(
	lockNamePrefix      = "DistributedLock_"
	lockQueueNamePrefix = "DistributedLockQueue_"
)

var(
	scriptReleaseLock = redis.NewScript(luaReleaseLock)
	scriptRenewalLock = redis.NewScript(luaRenewalLock)
	scriptQueueUp = redis.NewScript(luaQueueUp)
	scriptReJoin = redis.NewScript(luaAcquireLockAfterReceivingNotification)
)

var(
	luaReleaseLock = `
	-- return -1 if the token does not match
	if (redis.call('get', KEYS[1]) ~= ARGV[1]) then 
		return -1
	end

	-- release lock
	if (redis.call('del', KEYS[1]) ~= 1) then
	 	return -2
	end

	-- Publish messages to subscribers in the queue
	local publish_count = 0
	while(publish_count < 3)
	do
		local tempToken = redis.call('lpop', KEYS[2])
		if(not tempToken) then 
			break
		end

		if redis.call('get', tempToken) == tempToken then
			redis.call('del', tempToken)
			if redis.call('publish', tempToken, tempToken) == 1 then
				publish_count = publish_count + 1
			end
		end
	end

	return 1
	`

	luaRenewalLock = `
	-- token不符合则返回失败
	if (redis.call('get', KEYS[1]) ~= ARGV[1]) then 
		return -1
	end
	return redis.call('pexpire', KEYS[1], ARGV[2])
	`

	luaQueueUp = `
	-- 利用string的ttl功能控制实际的排队时长
	if not redis.call('set', KEYS[1], KEYS[1], 'nx','PX', ARGV[1]) then 
		return -1
	end

	-- 加入等待队列
	return redis.call('rpush', KEYS[2], KEYS[1])
	`

	luaAcquireLockAfterReceivingNotification = `
	-- 获取锁
	if (redis.call('set', KEYS[1], ARGV[1], 'nx', 'PX', ARGV[2])) then
		return 1
	end

	-- 重新加入队列
	if (not redis.call('set', ARGV[1], ARGV[1], 'nx', 'PX', ARGV[3])) then
		return -1
	end

	if (redis.call('lpush', KEYS[2], ARGV[1]) == 0) then
		return -2
	end
	
	return 0
	`
)

func generateLockName(srcLockName *string) string{
	return lockNamePrefix + *srcLockName
}

func generateLockQueueName(srcLockName *string) string{
	return lockQueueNamePrefix + *srcLockName
}