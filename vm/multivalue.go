package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"

// luaMultiValue is an abstraction over multiple Lua values
//
// These values are initially owned by the Rust/Lua layer with
// the luaMultiValue acting as a barrier between the Rust/Lua layer
// and the Go side.
//
// Internal API
type luaMultiValue struct {
	lua *GoLuaVmWrapper
	ptr *C.struct_GoMultiValue
}

// Creates a new empty LuaMultiValue object.
func (l *GoLuaVmWrapper) newMultiValueWithCapacity(cap uint64) *luaMultiValue {
	ptr := C.luago_create_multivalue_with_capacity(C.size_t(cap))
	if ptr == nil {
		return nil // Handle error if needed
	}
	mv := &luaMultiValue{
		ptr: ptr,
		lua: l,
	}
	return mv
}

// Add a Lua value to the luaMultiValue.
func (l *luaMultiValue) push(value Value) error {
	luaValue, err := l.lua.valueToC(value)
	if err != nil {
		return err // Return error if the value cannot be converted
	}

	C.luago_multivalue_push(l.ptr, luaValue)
	return nil
}

// Pop a Lua value from the luaMultiValue.
//
// This pops the first value in the multivalue, not the last one.
func (l *luaMultiValue) pop() Value {
	cValue := C.luago_multivalue_pop(l.ptr)
	return l.lua.valueFromC(cValue)
}

// Returns the number of values in the luaMultiValue.
func (l *luaMultiValue) len() uint64 {
	return uint64(C.luago_multivalue_len(l.ptr))
}

// fromValues takes a []Value and makes a MultiValue
func (l *GoLuaVmWrapper) multiValueFromValues(values []Value) (*luaMultiValue, error) {
	mw := l.newMultiValueWithCapacity(uint64(len(values)))
	for _, v := range values {
		err := mw.push(v)
		if err != nil {
			mw.close()
			return nil, err
		}
	}

	return mw, nil
}

// takes a MultiValue and outputs a []Value
func (l *luaMultiValue) take() []Value {
	len := l.len()
	values := make([]Value, 0, len)
	var i uint64
	for i = 0; i < len; i++ {
		values = append(values, l.pop())
	}
	return values
}

func (l *luaMultiValue) close() {
	C.luago_free_multivalue(l.ptr)
	l.ptr = nil
}
