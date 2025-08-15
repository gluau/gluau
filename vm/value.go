package vm

/*
#include "../rustlib/rustlib.h"
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"unsafe"
)

type LuaValueType int

const (
	LuaValueNil           LuaValueType = 0
	LuaValueBoolean       LuaValueType = 1
	LuaValueLightUserData LuaValueType = 2
	LuaValueInteger       LuaValueType = 3
	LuaValueNumber        LuaValueType = 4
	LuaValueVector        LuaValueType = 5
	LuaValueString        LuaValueType = 6
	LuaValueTable         LuaValueType = 7
	LuaValueFunction      LuaValueType = 8
	LuaValueThread        LuaValueType = 9
	LuaValueUserData      LuaValueType = 10
	LuaValueBuffer        LuaValueType = 11
	LuaValueError         LuaValueType = 12
	LuaValueOther         LuaValueType = 13

	// Custom types not sent by Rust ever
	// to make the library more ergonomic
	LuaValueCustom_GoString LuaValueType = 14
)

type Value interface {
	Type() LuaValueType
	Close()
	object() *object  // Returns the underlying object for this value
	needsClone() bool // Returns true if the value needs to be cloned in valueToC to be safely passed to rust
}

// ValueNil represents a Lua nil value.
type ValueNil struct{}

func (v *ValueNil) Type() LuaValueType {
	return LuaValueNil
}
func (v *ValueNil) Close() {}
func (v *ValueNil) object() *object {
	return nil // Nil has no underlying object
}
func (v *ValueNil) needsClone() bool {
	return false // Nil does not need to be cloned
}

type ValueBoolean struct {
	value bool
}

func NewValueBoolean(value bool) *ValueBoolean {
	return &ValueBoolean{value: value}
}

// Value returns the boolean value of the ValueBoolean.
func (v *ValueBoolean) Value() bool {
	return v.value
}
func (v *ValueBoolean) Type() LuaValueType {
	return LuaValueBoolean
}
func (v *ValueBoolean) Close() {}
func (v *ValueBoolean) object() *object {
	return nil // Boolean has no underlying object
}
func (v *ValueBoolean) needsClone() bool {
	return false // Boolean does not need to be cloned
}

// ValueLightUserData is a pointer to an arbitrary C data type.
type ValueLightUserData struct {
	value unsafe.Pointer
}

func NewValueLightUserData(value unsafe.Pointer) *ValueLightUserData {
	return &ValueLightUserData{value: value}
}

// Value returns the pointer to the light user data.
// This pointer is not managed by Lua and should be used with caution.
// It is typically used for passing pointers to C code or for storing arbitrary data.
func (v *ValueLightUserData) Value() unsafe.Pointer {
	return v.value
}
func (v *ValueLightUserData) Type() LuaValueType {
	return LuaValueLightUserData
}
func (v *ValueLightUserData) Close() {}
func (v *ValueLightUserData) object() *object {
	return nil // LightUserData has no underlying object
}
func (v *ValueLightUserData) needsClone() bool {
	return false // LightUserData does not need to be cloned
}

// ValueInteger represents a Lua integer value.
type ValueInteger struct {
	value int64
}

func NewValueInteger(value int64) *ValueInteger {
	return &ValueInteger{value: value}
}

func (v *ValueInteger) Value() int64 {
	return v.value
}
func (v *ValueInteger) Type() LuaValueType {
	return LuaValueInteger
}
func (v *ValueInteger) Close() {}
func (v *ValueInteger) object() *object {
	return nil // Integer has no underlying object
}
func (v *ValueInteger) needsClone() bool {
	return false // Integer does not need to be cloned
}

// ValueNumber represents a Lua number value.
type ValueNumber struct {
	value float64
}

func NewValueNumber(value float64) *ValueNumber {
	return &ValueNumber{value: value}
}

// Value returns the float64 value of the ValueNumber.
func (v *ValueNumber) Value() float64 {
	return v.value
}
func (v *ValueNumber) Type() LuaValueType {
	return LuaValueNumber
}
func (v *ValueNumber) Close() {}
func (v *ValueNumber) object() *object {
	return nil // Number has no underlying object
}
func (v *ValueNumber) needsClone() bool {
	return false // Number does not need to be cloned
}

// ValueVector represents a Luau vector value (3D vector).
//
// This is Luau-specific
type ValueVector struct {
	value [3]float32
}

func NewValueVector(x, y, z float32) *ValueVector {
	return &ValueVector{value: [3]float32{x, y, z}}
}

// Value returns the [3]float32 value of the ValueVector.
func (v *ValueVector) Value() [3]float32 {
	return v.value
}
func (v *ValueVector) Type() LuaValueType {
	return LuaValueVector
}
func (v *ValueVector) Close() {}
func (v *ValueVector) object() *object {
	return nil // Vector has no underlying object
}
func (v *ValueVector) needsClone() bool {
	return false // Vector does not need to be cloned
}

// ValueString represents a Lua string value.
type ValueString struct {
	value *LuaString
}

// NewValueString creates a new ValueString from a LuaString.
func (v *ValueString) Value() *LuaString {
	return v.value
}
func (v *ValueString) Type() LuaValueType {
	return LuaValueString
}
func (v *ValueString) Close() {
	v.value.Close()
}
func (v *ValueString) object() *object {
	if v.value == nil {
		return nil // String has no underlying object if nil
	}
	return v.value.object
}
func (v *ValueString) needsClone() bool {
	return true // String needs to be cloned to be safely passed to rust
}

// ValueTable represents a Lua table value.
type ValueTable struct {
	value *LuaTable
}

func (v *ValueTable) Value() *LuaTable {
	return v.value
}
func (v *ValueTable) Type() LuaValueType {
	return LuaValueTable
}
func (v *ValueTable) Close() {
	v.value.Close()
}
func (v *ValueTable) object() *object {
	if v.value == nil {
		return nil // Table has no underlying object if nil
	}
	return v.value.object
}
func (v *ValueTable) needsClone() bool {
	return true // Table needs to be cloned to be safely passed to rust
}

type ValueFunction struct {
	value *LuaFunction
}

func (v *ValueFunction) Value() *LuaFunction {
	return v.value
}
func (v *ValueFunction) Type() LuaValueType {
	return LuaValueFunction
}
func (v *ValueFunction) Close() {
	v.value.Close()
}
func (v *ValueFunction) object() *object {
	return v.value.object
}
func (v *ValueFunction) needsClone() bool {
	return true // Function needs to be cloned to be safely passed to rust
}

type ValueThread struct {
	value *C.void // TODO
}

func (v *ValueThread) Type() LuaValueType {
	return LuaValueThread
}
func (v *ValueThread) Close() {
	// TODO: Implement thread
}
func (v *ValueThread) object() *object {
	return nil // Thread has no underlying object
}
func (v *ValueThread) needsClone() bool {
	return true // Thread needs to be cloned to be safely passed to rust
}

type ValueUserData struct {
	value *C.void // TODO
}

func (v *ValueUserData) Type() LuaValueType {
	return LuaValueUserData
}
func (v *ValueUserData) Close() {
	// TODO: Implement user data
}
func (v *ValueUserData) object() *object {
	return nil // UserData has no underlying object
}
func (v *ValueUserData) needsClone() bool {
	return true // UserData needs to be cloned to be safely passed to rust
}

type ValueBuffer struct {
	value *C.void // TODO
}

func (v *ValueBuffer) Type() LuaValueType {
	return LuaValueBuffer
}
func (v *ValueBuffer) Close() {
	// TODO: Implement buffer
}
func (v *ValueBuffer) object() *object {
	return nil // Buffer has no underlying object
}
func (v *ValueBuffer) needsClone() bool {
	return true // Buffer needs to be cloned to be safely passed to rust
}

// ValueError represents a Lua error value.
type ValueError struct {
	value *ErrorVariant
}

func (v *ValueError) Value() *ErrorVariant {
	return v.value
}
func (v *ValueError) Type() LuaValueType {
	return LuaValueError
}
func (v *ValueError) Close() {
	v.value.Close()
}
func (v *ValueError) object() *object {
	if v.value == nil {
		return nil // Error has no underlying object if nil
	}
	return v.value.object
}
func (v *ValueError) needsClone() bool {
	return true // Error needs to be cloned to be safely passed to rust
}

type ValueOther struct {
	value *C.void // TODO
}

func (v *ValueOther) Type() LuaValueType {
	return LuaValueOther
}
func (v *ValueOther) Close() {}
func (v *ValueOther) object() *object {
	return nil // Other has no underlying object
}
func (v *ValueOther) needsClone() bool {
	return false // Other does not need to be cloned, as it is always nil (for now)
}

type GoString string

func (v GoString) Type() LuaValueType {
	return LuaValueCustom_GoString
}
func (v GoString) Close() {}
func (v GoString) object() *object {
	return nil // GoString has no underlying object
}
func (v GoString) needsClone() bool {
	// GoString does not need to be cloned as it is a extension type
	// that is converted to a LuaString when converted to C
	//
	// Memory leak safety: as all methods taking in an LuaValue in rust side
	// will take ownership of the value, the LuaString will be dropped in
	// rust side automatically.
	return false
}

// CloneValue clones a C struct_GoLuaValue to a new C struct_GoLuaValue.
func cloneValue(item C.struct_GoLuaValue) C.struct_GoLuaValue {
	return C.luago_value_clone(item)
}

// ValueFromC converts a C struct_GoLuaValue to a Go Value interface.
// Note: this does not clone the value, it simply converts it.
//
// Internal API: do not use unless you know what you're doing
func (l *GoLuaVmWrapper) valueFromC(item C.struct_GoLuaValue) Value {
	switch item.tag {
	case C.LuaValueTypeNil:
		return &ValueNil{}
	case C.LuaValueTypeBoolean:
		val := *(*bool)(unsafe.Pointer(&item.data))
		return &ValueBoolean{value: val}
	case C.LuaValueTypeLightUserData:
		valPtr := *(**unsafe.Pointer)(unsafe.Pointer(&item.data))
		val := *valPtr
		return &ValueLightUserData{value: val}
	case C.LuaValueTypeInteger:
		val := *(*int64)(unsafe.Pointer(&item.data))
		return &ValueInteger{value: val}
	case C.LuaValueTypeNumber:
		val := *(*float64)(unsafe.Pointer(&item.data))
		return &ValueNumber{value: val}
	case C.LuaValueTypeVector:
		vec := *(*[3]C.float)(unsafe.Pointer(&item.data))
		return &ValueVector{value: [3]float32{float32(vec[0]), float32(vec[1]), float32(vec[2])}}
	case C.LuaValueTypeString:
		ptrToPtr := (**C.struct_LuaString)(unsafe.Pointer(&item.data))
		strPtr := (*C.void)(unsafe.Pointer(*ptrToPtr))
		str := &LuaString{object: newObject(strPtr, stringTab)}
		return &ValueString{value: str}
	case C.LuaValueTypeTable:
		ptrToPtr := (**C.struct_LuaTable)(unsafe.Pointer(&item.data))
		tabPtr := (*C.void)(unsafe.Pointer(*ptrToPtr))
		tab := &LuaTable{object: newObject(tabPtr, tableTab), lua: l}
		return &ValueTable{value: tab}
	case C.LuaValueTypeFunction:
		ptrToPtr := (**C.struct_LuaFunction)(unsafe.Pointer(&item.data))
		funcPtr := (*C.void)(unsafe.Pointer(*ptrToPtr))
		funct := &LuaFunction{object: newObject(funcPtr, functionTab)}
		return &ValueFunction{value: funct}
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
		str := &ErrorVariant{object: newObject(strPtr, errorVariantTab)}
		return &ValueError{value: str}
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
// Note: this does not clone the value or performing any locking, it simply converts it.
//
// Internal API: do not use unless you know what you're doing
//
// # WARNING
//
// You probably want to use ValueToC instead of this function.
//
// In particular, ValueFromC should *never* be called directly on the result of this function,
// as it may lead to memory corruption or undefined behavior.
func (l *GoLuaVmWrapper) _directValueToC(value Value) (C.struct_GoLuaValue, error) {
	var cVal C.struct_GoLuaValue
	switch value.Type() {
	case LuaValueNil:
		break
	case LuaValueBoolean:
		boolVal := value.(*ValueBoolean)
		cVal.tag = C.LuaValueTypeBoolean
		*(*C.bool)(unsafe.Pointer(&cVal.data)) = C.bool(boolVal.value)
	case LuaValueLightUserData:
		lightUserDataVal := value.(*ValueLightUserData)
		cVal.tag = C.LuaValueTypeLightUserData
		*(*unsafe.Pointer)(unsafe.Pointer(&cVal.data)) = lightUserDataVal.value
	case LuaValueInteger:
		intVal := value.(*ValueInteger)
		cVal.tag = C.LuaValueTypeInteger
		*(*int64)(unsafe.Pointer(&cVal.data)) = intVal.value
	case LuaValueNumber:
		numVal := value.(*ValueNumber)
		cVal.tag = C.LuaValueTypeNumber
		*(*float64)(unsafe.Pointer(&cVal.data)) = numVal.value
	case LuaValueVector:
		cVal.tag = C.LuaValueTypeVector
		vecVal := value.(*ValueVector)
		*(*[3]float32)(unsafe.Pointer(&cVal.data)) = vecVal.value
	case LuaValueString:
		strVal := value.(*ValueString)
		ptr, err := strVal.value.object.PointerNoLock()
		if err != nil {
			return cVal, errors.New("cannot convert closed LuaString to C value")
		}
		cVal.tag = C.LuaValueTypeString
		*(*unsafe.Pointer)(unsafe.Pointer(&cVal.data)) = unsafe.Pointer(ptr)
	case LuaValueTable:
		tabVal := value.(*ValueTable)
		ptr, err := tabVal.value.object.PointerNoLock()
		if err != nil {
			return cVal, errors.New("cannot convert closed LuaTable to C value")
		}
		cVal.tag = C.LuaValueTypeString
		*(*unsafe.Pointer)(unsafe.Pointer(&cVal.data)) = unsafe.Pointer(ptr)
	case LuaValueFunction:
		funcVal := value.(*ValueFunction)
		ptr, err := funcVal.value.object.PointerNoLock()
		if err != nil {
			return cVal, errors.New("cannot convert closed LuaFunction to C value")
		}
		cVal.tag = C.LuaValueTypeFunction
		*(*unsafe.Pointer)(unsafe.Pointer(&cVal.data)) = unsafe.Pointer(ptr)
	case LuaValueThread:
		threadVal := value.(*ValueThread)
		if threadVal.value == nil {
			return cVal, errors.New("cannot convert nil LuaThread to C value")
		}
		cVal.tag = C.LuaValueTypeThread
		*(*unsafe.Pointer)(unsafe.Pointer(&cVal.data)) = unsafe.Pointer(threadVal.value)
	case LuaValueUserData:
		userDataVal := value.(*ValueUserData)
		if userDataVal.value == nil {
			return cVal, errors.New("cannot convert nil LuaUserData to C value")
		}
		cVal.tag = C.LuaValueTypeUserData
		*(*unsafe.Pointer)(unsafe.Pointer(&cVal.data)) = unsafe.Pointer(userDataVal.value)
	case LuaValueBuffer:
		bufferVal := value.(*ValueBuffer)
		if bufferVal.value == nil {
			return cVal, errors.New("cannot convert nil LuaBuffer to C value")
		}
		cVal.tag = C.LuaValueTypeBuffer
		*(*unsafe.Pointer)(unsafe.Pointer(&cVal.data)) = unsafe.Pointer(bufferVal.value)
	case LuaValueError:
		errVal := value.(*ValueError)
		ptr, err := errVal.value.object.PointerNoLock()
		if err != nil {
			return cVal, errors.New("cannot convert closed ErrorVariant to C value")
		}
		cVal.tag = C.LuaValueTypeError
		*(*unsafe.Pointer)(unsafe.Pointer(&cVal.data)) = unsafe.Pointer(ptr)
	case LuaValueOther:
		// Currently, always nil
		cVal.tag = C.LuaValueTypeOther
		*(*unsafe.Pointer)(unsafe.Pointer(&cVal.data)) = nil // Return nil
	case LuaValueCustom_GoString:
		// This is a temporary string that should not have a finalizer attached to it
		goStrVal := value.(GoString)
		// Create a LuaString from the Go string
		luaString, err := l.createStringAsPtr([]byte(goStrVal))
		if err != nil {
			return cVal, err // Return error if the string cannot be created
		}
		cVal.tag = C.LuaValueTypeString
		*(*unsafe.Pointer)(unsafe.Pointer(&cVal.data)) = unsafe.Pointer(luaString)
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
func (l *GoLuaVmWrapper) valueToC(value Value) (C.struct_GoLuaValue, error) {
	if value == nil {
		return C.struct_GoLuaValue{}, errors.New("cannot convert nil value to C")
	}

	obj := value.object()
	if obj != nil {
		// Acquire read lock to ensure the object is not closed while converting
		obj.RLock()
		defer obj.RUnlock()
	}

	cptr, err := l._directValueToC(value)
	if err != nil {
		return cptr, err
	}

	// Only values with needsClone need to be cloned
	//
	// As a matter of fact, cloning non objects may lead
	// to a memory leak when dealing with GoString etc.
	cloneNeeded := value.needsClone()
	if !cloneNeeded {
		return cptr, nil
	}

	return cloneValue(cptr), nil
}
