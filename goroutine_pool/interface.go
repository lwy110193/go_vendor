package goroutine_pool

import (
	"fmt"
	"sync"

	"github.com/panjf2000/ants/v2"
)

// NewPool 新建普通协程池, 包含错误收集
func NewPool(size int, opts ...ants.Option) (*pool, error) {
	if size <= 0 {
		size = 50
	}
	p, err := ants.NewPool(size, opts...)
	if err != nil {
		return nil, err
	}
	return &pool{pool: p}, nil
}

type pool struct {
	pool    *ants.Pool
	wg      sync.WaitGroup
	errList []error
	mtx     sync.Mutex
}

func (p *pool) Submit(task func()) error {
	p.wg.Add(1)
	return p.pool.Submit(
		func() {
			defer p.wg.Done()
			defer func() {
				if err := recover(); err != nil {
					p.mtx.Lock()
					p.errList = append(p.errList, fmt.Errorf("panic: %v", err))
					p.mtx.Unlock()
				}
			}()
			task()
		})
}

func (p *pool) Running() int {
	return p.pool.Running()
}

func (p *pool) Wait() {
	p.wg.Wait()
}

func (p *pool) Release() {
	p.pool.Release()
}

func (p *pool) ErrList() []error {
	return p.errList
}

// NewFuncPool 新建函数协程池, 包含错误收集
func NewFuncPool(size int, runTask func(i interface{}), opts ...ants.Option) (*funcPool, error) {
	if size <= 0 {
		size = 50
	}
	pool := &funcPool{}
	p, err := ants.NewPoolWithFunc(size, func(i interface{}) {
		defer func() {
			defer pool.wg.Done()
			if err := recover(); err != nil {
				pool.mtx.Lock()
				pool.errList = append(pool.errList, fmt.Errorf("panic: %v", err))
				pool.mtx.Unlock()
			}
		}()
		runTask(i)
	}, opts...)
	if err != nil {
		return nil, err
	}
	pool.pool = p
	return pool, nil
}

type funcPool struct {
	pool    *ants.PoolWithFunc
	mtx     sync.Mutex
	errList []error
	wg      sync.WaitGroup
}

func (p *funcPool) Invoke(i interface{}) error {
	p.wg.Add(1)
	return p.pool.Invoke(i)
}

func (p *funcPool) Waiting() {
	p.wg.Wait()
}

func (p *funcPool) Release() {
	p.pool.Release()
}

func (p *funcPool) ErrList() []error {
	return p.errList
}
