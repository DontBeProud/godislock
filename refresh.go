package godislock

import (
	"sync"
	"time"
)

type lockAutoRefreshController struct {
	ticker   	*time.Ticker
	quitChan 	chan bool
	quitOnce 	sync.Once
}

func createLockAutoRefreshController(milliseconds int64) *lockAutoRefreshController {
	return &lockAutoRefreshController{
		quitChan: make(chan bool),
		quitOnce: sync.Once{},
		ticker:   time.NewTicker(time.Duration(milliseconds / 3) * time.Millisecond),
	}
}

// stop auto refresh
func (c *lockAutoRefreshController) terminate(){
	c.quitOnce.Do(func() {
		c.quitChan <- true
		close(c.quitChan)
	})
}

func (c *lockAutoRefreshController) quitChannel() chan bool{
	return c.quitChan
}

func (c *lockAutoRefreshController) clock() <- chan time.Time{
	return c.ticker.C
}
