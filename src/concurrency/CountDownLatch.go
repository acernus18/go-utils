package concurrency

import "sync"

type CountDownLatch struct {
	countValue int
	condition  *sync.Cond
}

func (latch *CountDownLatch) CountDown() {
	latch.condition.L.Lock()
	latch.countValue--
	if latch.countValue == 0 {
		latch.condition.Signal()
	}
	latch.condition.L.Unlock()
}

func (latch *CountDownLatch) Await() {
	latch.condition.L.Lock()
	for latch.countValue != 0 {
		latch.condition.Wait()
	}
	latch.condition.L.Unlock()
}

func NewCountDownLatch(value int) *CountDownLatch {
	return &CountDownLatch{value, sync.NewCond(new(sync.Mutex))}
}
