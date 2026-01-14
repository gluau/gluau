package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"
import (
	"fmt"
	"runtime"
)

// A LuaFunction is an wrapper around a function
//
// API's to be implemented as of now: coverage, info (more complex to implement),
type LuaFunction struct {
	*BaseRef
}

// Call calls a function `f` returning either the returned arguments
// or the error
func (l *LuaFunction) Call(args ...any) ([]any, error) {
	return withBaseRef(l.BaseRef, func(ptr C.struct_GoLuaValueV2) ([]any, error) {
		mw, err := createMultiValue(args)
		defer freeMultiValueArray(mw)
		if err != nil {
			return nil, err // Return error if the value cannot be converted
		}

		res := C.luago_function_call(l.lua.ptr(), ptr, mw)
		runtime.KeepAlive(args) // ensure args are not GC'd before the call
		if res.error != nil {
			return nil, moveErrorToGo(res.error)
		}
		fmt.Println("Function call successful, copying return values")
		return copyAndFreeMultiValueArray(l.lua, res.value), nil
	})
}

// Returns a deep clone to a Lua-owned function
//
// If called on a Luau function, this method copies the function prototype and all its upvalues to the
// newly created function
//
// If called on a Go function, this method merely clones the function's handle
func (l *LuaFunction) DeepClone() (*LuaFunction, error) {
	return withBaseRef(l.BaseRef, func(ptr C.struct_GoLuaValueV2) (*LuaFunction, error) {
		val := C.luago_function_deepclone(l.lua.ptr(), ptr)
		if val.error != nil {
			err := moveErrorToGo(val.error)
			return nil, err
		}

		fn, ok := castValue(l.lua, val.value).(*LuaFunction)
		if !ok {
			return nil, fmt.Errorf("expected LuaFunction from deep clone, got %T", fn)
		}

		return fn, nil
	})
}

// Returns the environment table of the LuaFunction.
//
// If the function has no environment, it returns nil and Go functions will never have
// an environment table either.
func (l *LuaFunction) Environment() (*LuaTable, error) {
	return withBaseRef(l.BaseRef, func(ptr C.struct_GoLuaValueV2) (*LuaTable, error) {
		tabVal := C.luago_function_environment(l.lua.ptr(), ptr)
		tab, ok := castValue(l.lua, tabVal).(*LuaTable)
		if !ok {
			return nil, fmt.Errorf("expected LuaTable from function environment, got %T", tab)
		}

		return tab, nil
	})
}

// Sets the environment table of the LuaFunction returning true if the environment was set
func (l *LuaFunction) SetEnvironment(env *LuaTable) (bool, error) {
	if env.lua != l.lua {
		return false, fmt.Errorf("cannot set environment table from different Lua instance")
	}

	return withBaseRef(l.BaseRef, func(ptr C.struct_GoLuaValueV2) (bool, error) {
		res := C.luago_function_set_environment(l.lua.ptr(), ptr, env.BaseRef.value)
		if res.error != nil {
			err := moveErrorToGo(res.error)
			return false, err
		}

		return bool(res.value), nil
	})
}

// String returns a string representation of the LuaFunction.
//
// This is currently just the pointer address of the function.
func (l *LuaFunction) String() string {
	ptr := l.Pointer()
	if ptr == 0 {
		return "<closed LuaFunction>"
	}
	return fmt.Sprintf("LuaFunction 0x%x", ptr)
}
