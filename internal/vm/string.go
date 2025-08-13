package vm

import (
	"runtime"
	"unsafe"
)

/*
#cgo LDFLAGS: -L../../rustlib -lrustlib
#include "../../rustlib/rustlib.h"
*/
import "C"

// A LuaString is an abstraction over a Lua string object.
type LuaString struct {
	ptr *C.void
}

// NewString creates a LuaString from a C pointer.
func NewString(ptr *C.void) *LuaString {
	if ptr == nil {
		return nil // Return nil if the pointer is nil
	}

	luaStr := &LuaString{ptr: ptr}
	runtime.SetFinalizer(luaStr, (*LuaString).Close) // Set finalizer to clean up LuaString
	return luaStr
}

// Returns the LuaString as a byte slice
func (l *LuaString) Bytes() []byte {
	if l.ptr == nil {
		return nil
	}

	data := C.luago_string_as_bytes((*C.struct_LuaString)(unsafe.Pointer(l.ptr)))
	goSlice := C.GoBytes(unsafe.Pointer(data.data), C.int(data.len))
	return goSlice
}

// Returns the LuaString as a byte slice with nul terminator
func (l *LuaString) BytesWithNul() []byte {
	if l.ptr == nil {
		return nil
	}

	data := C.luago_string_as_bytes_with_nul((*C.struct_LuaString)(unsafe.Pointer(l.ptr)))
	goSlice := C.GoBytes(unsafe.Pointer(data.data), C.int(data.len))
	return goSlice
}

// Returns a 'pointer' to a LuaString
func (l *LuaString) Pointer() uint64 {
	if l.ptr == nil {
		return 0
	}

	ptr := C.luago_string_to_pointer((*C.struct_LuaString)(unsafe.Pointer(l.ptr)))
	return uint64(ptr)
}

func (l *LuaString) Close() {
	if l.ptr == nil {
		return
	}

	C.luago_free_string((*C.struct_LuaString)(unsafe.Pointer(l.ptr)))
	l.ptr = nil                  // Prevent double free
	runtime.SetFinalizer(l, nil) // Remove finalizer to prevent double calls
}
