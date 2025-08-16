package vm

import (
	"runtime/cgo"
	"unsafe"
)

/*
#include "../rustlib/rustlib.h"

void goCallbackTrampoline(void* val, void* handle);
void goDropTrampoline(void* handle);
void dynDropTrampoline(void* handle);
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

// dynamicData is a struct that holds an associated Go pointer
// for userdata
type dynamicData struct {
	// A drop function to clean up the callback.
	drop func()
	// The actual data value
	data any
	// cgo.Handle is used to safely pass Go functions to C.
	cgoHandle cgo.Handle
}

func newDynamicData(value any, ondrop func()) *dynamicData {
	dynData := &dynamicData{
		drop: ondrop,
		data: value,
	}
	cgoHandle := cgo.NewHandle(dynData)
	dynData.cgoHandle = cgoHandle
	return dynData
}

// Returns the dynamic data as a Go value.
func getDynamicData(handle uintptr) any {
	h := cgo.Handle(handle)
	if h == cgo.Handle(0) {
		return nil // Handle is invalid, nothing to do
	}
	dynData := h.Value().(*dynamicData)
	return dynData.data
}

//export dynDropTrampoline
func dynDropTrampoline(handle unsafe.Pointer) {
	h := cgo.Handle(handle)
	dynData := h.Value().(*dynamicData)
	if dynData == nil || dynData.cgoHandle == cgo.Handle(0) {
		return // Handle is invalid, nothing to do
	}
	dynData.Drop()
}

func (dyn *dynamicData) Drop() {
	if dyn == nil || dyn.cgoHandle == cgo.Handle(0) {
		return
	}
	if dyn.drop != nil {
		dyn.drop() // Call the drop function if it exists
	}
	dyn.cgoHandle.Delete()        // This will call the finalizer and clean up the callback
	dyn.data = nil                // Clear the data reference to allow it to be garbage collected
	dyn.drop = nil                // Clear the drop function
	dyn.cgoHandle = cgo.Handle(0) // Mark the handle as invalid
}

func (cb *dynamicData) ToC() C.struct_DynamicData {
	if cb == nil || cb.cgoHandle == cgo.Handle(0) {
		panic("dynamicData is nil or already dropped")
	}
	// Create the DynamicData struct to pass to Rust
	callback := C.struct_DynamicData{
		drop:   C.DropCallback(C.dynDropTrampoline),
		handle: C.uintptr_t(cb.cgoHandle),
	}
	return callback
}
