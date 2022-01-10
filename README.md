# godislock
Safe and rigorous distributed lock. Currently, godislock supports redis distributed lock.
1. Different from the conventional polling request lock scheme, godislock realizes the notification function based on redis subscription and golang channel. godislock takes full advantage of golang to improve efficiency while reducing the pressure on redis caused by frequent requests
2. The bottom layer of godislock uses lua scripts to ensure atomicity, and high concurrency business scenarios can be used with confidence
3. The interface is simple, and efficient and secure distributed locks can be used in less than 10 lines of code


安全严谨的分布式锁. 目前支持redis版本的分布式锁
1. 不同于常规的轮询请求锁的方案，godislock基于redis订阅与golang的channel实现通知功能。godislock充分利用golang的优势，提高效率的同时减轻频繁请求造成的redis的压力.
2. godislock底层使用lua脚本确保原子性，高并发的业务场景可以放心使用.
3. 接口简洁，不到10行代码就能使用高效安全的分布式锁.



# Installation
Install godislock using the "go get" command:

    go get github.com/DontBeProud/godislock

# Examples(redis)
### step.1 import dependency(引入依赖包)
```
import (
    "context"
    "github.com/DontBeProud/godislock"
    "github.com/go-redis/redis/v8"
    "time"
)
```
### step.2 init lock creator(初始化锁生成器)
```
ctx := context.Background()
lockName := "XXXXXXXXXXXXXXXXXXXX"
rdb := redis.NewClient(&redis.Options{
    Addr:     "localhost:6379",
    Password: "",
    DB:       2,
})
creator, err := godislock.CreateRedisLockCreator(ctx, lockName, rdb)
if err != nil{
    return
}
```
### step.3 use locks where you need it
```
// ttl: life cycle of lock. it should longer than 3ms
// ttl: 锁的生命周期,避免程序崩溃后其他程序无法获取该锁, 它的值应该大于等于3毫秒.
ttl := time.Duration(30) * time.Second

// waitTimeOut: If the lock cannot be acquired after waiting for a while. godislock will return timeout.
// waitTimeOut: 如果等待一段时间后仍然无法获取该锁则会返回超时。
waitTimeOut := time.Duration(3) * time.Second
```
```
lck, err := creator.Acquire(ctx, ttl, waitTimeOut)
if err != nil{
    return err			// get lock fail or timeout
}
defer lck.Release(ctx)	// automatically release before return
go lck.AutoRefresh(ctx)	// extend lock ttl

doSomething()
```
OR
```
lck, err := creator.Acquire(ctx, ttl, waitTimeOut)
if err != nil{
    return err			// get lock fail or timeout
}
go lck.AutoRefresh(ctx)	// extend lock ttl

doSomething()
lck.Release(ctx)        // manual release after completing the task
```

# TODO
support etcd.