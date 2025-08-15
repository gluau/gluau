package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"
import "unsafe"

var functionTab = objectTab{
	dtor: func(ptr *C.void) {
		C.luago_free_function((*C.struct_LuaFunction)(unsafe.Pointer(ptr)))
	},
}

// A LuaFunction is an wrapper around a function
type LuaFunction struct {
	object *object
	lua    *GoLuaVmWrapper
}

func (l *LuaFunction) innerPtr() (*C.struct_LuaFunction, error) {
	ptr, err := l.object.PointerNoLock()
	if err != nil {
		return nil, err // Return error if the object is closed
	}
	return (*C.struct_LuaFunction)(unsafe.Pointer(ptr)), nil
}

// Call calls a function `f` returning either the returned arguments
// or the error
func (l *LuaFunction) Call(args []Value) ([]Value, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return nil, err // Return error if the object is closed
	}
	mw, err := l.lua.multiValueFromValues(args)
	if err != nil {
		return nil, err // Return error if the value cannot be converted
	}

	res := C.luago_function_call(ptr, mw.ptr)
	if res.error != nil {
		return nil, moveErrorToGoError(res.error)
	}
	rets := &luaMultiValue{ptr: res.value}
	retsMw := rets.take()
	rets.close()
	return retsMw, nil
}

func (l *LuaFunction) Close() {
	if l == nil || l.object == nil {
		return // Nothing to close
	}
	// Close the LuaFunction object
	l.object.Close()
}
