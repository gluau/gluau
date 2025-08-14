package vm

import (
	"runtime"
	"unsafe"
)

/*
#include "../../rustlib/rustlib.h"
*/
import "C"

// A ErrorVariant is an wrapper around a Rust Arc<String> that holds an error string.
type ErrorVariant struct {
	ptr *C.void
}

// NewString creates a LuaString from a C pointer.
func NewErrorVariant(ptr *C.void) *ErrorVariant {
	if ptr == nil {
		return nil // Return nil if the pointer is nil
	}

	luaStr := &ErrorVariant{ptr: ptr}
	runtime.SetFinalizer(luaStr, (*ErrorVariant).Close) // Set finalizer to clean up LuaString
	return luaStr
}

// Returns the ErrorVariant as a byte slice
func (l *ErrorVariant) Bytes() []byte {
	if l.ptr == nil {
		return nil
	}

	data := C.luago_error_get_string((*C.struct_ErrorVariant)(unsafe.Pointer(l.ptr)))
	goSlice := C.GoBytes(unsafe.Pointer(data.data), C.int(data.len))
	return goSlice
}

func (l *ErrorVariant) Close() {
	if l.ptr == nil {
		return
	}

	C.luago_error_free((*C.struct_ErrorVariant)(unsafe.Pointer(l.ptr)))
	l.ptr = nil                  // Prevent double free
	runtime.SetFinalizer(l, nil) // Remove finalizer to prevent double calls
}
