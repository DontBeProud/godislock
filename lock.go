package godislock

import (
	"context"
	"time"
)

type lock struct {
	LockInterface
	uuid		*string
	ttl 		*time.Duration
	lockSrc 	*lockCreator
	refreshMgr	*lockAutoRefreshController
}

func (l *lock) Release(ctx context.Context){
	l.refreshMgr.terminate()
	_ = l.lockSrc.release(ctx, *l.uuid)
}

func (l *lock) AutoRefresh(ctx context.Context){
	for{
		select {
		case <- l.refreshMgr.clock():
			if err := l.lockSrc.refresh(ctx, *l.uuid, *l.ttl); err != nil{
				// retry
				_ = l.lockSrc.refresh(ctx, *l.uuid, *l.ttl)
			}
		case <- l.refreshMgr.quitChannel():
			return
		}
	}
}