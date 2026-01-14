package vm

import (
	"fmt"
	"unsafe"
)

// A LightUserData is a wrapper for Lua light userdata values
type LightUserData struct {
	value unsafe.Pointer
}

func NewLightUserData(ptr unsafe.Pointer) LightUserData {
	return LightUserData{value: ptr}
}

// Value returns the pointer to the light user data.
// This pointer is not managed by Lua and should be used with caution.
// It is typically used for passing pointers to C code or for storing arbitrary data.
func (v *LightUserData) Value() unsafe.Pointer {
	return v.value
}

// String returns a string representation of the LightUserData.
func (v *LightUserData) String() string {
	if v.value == nil {
		return "ValueLightUserData: <nil>"
	}
	return fmt.Sprintf("ValueLightUserData: %p", v.value)
}

// A OtherValue is a value of unknown or unsupported type
type OtherValue struct {
	*BaseRef
}

// String returns a string representation of the OtherValue.
func (v *OtherValue) String() string {
	return "OtherValue"
}
