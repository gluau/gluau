package vm

/*
#include "../../rustlib/rustlib.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

var luaVmTab = objectTab{
	dtor: func(ptr *C.void) {
		C.freeluavm((*C.struct_LuaVmWrapper)(unsafe.Pointer(ptr)))
	},
}

// Internal VM wrapper
type GoLuaVmWrapper struct {
	*object
}

func (l *GoLuaVmWrapper) lua() (*C.struct_LuaVmWrapper, error) {
	ptr, err := l.object.Pointer()
	if err != nil {
		return nil, err // Return error if the object is closed
	}
	return (*C.struct_LuaVmWrapper)(unsafe.Pointer(ptr)), nil
}

func (l *GoLuaVmWrapper) SetMemoryLimit(limit int) error {
	lua, err := l.lua()
	if err != nil {
		return err
	}
	res := C.luavm_setmemorylimit(lua, C.size_t(limit))
	var result = GoResultFromC[bool](res)
	if result.Error != "" {
		return errors.New(result.Error)
	}
	return nil
}

func (l *GoLuaVmWrapper) CreateString(s []byte) GoResult[LuaString] {
	lua, err := l.lua()
	if err != nil {
		return GoResult[LuaString]{Error: err.Error()}
	}

	if len(s) == 0 {
		// Passing nil to luago_create_string creates an empty string.
		res := C.luago_create_string(lua, (*C.char)(nil), C.size_t(len(s)))
		var stringResult = GoResultFromC[C.void](res)
		return MapResult(stringResult, func(t *C.void) *LuaString {
			return &LuaString{object: NewObject(t, stringTab)}
		})
	}

	res := C.luago_create_string(lua, (*C.char)(unsafe.Pointer(&s[0])), C.size_t(len(s)))
	var stringResult = GoResultFromC[C.void](res)
	return MapResult(stringResult, func(t *C.void) *LuaString {
		return &LuaString{object: NewObject(t, stringTab)}
	})
}

func (l *GoLuaVmWrapper) CreateTable() GoResult[LuaTable] {
	lua, err := l.lua()
	if err != nil {
		return GoResult[LuaTable]{Error: err.Error()}
	}

	res := C.luago_create_table(lua)
	var tableResult = GoResultFromC[C.void](res)
	return MapResult(tableResult, func(t *C.void) *LuaTable {
		return &LuaTable{object: NewObject(t, tableTab)}
	})
}

func (l *GoLuaVmWrapper) CreateTableWithCapacity(narr, nrec int) GoResult[LuaTable] {
	lua, err := l.lua()
	if err != nil {
		return GoResult[LuaTable]{Error: err.Error()}
	}

	res := C.luago_create_table_with_capacity(lua, C.size_t(narr), C.size_t(nrec))
	var tableResult = GoResultFromC[C.void](res)
	return MapResult(tableResult, func(t *C.void) *LuaTable {
		return &LuaTable{object: NewObject(t, tableTab)}
	})
}

func (l *GoLuaVmWrapper) DebugValue() [3]Value {
	lua, err := l.lua()
	if err != nil {
		panic(err.Error()) // This should not happen in normal operation
	}

	v := C.luago_dbg_value(lua)
	values := [3]Value{}
	for i, v := range v.values {
		values[i] = ValueFromC(v)
	}
	return values
}

func CreateLuaVm() (*GoLuaVmWrapper, error) {
	ptr := C.newluavm()
	if ptr == nil {
		return nil, fmt.Errorf("failed to create Lua VM")
	}
	vm := &GoLuaVmWrapper{object: NewObject((*C.void)(unsafe.Pointer(ptr)), luaVmTab)}
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

func GoResultFromC[T any](ptr C.struct_GoResult) GoResult[T] {
	result := GoResult[T]{}

	if ptr.error != nil {
		result.Error = C.GoString(ptr.error)
		C.luago_result_error_free(ptr.error) // Free the error string
	} else if ptr.value != nil {
		// If there's no error, cast the generic `void*` to the specific `*T`.
		// This is the only place we need to use `unsafe` logic.
		result.Value = (*T)(unsafe.Pointer(ptr.value))
	}
	return result
}
