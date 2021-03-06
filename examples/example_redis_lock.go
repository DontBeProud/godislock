package main

import (
	"context"
	"github.com/DontBeProud/godislock"
	"github.com/go-redis/redis/v8"
	"sync"
	"time"
)

const (
	productStockQuantity = "testProductStockQuantity"
)

// simulate a flash sale scene with redis distributed lock 模拟使用分布式锁处理秒杀活动的场景
func handleFlashSaleWithRedisDistributedLock(ctx context.Context, creator godislock.LockCreatorInterface, rdb *redis.Client) error{
	////////////////////////////////////////////////////////////////////////////////////////////
	////////////////////////////////////////////////////////////////////////////////////////////
	///////////////////////////////////////	use lock ///////////////////////////////////////////

	ttl := time.Duration(30) * time.Second
	waitTimeOut := time.Duration(20) * time.Second
	lck, err := creator.Acquire(ctx, ttl, waitTimeOut)
	if err != nil{
		println("receive err. ", err.Error())
		return err			// get lock fail or timeout
	}
	go lck.AutoRefresh(ctx)	// extend lock ttl
	///////////////////////////////////////	use lock ///////////////////////////////////////////
	////////////////////////////////////////////////////////////////////////////////////////////
	////////////////////////////////////////////////////////////////////////////////////////////

	//time.Sleep(10 * time.Millisecond)
	// do something
	res := handleFlashSale(ctx, rdb)
	lck.Release(ctx)
	return res
}

// simulate a flash sale scene 模拟秒杀活动场景
func handleFlashSale(ctx context.Context, rdb *redis.Client) error{
	quantity, _ := rdb.Get(ctx, productStockQuantity).Int64()
	switch {
	case quantity > 0:
		rdb.Decr(ctx, productStockQuantity)
		println("sell success (售卖成功)", quantity)
	case quantity == 0:
		println("sold out and not oversold (售罄, 且未超卖)", quantity)
	case quantity < 0:
		println("oversold (超卖)", quantity)
	}

	return nil
}

func main(){
	// 本demo模拟秒杀活动(高并发场景),同时介绍使用godislock确保商品状态一致性，避免超卖的方法

	////////////////////////////////////////////////////////////////////////////////////////////
	////////////////////////////////////////////////////////////////////////////////////////////
	//////////////////////////////////	init lock creator  /////////////////////////////////////
	lockName := "XXXXXXXXXXXXXXXXXXXX"
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       6,
	})
	creator, err := godislock.CreateRedisLockCreator(ctx, lockName, rdb)
	if err != nil{
		return
	}
	//////////////////////////////////	init lock creator  /////////////////////////////////////
	////////////////////////////////////////////////////////////////////////////////////////////
	////////////////////////////////////////////////////////////////////////////////////////////

	wg := sync.WaitGroup{}

	start := time.Now()
	// 不使用分布式锁，在高并发场景下必然出现超卖
	rdb.Set(ctx, productStockQuantity, 200, 1*time.Hour)

	for i := 0; i < 5000; i++{
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = handleFlashSale(ctx, rdb)
		}()
	}
	wg.Wait()
	End := time.Now()
	println(End.Sub(start).Milliseconds())
	println("-------------------------------- 不使用分布式锁，在高并发场景下必然出现超卖 --------------------------------")
	println("---------------------------------------------- 等待1秒 ----------------------------------------------")
	time.Sleep(1*time.Second)


	// 使用分布式锁可避免超卖
	start = time.Now()
	rdb.Set(ctx, productStockQuantity, 200, 1*time.Hour)
	for i := 0; i < 5000; i++{
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = handleFlashSaleWithRedisDistributedLock(ctx, creator, rdb)
		}()
	}
	wg.Wait()
	End = time.Now()
	println(End.Sub(start).Milliseconds())
}
