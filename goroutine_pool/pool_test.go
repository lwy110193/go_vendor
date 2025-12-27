package goroutine_pool_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	goroutinepool "github.com/lwy110193/go_vendor/goroutine_pool"
	"github.com/panjf2000/ants/v2"
)

func TestGoroutinePool(t *testing.T) {
	defer ants.Release() // 1. 程序结束时释放全局资源

	var wg sync.WaitGroup
	// 2. 创建一个容量为10的协程池
	p, _ := ants.NewPool(10)
	defer p.Release() // 3. 释放该池

	eList := []error{}
	for i := 0; i < 30; i++ {
		wg.Add(1)
		taskNum := i // 注意闭包问题
		// 4. 提交任务到池中
		eList = append(eList, p.Submit(func() {
			defer wg.Done()
			fmt.Printf("执行任务 %d\n", taskNum)
			time.Sleep(100 * time.Millisecond)
		}))
	}
	wg.Wait()
	fmt.Printf("运行中协程数: %d\n", p.Running())
	for _, err := range eList {
		if err != nil {
			t.Errorf(" %v", err)
		}
	}
}

func TestGoroutinePool1(t *testing.T) {
	pool, _ := goroutinepool.NewPool(10)
	defer pool.Release()

	for i := 0; i < 100; i++ {
		taskNum := i // 注意闭包问题
		// 4. 提交任务到池中
		_ = pool.Submit(func() {
			fmt.Printf("执行任务 %d\n", taskNum)
			time.Sleep(100 * time.Millisecond)
			if i%3 == 0 {
				panic("模拟panic " + fmt.Sprintf("%d", i))
			}
		})
	}
	pool.Wait()
	eList := pool.ErrList()
	if len(eList) != 34 {
		t.Errorf("期望34个错误，实际%d个", len(eList))
	}
	for _, err := range eList {
		t.Logf("错误: %v", err)
	}
}

func TestGoroutinePoolFunc(t *testing.T) {
	runTask := func(i interface{}) {
		fmt.Printf("Running task: %d\n", i.(int))
		time.Sleep(1 * time.Second) // 模拟任务处理
	}

	// 创建一个具有 10 个 goroutines 的池
	p, _ := ants.NewPoolWithFunc(10, func(i interface{}) {
		runTask(i)
	})
	defer p.Release()

	// 提交任务
	for i := 0; i < 30; i++ {
		_ = p.Invoke(i)
	}

	// 等待所有任务完成
	p.Waiting()
	fmt.Println("All tasks completed")
}

func TestGoroutinePoolFunc1(t *testing.T) {
	poolFunc, _ := goroutinepool.NewFuncPool(10, func(i interface{}) {
		fmt.Printf("Running task: %d\n", i.(int))
		time.Sleep(10 * time.Millisecond) // 模拟任务处理
		if i.(int)%3 == 0 {
			panic("模拟panic " + fmt.Sprintf("%d", i))
		}
	})
	defer poolFunc.Release()

	// 提交任务
	for i := 0; i < 100; i++ {
		err := poolFunc.Invoke(i)
		if err != nil {
			t.Errorf("任务 %d 执行失败: %v", i, err)
		}
	}

	// 等待所有任务完成
	poolFunc.Waiting()
	eList := poolFunc.ErrList()
	if len(eList) != 34 {
		t.Errorf("期望34个错误，实际%d个", len(eList))
	}
	for _, err := range eList {
		t.Logf("错误: %v", err)
	}
}
