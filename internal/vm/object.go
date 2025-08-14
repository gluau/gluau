package vm

import (
	"errors"
	"runtime"
	"sync"
)
import "C"

type objectTab struct {
	// dtor is the destructor function for the object
	// called on Close() or when finalizer is called
	dtor func(ptr *C.void)
}

// A object is a managed pointer to a Rust-owned handle
type object struct {
	mu  sync.Mutex // Ensure thread safety
	ptr *C.void
	tab objectTab
}

// NewObject creates a Object from a C pointer.
func NewObject(ptr *C.void, tab objectTab) *object {
	if ptr == nil {
		return nil // Return nil if the pointer is nil
	}

	obj := &object{ptr: ptr, tab: tab}
	runtime.SetFinalizer(obj, (*object).Close) // Set finalizer to clean up LuaString
	return obj
}

// Pointer returns the C pointer of the object.
func (o *object) Pointer() (*C.void, error) {
	// Only one Pointer() can run at a time
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.ptr == nil {
		return nil, errors.New("cannot use closed object")
	}

	return o.ptr, nil
}

// Close cleans up the Object by calling the destructor and setting the pointer to nil.
func (o *object) Close() {
	// Safety: Only one Close() can run at a time
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.ptr == nil {
		return
	}

	if o.tab.dtor != nil {
		o.tab.dtor(o.ptr) // Call the destructor if it exists
	}
	o.ptr = nil                  // Prevent double free
	runtime.SetFinalizer(o, nil) // Remove finalizer to prevent double calls
}
