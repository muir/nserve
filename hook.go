package nserve

import (
	"sync"
	"sync/atomic"
)

var hookCounter int32

type hookOrder string

const (
	// ForwardOrder is used to indicate that the items
	// registered for a hook will be invoked in the order
	// that they were registered.
	ForwardOrder hookOrder = "forward"
	// ReverseOrder is used to indicate that the items
	// registered for a hook will be invoked opposite to the order
	// that they were registered.
	ReverseOrder = "forward"
)

type hookId int32

// Hook is the handle/name for a list of callbacks to invoke.
type Hook struct {
	Id            hookId
	lock          *sync.Mutex
	Name          string
	Order         hookOrder
	InvokeOnError []*Hook
	ContinuePast  bool
	ErrorCombiner func(first, second error) error
	Providers     []interface{}
}

// Copy makes a deep copy of a hook and the new hook gets a new Id.
// Copy is thread-safe.
func (h *Hook) Copy() *Hook {
	h.lock.Lock()
	defer h.lock.Unlock()
	oe := make([]*Hook, len(h.InvokeOnError))
	copy(oe, h.InvokeOnError)
	op := make([]interface{}, len(h.Providers))
	copy(op, h.Providers)
	hc := *h
	hc.InvokeOnError = oe
	hc.Id = hookId(atomic.AddInt32(&hookCounter, 1))
	hc.Providers = op
	hc.lock = new(sync.Mutex)
	return &hc
}

// NewHook creates a new category of callbacks.
func NewHook(name string, order hookOrder) *Hook {
	return &Hook{
		Id:    hookId(atomic.AddInt32(&hookCounter, 1)),
		Name:  name,
		Order: order,
		lock:  new(sync.Mutex),
	}
}

// OnError adds to the set of hooks to invoke when this hook is
// thows an error.  Call with nil to clear the set of hooks to invoke.
// OnError is thread-safe.
func (h *Hook) OnError(e *Hook) *Hook {
	h.lock.Lock()
	defer h.lock.Unlock()
	if e == nil {
		h.InvokeOnError = nil
	} else {
		h.InvokeOnError = append(h.InvokeOnError, e)
	}
	return h
}

// SetErrorCombiner sets a function to combine two errors into one when there
// is more than one error to return from a invoking all the callbacks
// SetErrorCombiner is thread-safe.
func (h *Hook) SetErrorCombiner(f func(first, second error) error) *Hook {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.ErrorCombiner = f
	return h
}

// ContinuePastError sets if callbacks should continue to be invoked
// if there has already been an error.
// ContinuePastError is thread-safe.
func (h *Hook) ContinuePastError(b bool) *Hook {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.ContinuePast = b
	return h
}

// String is not thread-safe with respect to reaching into a hook and
// changing it's Name.  Don't do that.
func (h *Hook) String() string {
	return "hook " + h.Name
}

// Shutdown is a reverse-order hook meant to be used for forced shutdowns.
// If Stop encounters an error, then Shutdown will also be called.
var Shutdown = NewHook("shutdown", ReverseOrder)

// Stop is a reverse-order hook meant to be used when stopping. If an error
// is encountered, Shutdown will also be used.
var Stop = NewHook("stop", ReverseOrder).OnError(Shutdown).ContinuePastError(true)

// Start is a forward-order hook for starting services. If it encounters
// an error, it will invoke Stop on whatever was started.
var Start = NewHook("start", ForwardOrder).OnError(Stop)
