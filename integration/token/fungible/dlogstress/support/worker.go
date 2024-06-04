/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package support

import (
	"sync"
)

type Task func()

type Pool struct {
	taskQueue chan Task
	wg        sync.WaitGroup
	shutdown  chan struct{}
}

func NewPool(numWorkers int) *Pool {
	pool := &Pool{
		taskQueue: make(chan Task, numWorkers),
		wg:        sync.WaitGroup{},
	}

	pool.wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer pool.wg.Done()
			for {
				if task, ok := <-pool.taskQueue; ok {
					task()
				} else {
					return
				}
			}
		}()
	}
	return pool
}

func (p *Pool) ScheduleTask(task Task) {
	for {
		select {
		case <-p.shutdown:
			return
		default:
			p.taskQueue <- task
		}
	}
}

func (p *Pool) Shutdown() {
	close(p.taskQueue)
	p.wg.Wait()
}
