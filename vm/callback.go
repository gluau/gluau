package vm

import (
	"runtime/cgo"
	"unsafe"
)

/*
#include "../rustlib/rustlib.h"

void goCallbackTrampoline(void* val, void* handle);
void goDropTrampoline(void* handle);
*/
import "C"

// goCallback is a struct that holds a Go function to be called from Rust.
type goCallback struct {
	// The actual Go function to call.
	// The go function accepts a single unsafe.Pointer argument
	// (which is the args from the Rust side)
	handle func(val unsafe.Pointer)
	// A drop function to clean up the callback.
	drop func()
	// cgo.Handle is used to safely pass Go functions to C.
	cgoHandle cgo.Handle
}

func newGoCallback(fn func(val unsafe.Pointer), ondrop func()) *goCallback {
	callback := &goCallback{
		handle: fn,
		drop:   ondrop,
	}
	cgoHandle := cgo.NewHandle(callback)
	callback.cgoHandle = cgoHandle
	return callback
}

//export goCallbackTrampoline
func goCallbackTrampoline(val unsafe.Pointer, handle unsafe.Pointer) {
	h := cgo.Handle(handle)
	if h == cgo.Handle(0) {
		return // Handle is invalid, nothing to do
	}
	callbackObj := h.Value().(*goCallback)
	callbackObj.handle(val)
}

//export goDropTrampoline
func goDropTrampoline(handle unsafe.Pointer) {
	h := cgo.Handle(handle)
	callbackObj := h.Value().(*goCallback)
	if callbackObj == nil || callbackObj.cgoHandle == cgo.Handle(0) {
		return // Handle is invalid, nothing to do
	}
	callbackObj.Drop()
	// Modify the handle pointer itself to prevent further use
}

func (cb *goCallback) Drop() {
	if cb == nil || cb.cgoHandle == cgo.Handle(0) {
		return
	}
	if cb.drop != nil {
		cb.drop() // Call the drop function if it exists
	}
	cb.cgoHandle.Delete()        // This will call the finalizer and clean up the callback
	cb.handle = nil              // Clear the handle to prevent further calls
	cb.drop = nil                // Clear the drop function
	cb.cgoHandle = cgo.Handle(0) // Mark the handle as invalid
}

func (cb *goCallback) ToC() C.struct_IGoCallback {
	if cb == nil || cb.cgoHandle == cgo.Handle(0) {
		panic("GoCallback is nil or already dropped")
	}
	// Create the IGoCallback struct to pass to Rust
	callback := C.struct_IGoCallback{
		callback: C.Callback(C.goCallbackTrampoline),
		drop:     C.DropCallback(C.goDropTrampoline),
		handle:   C.uintptr_t(cb.cgoHandle),
	}
	return callback
}
