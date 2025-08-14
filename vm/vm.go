package vm

/*
#include "../rustlib/rustlib.h"
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
	obj *object
}

func (l *GoLuaVmWrapper) lua() (*C.struct_LuaVmWrapper, error) {
	ptr, err := l.obj.PointerNoLock()
	if err != nil {
		return nil, err // Return error if the object is closed
	}
	return (*C.struct_LuaVmWrapper)(unsafe.Pointer(ptr)), nil
}

// SetMemoryLimit sets the memory limit for the Lua VM.
//
// Upon exceeding this limit, Luau will return a memory error
// back to the caller (which may either be in Luau still or in Go
// as a error value).
func (l *GoLuaVmWrapper) SetMemoryLimit(limit int) error {
	l.obj.RLock()
	defer l.obj.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return err
	}
	res := C.luavm_setmemorylimit(lua, C.size_t(limit))
	var result = goResultFromC[bool](res)
	if result.Error != "" {
		return errors.New(result.Error)
	}
	return nil
}

// CreateString creates a Lua string from a Go string.
func (l *GoLuaVmWrapper) CreateString(s string) (*LuaString, error) {
	return l.createString([]byte(s))
}

// CreateStringBytes creates a Lua string from a byte slice.
// This is useful for creating strings from raw byte data.
func (l *GoLuaVmWrapper) CreateStringBytes(s []byte) (*LuaString, error) {
	return l.createString(s)
}

func (l *GoLuaVmWrapper) createString(s []byte) (*LuaString, error) {
	l.obj.RLock()
	defer l.obj.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	if len(s) == 0 {
		// Passing nil to luago_create_string creates an empty string.
		res := C.luago_create_string(lua, (*C.char)(nil), C.size_t(len(s)))
		var stringResult = goResultFromC[C.void](res)
		if stringResult.Error != "" {
			return nil, errors.New(stringResult.Error)
		} else if stringResult.Value == nil {
			return nil, errors.New("failed to create Lua string")
		} else {
			return &LuaString{object: newObject(stringResult.Value, stringTab)}, nil
		}
	}

	res := C.luago_create_string(lua, (*C.char)(unsafe.Pointer(&s[0])), C.size_t(len(s)))
	var stringResult = goResultFromC[C.void](res)
	if stringResult.Error != "" {
		return nil, errors.New(stringResult.Error)
	} else if stringResult.Value == nil {
		return nil, errors.New("failed to create Lua string")
	} else {
		return &LuaString{object: newObject(stringResult.Value, stringTab)}, nil
	}
}

// CreateTable creates a new Lua table.
func (l *GoLuaVmWrapper) CreateTable() (*LuaTable, error) {
	l.obj.RLock()
	defer l.obj.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	res := C.luago_create_table(lua)
	var tableResult = goResultFromC[C.void](res)
	if tableResult.Error != "" {
		return nil, errors.New(tableResult.Error)
	} else if tableResult.Value == nil {
		return nil, errors.New("failed to create Lua table")
	} else {
		return &LuaTable{object: newObject(tableResult.Value, tableTab)}, nil
	}
}

// CreateTableWithCapacity creates a new Lua table with specified capacity for array and record parts.
// with narr as the number of array elements and nrec as the number of record elements.
func (l *GoLuaVmWrapper) CreateTableWithCapacity(narr, nrec int) (*LuaTable, error) {
	l.obj.RLock()
	defer l.obj.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	res := C.luago_create_table_with_capacity(lua, C.size_t(narr), C.size_t(nrec))
	var tableResult = goResultFromC[C.void](res)
	if tableResult.Error != "" {
		return nil, errors.New(tableResult.Error)
	} else if tableResult.Value == nil {
		return nil, errors.New("failed to create Lua table with capacity")
	} else {
		return &LuaTable{object: newObject(tableResult.Value, tableTab)}, nil
	}
}

func (l *GoLuaVmWrapper) DebugValue() [3]Value {
	l.obj.RLock()
	defer l.obj.RUnlock()

	lua, err := l.lua()
	if err != nil {
		panic(err.Error()) // This should not happen in normal operation
	}

	v := C.luago_dbg_value(lua)
	values := [3]Value{}
	for i, v := range v.values {
		values[i] = valueFromC(v)
	}
	return values
}

func (l *GoLuaVmWrapper) Close() {
	if l == nil || l.obj == nil {
		return // Nothing to close
	}

	// Close the Lua VM object
	l.obj.Close()
}

func CreateLuaVm() (*GoLuaVmWrapper, error) {
	ptr := C.newluavm()
	if ptr == nil {
		return nil, fmt.Errorf("failed to create Lua VM")
	}
	vm := &GoLuaVmWrapper{obj: newObject((*C.void)(unsafe.Pointer(ptr)), luaVmTab)}
	return vm, nil
}

// GoResult provides both the value and a error string returned by Lua.
//
// This is a internal structure and should *not* be used directly.
type goResult[T any] struct {
	Value *T
	Error string
}

func goResultFromC[T any](ptr C.struct_GoResult) goResult[T] {
	result := goResult[T]{}

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
