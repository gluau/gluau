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
	sync.RWMutex // read = anything that doesnt close the object, write = Close()
	ptr          *C.void
	closed       bool
	tab          objectTab
}

// NewObject creates a Object from a C pointer.
func newObject(ptr *C.void, tab objectTab) *object {
	if ptr == nil {
		return nil // Return nil if the pointer is nil
	}

	obj := &object{ptr: ptr, tab: tab}
	runtime.SetFinalizer(obj, (*object).Close) // Set finalizer to clean up LuaString
	return obj
}

// PointerLock returns the C pointer of the object after
// acquiring a read lock. Use this when you need to ensure
func (o *object) PointerLock() (*C.void, error) {
	// Pointer can be read concurrently as long as Close() is not called
	o.RWMutex.RLock()
	defer o.RWMutex.RUnlock()

	if o.ptr == nil || o.closed {
		return nil, errors.New("cannot use closed object")
	}

	return o.ptr, nil
}

// PointerNoLock returns the C pointer of the object
// without acquiring the read lock. Use with caution.
func (o *object) PointerNoLock() (*C.void, error) {
	if o.ptr == nil || o.closed {
		return nil, errors.New("cannot use closed object")
	}

	return o.ptr, nil
}

// UnsafePointer returns the pointer without any locks
// and will not check if the object is closed or not (besides nil check).
func (o *object) UnsafePointer() (*C.void, error) {
	if o.ptr == nil {
		return nil, errors.New("cannot use closed object")
	}
	return o.ptr, nil
}

// Returns if the object is closed or not
func (o *object) IsClosed() bool {
	o.RLock()
	defer o.RUnlock()
	return o.closed
}

// Close cleans up the Object by calling the destructor and setting the pointer to nil.
func (o *object) Close() error {
	// Safety: Close() can only be called if no one is reading/using the object.
	ok := o.RWMutex.TryLock()
	if !ok {
		return errors.New("recursive lock detected, cannot close object")
	}
	defer o.RWMutex.Unlock()

	if o.ptr == nil {
		return nil
	}

	if o.tab.dtor != nil {
		o.tab.dtor(o.ptr) // Call the destructor if it exists
	}

	//fmt.Println("Object closed, pointer freed")
	o.ptr = nil                  // Prevent double free
	o.closed = true              // Mark as closed
	runtime.SetFinalizer(o, nil) // Remove finalizer to prevent double calls
	return nil
}

// Disarm disarms the object by setting the pointer to nil
// without calling the destructor.
func (o *object) Disarm() error {
	ok := o.RWMutex.TryLock()
	if !ok {
		return errors.New("recursive lock detected, cannot disarm object")
	}
	defer o.RWMutex.Unlock()

	return o.DisarmNoLock()
}

// DisarmNoLock disarms the object by closing the object
// without calling the destructor. Use with caution.
//
// Note that the original pointer will not be nilled as that is the job of the caller.
func (o *object) DisarmNoLock() error {
	if o == nil || o.ptr == nil || o.closed {
		return errors.New("cannot disarm closed object")
	}

	o.closed = true
	runtime.SetFinalizer(o, nil) // Remove finalizer to prevent double calls
	return nil
}
