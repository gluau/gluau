package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"
import (
	"errors"
	"unsafe"
)

var userdataTab = objectTab{
	dtor: func(ptr *C.void) {
		C.luago_free_userdata((*C.struct_LuaUserData)(unsafe.Pointer(ptr)))
	},
}

// A LuaUserData is an abstraction over a Lua userdata object.
type LuaUserData struct {
	lua    *GoLuaVmWrapper // The Lua VM wrapper that owns this userdata
	object *object
}

func (l *LuaUserData) innerPtr() (*C.struct_LuaUserData, error) {
	ptr, err := l.object.PointerNoLock()
	if err != nil {
		return nil, err // Return error if the object is closed
	}
	return (*C.struct_LuaUserData)(unsafe.Pointer(ptr)), nil
}

// Returns the associated data within the LuaUserData.
//
// Errors if there is no associated data or if the userdata is closed.
func (l *LuaUserData) AssociatedData() (any, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	ptr, err := l.innerPtr()
	if err != nil {
		return nil, err // Return error if the object is closed
	}

	res := C.luago_get_userdata_handle(ptr)
	if res.error != nil {
		err := moveErrorToGoError(res.error)
		return nil, err
	}

	value := uintptr(res.value)
	if value == 0 {
		return nil, nil // No associated data
	}
	data := getDynamicData(value)
	if data == nil {
		return nil, errors.New("internal error: handle is invalid")
	}
	return data, nil
}

// ToValue converts the LuaUserData to a Value.
func (l *LuaUserData) ToValue() Value {
	return &ValueUserData{value: l}
}

func (l *LuaUserData) Close() {
	if l == nil || l.object == nil {
		return // Nothing to close
	}
	// Close the LuaUserData object
	l.object.Close()
}
