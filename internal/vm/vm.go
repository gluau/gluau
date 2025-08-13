package vm

/*
#cgo LDFLAGS: -L../../rustlib -lrustlib
#include "../../rustlib/rustlib.h"
*/
import "C"
import (
	"fmt"
	"runtime"
	"unsafe"
)

// Internal VM wrapper
type GoLuaVmWrapper struct {
	lua *C.struct_LuaVmWrapper
}

// A LuaString is an abstraction over a Lua string object.
type LuaString struct {
	ptr *C.void
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
//
// Note: this pointer is only useful for hashing and debugging and you cannot
// get back a LuaString from it.
func (l *LuaString) Pointer() uint64 {
	if l.ptr == nil {
		return 0
	}

	ptr := C.luago_string_to_pointer((*C.struct_LuaString)(unsafe.Pointer(l.ptr)))
	return uint64(ptr)
}

// Closes the Lua string object and frees its memory.
func (l *LuaString) Close() {
	if l.ptr == nil {
		return
	}

	C.luago_free_string((*C.struct_LuaString)(unsafe.Pointer(l.ptr)))
	l.ptr = nil                  // Prevent double free
	runtime.SetFinalizer(l, nil) // Remove finalizer to prevent double calls
}

func (l *GoLuaVmWrapper) CreateString(s []byte) GoResult[LuaString] {
	res := C.luago_create_string(l.lua, (*C.char)(unsafe.Pointer(&s[0])), C.size_t(len(s)))
	var stringResult = GoResultFromC[C.void](res)
	return MapResult(stringResult, func(l *C.void) *LuaString {
		luaStr := &LuaString{ptr: l}
		runtime.SetFinalizer(luaStr, (*LuaString).Close)
		return luaStr
	})
}

// Close cleans up the Lua VM
func (l *GoLuaVmWrapper) Close() {
	if l.lua == nil {
		return
	}

	fmt.Println("Closing Lua VM")

	C.freeluavm(l.lua)
	l.lua = nil                  // Prevent double free
	runtime.SetFinalizer(l, nil) // Remove finalizer to prevent double calls
}

func CreateLuaVm() (*GoLuaVmWrapper, error) {
	ptr := C.newluavm()
	if ptr == nil {
		return nil, fmt.Errorf("failed to create Lua VM")
	}
	vm := &GoLuaVmWrapper{lua: ptr}
	runtime.SetFinalizer(vm, (*GoLuaVmWrapper).Close)
	return vm, nil
}

// GoResult provides both the value and a error string returned by Lua.
//
// This is a internal structure and should *not* be used directly.
type GoResult[T any] struct {
	Value *T
	Error string
}

func MapResult[T any, U any](result GoResult[T], mapper func(*T) *U) GoResult[U] {
	if result.Error != "" {
		return GoResult[U]{Error: result.Error}
	}
	if result.Value == nil {
		return GoResult[U]{Value: nil}
	}
	mappedValue := mapper(result.Value)
	return GoResult[U]{Value: mappedValue}
}

func GoResultFromC[T any](ptr *C.struct_GoResult) GoResult[T] {
	if ptr == nil {
		return GoResult[T]{}
	}

	result := GoResult[T]{}

	if ptr.error != nil {
		result.Error = C.GoString(ptr.error)
	} else if ptr.value != nil {
		// If there's no error, cast the generic `void*` to the specific `*T`.
		// This is the only place we need to use `unsafe` logic.
		result.Value = (*T)(unsafe.Pointer(ptr.value))
	}

	// Deallocates everything but the Value pointer.
	//
	// The error is already copied at this stage anyways
	C.luago_result_free(ptr)
	return result
}
