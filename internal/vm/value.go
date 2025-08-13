package vm

/*
#cgo LDFLAGS: -L../../rustlib -lrustlib
#include "../../rustlib/rustlib.h"
*/
import "C"
import "unsafe"

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
	value *C.void // TODO
}

func (v *ValueTable) Type() luaValueType {
	return luaValueTable
}
func (v *ValueTable) Close() {
	// TODO: Implement table
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
	Value string
}

func (v *ValueError) Type() luaValueType {
	return luaValueError
}
func (v *ValueError) Close() {}

type ValueOther struct {
	value *C.void // TODO
}

func (v *ValueOther) Type() luaValueType {
	return luaValueOther
}
func (v *ValueOther) Close() {}

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
		tablePtrPtr := (**C.void)(unsafe.Pointer(&item.data))
		tablePtr := *tablePtrPtr
		return &ValueTable{value: tablePtr} // TODO: Support tables
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
		errorPtr := *(**C.char)(unsafe.Pointer(&item.data))
		errStr := C.GoString(errorPtr)
		C.luago_error_free(errorPtr)      // Free the C string memory
		return &ValueError{Value: errStr} // Return the error as a Go string
	case C.LuaValueTypeOther:
		// Currently, always nil
		return &ValueOther{value: nil} // TODO: Support other types
	default:
		// Unknown type, return as Other
		return &ValueOther{value: nil} // Return nil for unknown types (as we cannot safely handle them)
	}
}
