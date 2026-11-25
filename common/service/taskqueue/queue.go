package taskqueue

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/panjf2000/ants/v2"
)

type queuedTask[T comparable] struct{ value T }

// Queue provides a bounded in-process queue and worker pool, deduplicating task
// IDs while queued or executing. Callers retain durable task state and claims;
// unstarted work is rediscovered from that state after restart.
type Queue[T comparable] struct {
	ctx     context.Context
	cancel  context.CancelFunc
	tasks   chan T
	pool    *ants.PoolWithFunc
	active  sync.WaitGroup
	done    chan struct{}
	pending sync.Map
}

func New[T comparable](ctx context.Context, concurrency, capacity int, handle func(context.Context, T)) (*Queue[T], error) {
	if concurrency <= 0 || capacity <= 0 || handle == nil {
		return nil, fmt.Errorf("任务并发数、队列容量必须大于 0，并提供处理方法")
	}
	workerCtx, cancel := context.WithCancel(ctx)
	q := &Queue[T]{ctx: workerCtx, cancel: cancel, tasks: make(chan T, capacity), done: make(chan struct{})}
	pool, err := ants.NewPoolWithFunc(concurrency, func(value any) {
		task := value.(queuedTask[T]).value
		defer q.active.Done()
		defer q.pending.Delete(task)
		if q.ctx.Err() == nil {
			handle(q.ctx, task)
		}
	}, ants.WithPanicHandler(func(cause any) {
		slog.Error("task queue worker panic", "panic", cause)
	}))
	if err != nil {
		cancel()
		return nil, err
	}
	q.pool = pool
	go q.run()
	return q, nil
}

// Push enqueues a task ID and blocks when the buffer is full. An already pending
// ID is accepted without another execution. False means the queue is stopping.
func (q *Queue[T]) Push(task T) bool {
	if q.ctx.Err() != nil {
		return false
	}
	if _, exists := q.pending.LoadOrStore(task, struct{}{}); exists {
		return q.ctx.Err() == nil
	}
	select {
	case <-q.ctx.Done():
		q.pending.Delete(task)
		return false
	case q.tasks <- task:
		if q.ctx.Err() != nil {
			q.pending.Delete(task)
			return false
		}
		return true
	}
}

// Close stops dispatch, cancels active handlers, and waits for their cleanup.
// Buffered tasks are not claimed or executed during shutdown.
func (q *Queue[T]) Close() {
	q.cancel()
	<-q.done
}

func (q *Queue[T]) run() {
	defer close(q.done)
	defer q.pool.Release()
	defer q.pending.Clear()
	defer q.active.Wait()
	for {
		if q.ctx.Err() != nil {
			return
		}
		select {
		case <-q.ctx.Done():
			return
		case task := <-q.tasks:
			if q.ctx.Err() != nil {
				return
			}
			q.active.Add(1)
			if err := q.pool.Invoke(queuedTask[T]{value: task}); err != nil {
				q.active.Done()
				q.pending.Delete(task)
				slog.Error("task queue dispatch", "task", task, "error", err)
			}
		}
	}
}
