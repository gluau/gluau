package vm

/*
#cgo LDFLAGS: -L../../rustlib -lrustlib
#include "../../rustlib/rustlib.h"
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"unsafe"
)

type luaValueType int

const (
	luaValueNil           luaValueType = 0
	luaValueBoolean       luaValueType = 1
	luaValueLightUserData luaValueType = 2
	luaValueInteger       luaValueType = 3
	luaValueNumber        luaValueType = 4
	luaValueVector        luaValueType = 5
	luaValueString        luaValueType = 6
	luaValueTable         luaValueType = 7
	luaValueFunction      luaValueType = 8
	luaValueThread        luaValueType = 9
	luaValueUserData      luaValueType = 10
	luaValueBuffer        luaValueType = 11
	luaValueError         luaValueType = 12
	luaValueOther         luaValueType = 13
)

type Value interface {
	Type() luaValueType
	Close()
}

type ValueNil struct{}

func (v *ValueNil) Type() luaValueType {
	return luaValueNil
}
func (v *ValueNil) Close() {}

type ValueBoolean struct {
	Value bool
}

func (v *ValueBoolean) Type() luaValueType {
	return luaValueBoolean
}
func (v *ValueBoolean) Close() {}

type ValueLightUserData struct {
	Value unsafe.Pointer
}

func (v *ValueLightUserData) Type() luaValueType {
	return luaValueLightUserData
}

// Lightuserdata is a pointer to an arbitrary C data type.
// It does not need to be closed
func (v *ValueLightUserData) Close() {}

type ValueInteger struct {
	Value int64
}

func (v *ValueInteger) Type() luaValueType {
	return luaValueInteger
}
func (v *ValueInteger) Close() {}

type ValueNumber struct {
	Value float64
}

func (v *ValueNumber) Type() luaValueType {
	return luaValueNumber
}
func (v *ValueNumber) Close() {}

type ValueVector struct {
	Value [3]float32
}

func (v *ValueVector) Type() luaValueType {
	return luaValueVector
}
func (v *ValueVector) Close() {}

type ValueString struct {
	Value *LuaString
}

func (v *ValueString) Type() luaValueType {
	return luaValueString
}
func (v *ValueString) Close() {
	v.Value.Close()
}

type ValueTable struct {
	Value *LuaTable
}

func (v *ValueTable) Type() luaValueType {
	return luaValueTable
}
func (v *ValueTable) Close() {
	v.Value.Close()
}

type ValueFunction struct {
	value *C.void // TODO
}

func (v *ValueFunction) Type() luaValueType {
	return luaValueFunction
}
func (v *ValueFunction) Close() {
	// TODO: Implement function
}

type ValueThread struct {
	value *C.void // TODO
}

func (v *ValueThread) Type() luaValueType {
	return luaValueThread
}
func (v *ValueThread) Close() {
	// TODO: Implement thread
}

type ValueUserData struct {
	value *C.void // TODO
}

func (v *ValueUserData) Type() luaValueType {
	return luaValueUserData
}
func (v *ValueUserData) Close() {
	// TODO: Implement user data
}

type ValueBuffer struct {
	value *C.void // TODO
}

func (v *ValueBuffer) Type() luaValueType {
	return luaValueBuffer
}
func (v *ValueBuffer) Close() {
	// TODO: Implement buffer
}

type ValueError struct {
	Value *ErrorVariant
}

func (v *ValueError) Type() luaValueType {
	return luaValueError
}
func (v *ValueError) Close() {
	v.Value.Close()
}

type ValueOther struct {
	value *C.void // TODO
}

func (v *ValueOther) Type() luaValueType {
	return luaValueOther
}
func (v *ValueOther) Close() {}

// CloneValue clones a C struct_GoLuaValue to a new C struct_GoLuaValue.
func CloneValue(item C.struct_GoLuaValue) C.struct_GoLuaValue {
	return C.luago_value_clone(item)
}

// ValueFromC converts a C struct_GoLuaValue to a Go Value interface.
// Note: this does not clone the value, it simply converts it.
//
// Internal API: do not use unless you know what you're doing
func ValueFromC(item C.struct_GoLuaValue) Value {
	switch item.tag {
	case C.LuaValueTypeNil:
		return &ValueNil{}
	case C.LuaValueTypeBoolean:
		val := *(*bool)(unsafe.Pointer(&item.data))
		return &ValueBoolean{Value: val}
	case C.LuaValueTypeLightUserData:
		valPtr := *(**unsafe.Pointer)(unsafe.Pointer(&item.data))
		val := *valPtr
		return &ValueLightUserData{Value: val}
	case C.LuaValueTypeInteger:
		val := *(*int64)(unsafe.Pointer(&item.data))
		return &ValueInteger{Value: val}
	case C.LuaValueTypeNumber:
		val := *(*float64)(unsafe.Pointer(&item.data))
		return &ValueNumber{Value: val}
	case C.LuaValueTypeVector:
		vec := *(*[3]C.float)(unsafe.Pointer(&item.data))
		return &ValueVector{Value: [3]float32{float32(vec[0]), float32(vec[1]), float32(vec[2])}}
	case C.LuaValueTypeString:
		ptrToPtr := (**C.struct_LuaString)(unsafe.Pointer(&item.data))
		strPtr := (*C.void)(unsafe.Pointer(*ptrToPtr))
		str := NewString(strPtr)
		return &ValueString{Value: str}
	case C.LuaValueTypeTable:
		ptrToPtr := (**C.struct_LuaTable)(unsafe.Pointer(&item.data))
		tabPtr := (*C.void)(unsafe.Pointer(*ptrToPtr))
		tab := NewTable(tabPtr)
		return &ValueTable{Value: tab}
	case C.LuaValueTypeFunction:
		funcPtrPtr := (**C.void)(unsafe.Pointer(&item.data))
		funcPtr := *funcPtrPtr
		return &ValueFunction{value: funcPtr} // TODO: Support functions
	case C.LuaValueTypeThread:
		threadPtrPtr := (**C.void)(unsafe.Pointer(&item.data))
		threadPtr := *threadPtrPtr
		return &ValueThread{value: threadPtr} // TODO: Support threads
	case C.LuaValueTypeUserData:
		userDataPtrPtr := (**C.void)(unsafe.Pointer(&item.data))
		userDataPtr := *userDataPtrPtr
		return &ValueUserData{value: userDataPtr} // TODO: Support user data
	case C.LuaValueTypeBuffer:
		bufferPtrPtr := (**C.void)(unsafe.Pointer(&item.data))
		bufferPtr := *bufferPtrPtr
		return &ValueBuffer{value: bufferPtr} // TODO: Support buffers
	case C.LuaValueTypeError:
		ptrToPtr := (**C.struct_ErrorVariant)(unsafe.Pointer(&item.data))
		strPtr := (*C.void)(unsafe.Pointer(*ptrToPtr))
		str := NewErrorVariant(strPtr)
		return &ValueError{Value: str}
	case C.LuaValueTypeOther:
		// Currently, always nil
		return &ValueOther{value: nil} // TODO: Support other types
	default:
		// Unknown type, return as Other
		return &ValueOther{value: nil} // Return nil for unknown types (as we cannot safely handle them)
	}
}

// DirectValueToC converts a Go Value interface to a C struct_GoLuaValue
// with the intent that the value will be passed to Rust code.
// Note: this does not clone the value, it simply converts it.
//
// Internal API: do not use unless you know what you're doing
//
// # WARNING
//
// You probably want to use ValueToC instead of this function.
//
// In particular, ValueFromC should *never* be called directly on the result of this function,
// as it may lead to memory corruption or undefined behavior.
func DirectValueToC(value Value) (C.struct_GoLuaValue, error) {
	var cVal C.struct_GoLuaValue
	switch value.Type() {
	case luaValueNil:
		break
	case luaValueBoolean:
		boolVal := value.(*ValueBoolean)
		cVal.tag = C.LuaValueTypeBoolean
		*(*C.bool)(unsafe.Pointer(&cVal.data)) = C.bool(boolVal.Value)
	case luaValueLightUserData:
		lightUserDataVal := value.(*ValueLightUserData)
		cVal.tag = C.LuaValueTypeLightUserData
		*(*unsafe.Pointer)(unsafe.Pointer(&cVal.data)) = lightUserDataVal.Value
	case luaValueInteger:
		intVal := value.(*ValueInteger)
		cVal.tag = C.LuaValueTypeInteger
		*(*int64)(unsafe.Pointer(&cVal.data)) = intVal.Value
	case luaValueNumber:
		numVal := value.(*ValueNumber)
		cVal.tag = C.LuaValueTypeNumber
		*(*float64)(unsafe.Pointer(&cVal.data)) = numVal.Value
	case luaValueVector:
		cVal.tag = C.LuaValueTypeVector
		vecVal := value.(*ValueVector)
		*(*[3]float32)(unsafe.Pointer(&cVal.data)) = vecVal.Value
	case luaValueString:
		strVal := value.(*ValueString)
		if strVal.Value == nil || strVal.Value.ptr == nil {
			return cVal, errors.New("cannot convert nil LuaString to C value")
		}
		cVal.tag = C.LuaValueTypeString
		*(*unsafe.Pointer)(unsafe.Pointer(&cVal.data)) = unsafe.Pointer(strVal.Value.ptr)
	case luaValueTable:
		tabVal := value.(*ValueTable)
		if tabVal.Value == nil || tabVal.Value.ptr == nil {
			return cVal, errors.New("cannot convert nil LuaTable to C value")
		}
		cVal.tag = C.LuaValueTypeString
		*(*unsafe.Pointer)(unsafe.Pointer(&cVal.data)) = unsafe.Pointer(tabVal.Value.ptr)
	case luaValueFunction:
		funcVal := value.(*ValueFunction)
		if funcVal.value == nil {
			return cVal, errors.New("cannot convert nil LuaFunction to C value")
		}
		cVal.tag = C.LuaValueTypeFunction
		*(*unsafe.Pointer)(unsafe.Pointer(&cVal.data)) = unsafe.Pointer(funcVal.value)
	case luaValueThread:
		threadVal := value.(*ValueThread)
		if threadVal.value == nil {
			return cVal, errors.New("cannot convert nil LuaThread to C value")
		}
		cVal.tag = C.LuaValueTypeThread
		*(*unsafe.Pointer)(unsafe.Pointer(&cVal.data)) = unsafe.Pointer(threadVal.value)
	case luaValueUserData:
		userDataVal := value.(*ValueUserData)
		if userDataVal.value == nil {
			return cVal, errors.New("cannot convert nil LuaUserData to C value")
		}
		cVal.tag = C.LuaValueTypeUserData
		*(*unsafe.Pointer)(unsafe.Pointer(&cVal.data)) = unsafe.Pointer(userDataVal.value)
	case luaValueBuffer:
		bufferVal := value.(*ValueBuffer)
		if bufferVal.value == nil {
			return cVal, errors.New("cannot convert nil LuaBuffer to C value")
		}
		cVal.tag = C.LuaValueTypeBuffer
		*(*unsafe.Pointer)(unsafe.Pointer(&cVal.data)) = unsafe.Pointer(bufferVal.value)
	case luaValueError:
		errVal := value.(*ValueError)
		if errVal.Value == nil || errVal.Value.ptr == nil {
			return cVal, errors.New("cannot convert nil ErrorVariant to C value")
		}
		cVal.tag = C.LuaValueTypeError
		*(*unsafe.Pointer)(unsafe.Pointer(&cVal.data)) = unsafe.Pointer(errVal.Value.ptr)
	case luaValueOther:
		// Currently, always nil
		cVal.tag = C.LuaValueTypeOther
		*(*unsafe.Pointer)(unsafe.Pointer(&cVal.data)) = nil // Return nil
	default:
		return cVal, errors.New("unknown Lua value type")
	}

	return cVal, nil
}

// ValueToC converts a Go Value interface to a C struct_GoLuaValue
// with the intent that the value will be passed to Rust code.
// It clones the value ref pointer to ensure it is safe to use in C code.
//
// Internal API: do not use unless you know what you're doing
func ValueToC(value Value) (C.struct_GoLuaValue, error) {
	if value == nil {
		return C.struct_GoLuaValue{}, errors.New("cannot convert nil value to C")
	}
	cptr, err := DirectValueToC(value)
	if err != nil {
		return cptr, err
	}
	return CloneValue(cptr), nil
}
