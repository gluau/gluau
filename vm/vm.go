package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"
import (
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
	if res.error != nil {
		return moveErrorToGoError(res.error)
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
		if res.error != nil {
			return nil, moveErrorToGoError(res.error)
		}
		return &LuaString{object: newObject((*C.void)(unsafe.Pointer(res.value)), stringTab)}, nil
	}

	res := C.luago_create_string(lua, (*C.char)(unsafe.Pointer(&s[0])), C.size_t(len(s)))
	if res.error != nil {
		return nil, moveErrorToGoError(res.error)
	}
	return &LuaString{object: newObject((*C.void)(unsafe.Pointer(res.value)), stringTab)}, nil
}

// Create string as pointer (without any finalizer)
func (l *GoLuaVmWrapper) createStringAsPtr(s []byte) (*C.struct_LuaString, error) {
	l.obj.RLock()
	defer l.obj.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	if len(s) == 0 {
		// Passing nil to luago_create_string creates an empty string.
		res := C.luago_create_string(lua, (*C.char)(nil), C.size_t(len(s)))
		if res.error != nil {
			return nil, moveErrorToGoError(res.error)
		}
		return res.value, nil
	}

	res := C.luago_create_string(lua, (*C.char)(unsafe.Pointer(&s[0])), C.size_t(len(s)))
	if res.error != nil {
		return nil, moveErrorToGoError(res.error)
	}
	return res.value, nil
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
	if res.error != nil {
		return nil, moveErrorToGoError(res.error)
	}
	return &LuaTable{object: newObject((*C.void)(unsafe.Pointer(res.value)), tableTab), lua: l}, nil
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
	if res.error != nil {
		return nil, moveErrorToGoError(res.error)
	}
	return &LuaTable{object: newObject((*C.void)(unsafe.Pointer(res.value)), tableTab), lua: l}, nil
}

// CreateErrorVariant creates a new ErrorVariant from a byte slice.
func CreateErrorVariant(s []byte) *ErrorVariant {
	if len(s) == 0 {
		// Passing nil to luago_create_string creates an empty string.
		res := C.luago_error_new((*C.char)(nil), C.size_t(len(s)))
		return &ErrorVariant{object: newObject((*C.void)(unsafe.Pointer(res)), errorVariantTab)}
	}

	res := C.luago_error_new((*C.char)(unsafe.Pointer(&s[0])), C.size_t(len(s)))
	return &ErrorVariant{object: newObject((*C.void)(unsafe.Pointer(res)), errorVariantTab)}
}

type FunctionFn = func(funcVm *GoLuaVmWrapper, args []Value) ([]Value, error)

// CreateFunction creates a new Function
//
// Note that funcVm will only be open until the callback function returns
func (l *GoLuaVmWrapper) CreateFunction(callback FunctionFn) (*LuaFunction, error) {
	l.obj.RLock()
	defer l.obj.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	cbWrapper := newGoCallback(func(val unsafe.Pointer) {
		cval := (*C.struct_FunctionCallbackData)(val)

		// Safety: it is undefined behavior for the callback to unwind into
		// Rust (or even C!) frames from Go, so we must recover() any panic
		// that occurs in the callback to prevent a crash.
		defer func() {
			if r := recover(); r != nil {
				// Deallocate any existing error
				if cval.error != nil {
					C.luago_error_free(cval.error)
				}

				// Replace
				errBytes := []byte(fmt.Sprintf("panic in ForEachValue callback: %v", r))
				errv := C.luago_error_new((*C.char)(unsafe.Pointer(&errBytes[0])), C.size_t(len(errBytes)))
				cval.error = errv // Rust side will deallocate it for us
			}
		}()

		// Take out args
		mw := &luaMultiValue{ptr: cval.args, lua: l}
		args := mw.take()
		mw.close()

		callbackVm := &GoLuaVmWrapper{obj: newObject((*C.void)(unsafe.Pointer(cval.lua)), luaVmTab)}
		values, err := callback(callbackVm, args)
		defer callbackVm.Close() // Free the memory associated with the callback VM

		if err != nil {
			errBytes := []byte(err.Error())
			errv := C.luago_error_new((*C.char)(unsafe.Pointer(&errBytes[0])), C.size_t(len(errBytes)))
			cval.error = errv // Rust side will deallocate it for us
			return
		}

		outMw, err := l.multiValueFromValues(values)
		if err != nil {
			errBytes := []byte(err.Error())
			errv := C.luago_error_new((*C.char)(unsafe.Pointer(&errBytes[0])), C.size_t(len(errBytes)))
			cval.error = errv // Rust side will deallocate it for us
			return
		}

		cval.values = outMw.ptr // Rust will deallocate values as well
	}, func() {
		fmt.Println("function callback is being dropped")
	})

	res := C.luago_create_function(lua, cbWrapper.ToC())

	return &LuaFunction{object: newObject((*C.void)(unsafe.Pointer(res.value)), functionTab), lua: l}, nil
}

func (l *GoLuaVmWrapper) DebugValue() [4]Value {
	l.obj.RLock()
	defer l.obj.RUnlock()

	lua, err := l.lua()
	if err != nil {
		panic(err.Error()) // This should not happen in normal operation
	}

	v := C.luago_dbg_value(lua)
	values := [4]Value{}
	for i, v := range v.values {
		values[i] = l.valueFromC(v)
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
