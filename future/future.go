package future

import (
	"context"
	"sync"
	"time"
)

// Interface represents a future. No concrete implementation is
// exposed; all access to a future is via this interface.
type Interface interface {
	// Get returns the values calculated by the future. It will pause until
	// the value is calculated.
	//
	// If Get is invoked multiple times, the same value will be returned each time.
	// Subsequent calls to Get will return instantaneously.
	//
	// When the future is cancelled, nil is returned for both the value and the error.
	Get() (interface{}, error)

	// GetUntil waits for up to Duration d for the future to complete. If the
	// future completes before the Duration completes, the value and error are returned
	// and timeout is returned as false. If the Duration completes before the future
	// returns, nil is returned for the value and the error and timeout is returned
	// as true.
	//
	// When the future is cancelled, nil is returned for both the value and the error.
	GetUntil(d time.Duration) (interface{}, bool, error)

	// Then allows multiple function calls to be chained together into a
	// single future.
	//
	// Each call is run in order, with the output of the previous call passed into
	// the next function in the chain. If an error occurs at any step in the chain,
	// processing ceases and the error is returned via Get or GetUntil.
	Then(Step) Interface

	// Cancel prevents a future that hasn’t completed from returning a
	// value. Any current or future calls to Get or GetUntil will return
	// immediately.
	//
	// If the future has already completed or has already been
	// cancelled, calling Cancel will do nothing.
	// After a successful cancel, IsCancelled returns true.
	//
	// Calling Cancel on a future that has not completed does not stop the
	// currently running function. However, any chained functions will not
	// be run and the values returned by the current function are not accessible.
	Cancel()

	// IsCancelled indicates if a future terminated due to cancellation.
	// If Cancel was called and the future’s work was not completed, IsCancelled
	// returns true. Otherwise, it returns false
	IsCancelled() bool
}

type Step func(interface{}) (interface{}, error)

type Process func() (interface{}, error)

func NewWithContext(ctx context.Context, inFunc Process) Interface {
	f := New(inFunc).(*futureImpl)

	c := ctx.Done()

	go func() {
		select {
		case <-c:
			f.Cancel()
		case <-f.cancel:
		case <-f.done:
		}

	}()
	return f
}

func New(inFunc Process) Interface {
	return newInner(make(chan struct{}), &sync.Once{}, inFunc)
}

func newInner(cancelChan chan struct{}, o *sync.Once, inFunc Process) Interface {
	f := futureImpl{
		done:   make(chan struct{}),
		cancel: cancelChan,
		o:      o,
	}
	go func() {
		go func() {
			f.val, f.err = inFunc()
			close(f.done)
		}()
		select {
		case <-f.done:
			//do nothing, just waiting to see which will happen first
		case <-f.cancel:
			//do nothing, leave val and err nil
		}
	}()
	return &f
}

type futureImpl struct {
	done   chan struct{}
	cancel chan struct{}
	val    interface{}
	err    error
	o      *sync.Once
}

func (f *futureImpl) Get() (interface{}, error) {
	select {
	case <-f.done:
		return f.val, f.err
	case <-f.cancel:
		//on cancel, just fall out
	}
	return nil, nil
}

func (f *futureImpl) GetUntil(d time.Duration) (interface{}, bool, error) {
	select {
	case <-f.done:
		return f.val, false, f.err
	case <-time.After(d):
		return nil, true, nil
	case <-f.cancel:
		//on cancel, just fall out
	}
	return nil, false, nil
}

func (f *futureImpl) Then(next Step) Interface {
	nextFuture := newInner(f.cancel, f.o, func() (interface{}, error) {
		result, err := f.Get()
		if f.IsCancelled() || err != nil {
			return result, err
		}
		return next(result)
	})
	return nextFuture
}

func (f *futureImpl) Cancel() {
	select {
	case <-f.done:
	case <-f.cancel:
	default:
		f.o.Do(func() {
			close(f.cancel)
		})
	}
}
func (f *futureImpl) IsCancelled() bool {
	select {
	case <-f.cancel:
		return true
	default:
		return false
	}
}
