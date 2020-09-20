package concurrency

import (
	"sync"
)

type RoutinesPool struct {
	activeWorker int
	condition    *sync.Cond
	blockQueue   chan func()
}

func workHandler(pool *RoutinesPool) {
	// Consider channel as a block queue,
	// and range like poll method
	for task := range pool.blockQueue {
		task()
		pool.condition.L.Lock()
		pool.activeWorker--
		if !(pool.activeWorker > 0) {
			pool.condition.Signal()
		}
		pool.condition.L.Unlock()
	}
}

func NewRoutinesPool(size int) *RoutinesPool {
	instance := new(RoutinesPool)
	instance.condition = sync.NewCond(&sync.Mutex{})
	instance.blockQueue = make(chan func(), size)
	for i := 0; i < size; i++ {
		go workHandler(instance)
	}
	return instance
}

func (pool *RoutinesPool) Submit(task func()) {
	pool.condition.L.Lock()
	pool.activeWorker++
	pool.condition.L.Unlock()
	pool.blockQueue <- task
}

func (pool *RoutinesPool) Close() {
	pool.condition.L.Lock()
	for pool.activeWorker > 0 {
		pool.condition.Wait()
	}
	pool.condition.L.Unlock()
	close(pool.blockQueue)
}
