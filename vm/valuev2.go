package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"
	"unsafe"
)

// Returns the null C GoLuaValueV2
func nullCValueV2() C.struct_GoLuaValueV2 {
	var cVal C.struct_GoLuaValueV2
	cVal.tag = C.LuaValueTypeV2Nil
	return cVal
}

// castValue converts a C GoLuaValueV2 to a Go value.
//
// The returned type is guaranteed to clean up after itself when it goes out of scope.
func castValue(lua *Lua, value C.struct_GoLuaValueV2) any {
	switch value.tag {
	case C.LuaValueTypeV2Nil:
		return nil
	case C.LuaValueTypeV2Boolean:
		val := *(*bool)(unsafe.Pointer(&value.data))
		return val
	case C.LuaValueTypeV2LightUserData:
		val := *(*unsafe.Pointer)(unsafe.Pointer(&value.data))
		return LightUserData{value: val}
	case C.LuaValueTypeV2Integer:
		val := *(*int64)(unsafe.Pointer(&value.data))
		return val
	case C.LuaValueTypeV2Number:
		val := *(*float64)(unsafe.Pointer(&value.data))
		return val
	case C.LuaValueTypeV2Vector:
		vec := *(*[3]C.float)(unsafe.Pointer(&value.data))
		return [3]float32{float32(vec[0]), float32(vec[1]), float32(vec[2])}
	case C.LuaValueTypeV2String:
		return &LuaString{BaseRef: newBaseRef(lua, value)}
	case C.LuaValueTypeV2Function:
		return &LuaFunction{BaseRef: newBaseRef(lua, value)}
	case C.LuaValueTypeV2Table:
		return &LuaTable{BaseRef: newBaseRef(lua, value)}
	case C.LuaValueTypeV2Thread:
		return &LuaThread{BaseRef: newBaseRef(lua, value)}
	case C.LuaValueTypeV2UserData:
		return &LuaUserData{BaseRef: newBaseRef(lua, value)}
	case C.LuaValueTypeV2Buffer:
		return &LuaBuffer{BaseRef: newBaseRef(lua, value)}
	default:
		// Unknown type, return as Other
		return &OtherValue{BaseRef: newBaseRef(lua, value)}
	}
}

// Given a Go value, create a C GoLuaValueV2
//
// The underlying value should still be kept alive using runtime.KeepAlive() etc until the C call
// is finished.
func valueToC(lua *Lua, val any) (C.struct_GoLuaValueV2, error, *BaseRef) {
	var cVal C.struct_GoLuaValueV2
	switch v := val.(type) {
	case nil:
		cVal.tag = C.LuaValueTypeV2Nil
	case bool:
		cVal.tag = C.LuaValueTypeV2Boolean
		*(*C.bool)(unsafe.Pointer(&cVal.data)) = C.bool(v)
	case LightUserData:
		cVal.tag = C.LuaValueTypeV2LightUserData
		*(*unsafe.Pointer)(unsafe.Pointer(&cVal.data)) = v.value
	case int64:
		cVal.tag = C.LuaValueTypeV2Integer
		*(*int64)(unsafe.Pointer(&cVal.data)) = v
	case int:
		cVal.tag = C.LuaValueTypeV2Integer
		*(*int64)(unsafe.Pointer(&cVal.data)) = int64(v)
	case float64:
		cVal.tag = C.LuaValueTypeV2Number
		*(*float64)(unsafe.Pointer(&cVal.data)) = v
	case [3]float32:
		cVal.tag = C.LuaValueTypeV2Vector
		*(*[3]C.float)(unsafe.Pointer(&cVal.data)) = [3]C.float{C.float(v[0]), C.float(v[1]), C.float(v[2])}
	case string:
		// Create a LuaString for the string
		ls, err := lua.CreateString(v)
		if err != nil {
			return C.struct_GoLuaValueV2{}, fmt.Errorf("cannot convert string to LuaString: %w", err), nil
		}
		return baseRefToC(lua, ls.BaseRef)
	case *LuaString:
		return baseRefToC(lua, v.BaseRef)
	case *LuaFunction:
		return baseRefToC(lua, v.BaseRef)
	case *LuaTable:
		return baseRefToC(lua, v.BaseRef)
	case *LuaThread:
		return baseRefToC(lua, v.BaseRef)
	case *LuaUserData:
		return baseRefToC(lua, v.BaseRef)
	case *LuaBuffer:
		return baseRefToC(lua, v.BaseRef)
	case *OtherValue:
		return baseRefToC(lua, v.BaseRef)
	default:
		return C.struct_GoLuaValueV2{}, fmt.Errorf("cannot convert Go value of type %T to GoLuaValueV2", val), nil
	}
	return cVal, nil, nil
}

// StringifyValue converts a Go value to its string representation
func StringifyValue(value any) string {
	switch v := value.(type) {
	case nil:
		return "nil"
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int64:
		return fmt.Sprintf("%d", v)
	case int:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%f", v)
	case [3]float32:
		return fmt.Sprintf("vector(%f, %f, %f)", v[0], v[1], v[2])
	case *LuaString:
		return v.String()
	case *LuaFunction:
		return v.String()
	case *LuaTable:
		return v.String()
	case *LuaThread:
		return v.String()
	case *LuaUserData:
		return v.String()
	case *LuaBuffer:
		return v.String()
	case *OtherValue:
		return v.String()
	default:
		return fmt.Sprintf("<unknown type %T>", value)
	}
}

// Creates a GoLuaValueV2Array from a slice of Go values
func createMultiValue(lua *Lua, values []any) (C.struct_GoLuaValueV2Array, error) {
	size := len(values)
	cArray := C.luago_valuev2array_alloc(C.size_t(size))

	// Fill the array
	cSlice := unsafe.Slice(cArray.values, size)
	for i, val := range values {
		cVal, err, m := valueToC(lua, val)
		if m != nil {
			if m.closed.Load() {
				//fmt.Println("Error: cannot use closed argument in function call")
				C.luago_valuev2array_destroy(cArray)
				return C.struct_GoLuaValueV2Array{}, errors.New("cannot use closed argument in function call")
			}
		}
		if err != nil {
			//fmt.Println("Error converting value to C GoLuaValueV2:", err)
			C.luago_valuev2array_destroy(cArray)
			return C.struct_GoLuaValueV2Array{}, err
		}
		cSlice[i] = cVal
	}

	return cArray, nil
}

func freeMultiValueArray(arr C.struct_GoLuaValueV2Array) {
	// Free the array itself
	C.luago_valuev2array_destroy(arr)
}

// Safety:
// 1. Rust owns the values in 'arr'.
// 2. castValue attaches finalizers to each value
// 3. Rust's destroy function does not free individual values, only the array container
func copyAndFreeMultiValueArray(lua *Lua, arr C.struct_GoLuaValueV2Array) []any {
	values := make([]any, arr.size)
	cSlice := unsafe.Slice(arr.values, arr.size)
	for i := 0; i < int(arr.size); i++ {
		values[i] = castValue(lua, cSlice[i])
	}
	C.luago_valuev2array_destroy(arr)
	return values
}

// Casts a BaseRef to a GoLuaValueV2
func baseRefToC(lua *Lua, bv *BaseRef) (C.struct_GoLuaValueV2, error, *BaseRef) {
	if bv == nil || bv.closed.Load() {
		return C.struct_GoLuaValueV2{}, errors.New("cannot convert nil LuaTable to GoLuaValueV2"), nil
	}

	if !lua.IsSameVm(bv.lua) {
		return C.struct_GoLuaValueV2{}, errors.New("cannot convert LuaTable from different Lua VM to GoLuaValueV2"), nil
	}
	if bv.lua.IsClosed() {
		return C.struct_GoLuaValueV2{}, errors.New("cannot convert LuaTable from closed Lua VM to GoLuaValueV2"), nil
	}
	return bv.value, nil, bv
}

// A base reference type holding onto a GoLuaValueV2 that is a reference type
type BaseRef struct {
	value  C.struct_GoLuaValueV2
	closed atomic.Bool
	lua    *Lua // The Lua VM wrapper that owns this buffer
}

func newBaseRef(lua *Lua, value C.struct_GoLuaValueV2) *BaseRef {
	obj := &BaseRef{lua: lua, value: value}
	runtime.SetFinalizer(obj, (*BaseRef).Close) // Set finalizer to clean up LuaString
	return obj
}

// withBaseRef() calls a callback function with the underlying GoLuaValueV2
func withBaseRef[T any](bv *BaseRef, fn func(ptr C.struct_GoLuaValueV2) (T, error)) (T, error) {
	if bv.lua.IsClosed() {
		var zero T
		return zero, errors.New("cannot use closed Lua VM")
	}

	if bv.closed.Load() {
		var zero T
		return zero, errors.New("cannot use closed object")
	}

	return fn(bv.value)
}

// withBaseRef() calls a callback function with the underlying GoLuaValueV2
func withBaseRefNoRet(bv *BaseRef, fn func(ptr C.struct_GoLuaValueV2) error) error {

	if bv.lua.IsClosed() {
		return errors.New("cannot use closed Lua VM")
	}

	if bv.closed.Load() {
		return errors.New("cannot use closed object")
	}

	return fn(bv.value)
}

// withBaseRef() calls a callback function with the underlying GoLuaValueV2
func withBaseRefDefault[T any](bv *BaseRef, fn func(ptr C.struct_GoLuaValueV2) T) T {

	if bv.lua.IsClosed() {
		var zero T
		return zero
	}

	if bv.closed.Load() {
		var zero T
		return zero
	}

	return fn(bv.value)
}

// Returns a 'pointer' to a Lua-owned value
//
// This pointer is only useful for hashing/debugging
// and cannot be converted back to the original Lua value.
func (bv *BaseRef) Pointer() uint64 {

	if bv.closed.Load() {
		return 0
	}

	return uint64(C.luago_valuev2_topointer(bv.lua.ptr(), bv.value))
}

// Equals checks if the value equals another value by lua value reference
//
// If the values are userdata or tables, the values are first compared by reference.
// Otherwise, the __eq metamethod may be called to compare the two tables.
func (bv *BaseRef) Equals(v any) (bool, error) {
	if bv == nil || v == nil {
		return false, nil
	}

	var other *BaseRef
	switch o := v.(type) {
	case *LuaString:
		other = o.BaseRef
	case *LuaFunction:
		other = o.BaseRef
	case *LuaTable:
		other = o.BaseRef
	case *LuaThread:
		other = o.BaseRef
	case *LuaUserData:
		other = o.BaseRef
	case *LuaBuffer:
		other = o.BaseRef
	case *OtherValue:
		other = o.BaseRef
	default:
		return false, fmt.Errorf("cannot compare BaseRef with value of type %T", v)
	}

	if !bv.lua.IsSameVm(other.lua) {
		return false, errors.New("cannot compare values from different Lua VMs")
	}

	if bv.closed.Load() || other.closed.Load() {
		return false, errors.New("cannot use closed object")
	}

	res := C.luago_valuev2_equals(bv.lua.ptr(), bv.value, other.value)
	if res.error != nil {
		return false, moveErrorToGo(res.error)
	}
	return bool(res.value), nil
}

// Similar to Equal, but returns false if Equals returns an error
func (bv *BaseRef) LooseEquals(v any) bool {
	eq, err := bv.Equals(v)
	if err != nil {
		return false
	}
	return eq
}

// Close cleans up the value by calling the destructor and setting the pointer to nil.
func (bv *BaseRef) Close() error {

	if bv.closed.Swap(true) {
		return nil
	}

	if bv.lua.IsClosed() {
		return nil
	}

	C.luago_valuev2_destroy(bv.lua.ptr(), bv.value)
	runtime.SetFinalizer(bv, nil) // Remove finalizer to prevent double calls
	return nil
}
