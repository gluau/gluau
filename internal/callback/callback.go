package callback

import (
	"runtime/cgo"
	"unsafe"
)

/*
#include "../../rustlib/rustlib.h"

void GoCallbackTrampoline(void* val, void* handle);
void GoDropTrampoline(void* handle);
*/
import "C"

// GoCallback is a struct that holds a Go function to be called from Rust.
type GoCallback struct {
	// The actual Go function to call.
	// The go function accepts a single unsafe.Pointer argument
	// (which is the args from the Rust side)
	handle func(val unsafe.Pointer)
	// A drop function to clean up the callback.
	drop func()
	// cgo.Handle is used to safely pass Go functions to C.
	cgoHandle cgo.Handle
}

func NewGoCallback(fn func(val unsafe.Pointer), ondrop func()) *GoCallback {
	callback := &GoCallback{
		handle: fn,
		drop:   ondrop,
	}
	cgoHandle := cgo.NewHandle(callback)
	callback.cgoHandle = cgoHandle
	return callback
}

//export GoCallbackTrampoline
func GoCallbackTrampoline(val unsafe.Pointer, handle unsafe.Pointer) {
	h := cgo.Handle(handle)
	if h == cgo.Handle(0) {
		return // Handle is invalid, nothing to do
	}
	callbackObj := h.Value().(*GoCallback)
	callbackObj.handle(val)
}

//export GoDropTrampoline
func GoDropTrampoline(handle unsafe.Pointer) {
	h := cgo.Handle(handle)
	callbackObj := h.Value().(*GoCallback)
	if callbackObj == nil || callbackObj.cgoHandle == cgo.Handle(0) {
		return // Handle is invalid, nothing to do
	}
	callbackObj.Drop()
	// Modify the handle pointer itself to prevent further use
}

func (cb *GoCallback) Drop() {
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

func (cb *GoCallback) ToC() *C.struct_IGoCallback {
	if cb == nil || cb.cgoHandle == cgo.Handle(0) {
		panic("GoCallback is nil or already dropped")
	}
	// Create the IGoCallback struct to pass to Rust
	callback := C.struct_IGoCallback{
		callback: C.Callback(C.GoCallbackTrampoline),
		drop:     C.DropCallback(C.GoDropTrampoline),
		handle:   C.uintptr_t(cb.cgoHandle),
	}
	callbackPtr := &callback
	return callbackPtr
}
