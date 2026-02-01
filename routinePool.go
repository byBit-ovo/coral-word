package main

import (
	"context"
	"sync"
)

type Task func(ctx context.Context)


// type GoRoutinePool struct{
// 	count int
// 	wg sync.WaitGroup
// 	tasks chan Task
// }

type GoRoutinePool struct {
	ctx    context.Context
	cancel context.CancelFunc

	tasks chan Task
	wg    sync.WaitGroup
}

func NewPool(workerCount int, queueSize int) *GoRoutinePool {
	cx, cl := context.WithCancel(context.Background())

	p := &GoRoutinePool{
		ctx:    cx,
		cancel: cl,
		tasks:  make(chan Task, queueSize),
	}

	for i := 0; i < workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	return p
}


func (p *GoRoutinePool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
			// 如果上下文被取消，则退出
		case <-p.ctx.Done():
			return
			// 如果任务通道被关闭，则退出,否则执行任务
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			task(p.ctx)
		}
	}
}

func (p *GoRoutinePool) Submit(task Task) bool {
	select {
	case p.tasks <- task:
		return true
	case <-p.ctx.Done():
		return false
	}
}

func (p *GoRoutinePool) Shutdown() {
	p.cancel()
	close(p.tasks)
	p.wg.Wait()
}


