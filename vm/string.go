package vm

import (
	"unsafe"
)

/*
#include "../rustlib/rustlib.h"
*/
import "C"

var stringTab = objectTab{
	dtor: func(ptr *C.void) {
		C.luago_free_string((*C.struct_LuaString)(unsafe.Pointer(ptr)))
	},
}

// A LuaString is an abstraction over a Lua string object.
type LuaString struct {
	object *object
}

// Returns the LuaString as a byte slice
func (l *LuaString) Bytes() []byte {
	l.object.RLock()
	defer l.object.RUnlock()
	ptr, err := l.object.PointerNoLock()
	if err != nil {
		return nil // Return nil if the object is closed
	}
	data := C.luago_string_as_bytes((*C.struct_LuaString)(unsafe.Pointer(ptr)))
	goSlice := C.GoBytes(unsafe.Pointer(data.data), C.int(data.len))
	return goSlice
}

// Returns the LuaString as a byte slice with nul terminator
func (l *LuaString) BytesWithNul() []byte {
	l.object.RLock()
	defer l.object.RUnlock()
	ptr, err := l.object.PointerNoLock()
	if err != nil {
		return nil // Return nil if the object is closed
	}

	data := C.luago_string_as_bytes_with_nul((*C.struct_LuaString)(unsafe.Pointer(ptr)))
	goSlice := C.GoBytes(unsafe.Pointer(data.data), C.int(data.len))
	return goSlice
}

// Returns a 'pointer' to a LuaString
func (l *LuaString) Pointer() uint64 {
	l.object.RLock()
	defer l.object.RUnlock()
	lptr, err := l.object.PointerNoLock()
	if err != nil {
		return 0 // Return 0 if the object is closed
	}

	ptr := C.luago_string_to_pointer((*C.struct_LuaString)(unsafe.Pointer(lptr)))
	return uint64(ptr)
}

func (l *LuaString) String() string {
	// Convert the LuaString to a Go string
	return string(l.Bytes())
}

func (l *LuaString) Close() {
	if l == nil || l.object == nil {
		return // Nothing to close
	}
	// Close the LuaString object
	l.object.Close()
}
