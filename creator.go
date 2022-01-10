package godislock

import (
	"context"
	"errors"
	. "github.com/DontBeProud/godislock/lock_creator"
	"github.com/go-basic/uuid"
	"strconv"
	"time"
)

const(
	lockPrefix = "lockId_"
)

var (
	LockErrorInvalidLockName 			= errors.New("godislock: lock name is invalid. The length of the value should be greater than zero. ")
	LockErrorInvalidTtl					= errors.New("godislock: ttl is too short. ")
)

type lockCreator struct {
	LockCreatorInterface
	lockName        string
	object        	LockCreatorFactoryInterface
}

func generateLockCreator(ctx context.Context, lockName string, creator LockCreatorFactoryInterface) (*lockCreator, error){
	if len(lockName) == 0{
		return nil, LockErrorInvalidLockName
	}

	if err := creator.CheckConnection(ctx); err != nil{
		return nil, err
	}

	return &lockCreator{
		object:        creator,
		lockName:      lockName,
	}, nil
}

func (o *lockCreator) Acquire(ctx context.Context, ttl time.Duration, waitTimeOut time.Duration) (LockInterface, error){

	// ttl should >= 3ms
	milliseconds := ttl.Milliseconds()
	if milliseconds < 3{
		return nil, LockErrorInvalidTtl
	}

	// acquire
	u := generateUuid()
	if err := o.object.Acquire(ctx, o.lockName, *u, ttl, waitTimeOut); err != nil{
		return nil, err
	}

	return &lock{
		lockSrc:    o,
		uuid:       u,
		ttl:        &ttl,
		refreshMgr: createLockAutoRefreshController(milliseconds),
	}, nil
}

func (o *lockCreator) refresh(ctx context.Context, uuid string, ttl time.Duration) error{
	return o.object.Refresh(ctx, o.lockName, uuid, ttl)
}

func (o *lockCreator) release(ctx context.Context, uuid string) error{
	return o.object.Release(ctx, o.lockName, uuid)
}

func generateUuid() *string{
	res := lockPrefix + strconv.Itoa(int(time.Now().UnixNano())) + "-" + uuid.New()
	return &res
}