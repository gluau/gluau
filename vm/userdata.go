package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"
import (
	"errors"
	"fmt"
)

// A LuaUserData is an abstraction over a Lua userdata object.
type LuaUserData struct {
	*BaseRef
}

// Returns the associated data within the LuaUserData.
//
// Errors if there is no associated data or if the userdata is closed.
func (l *LuaUserData) AssociatedData() (any, error) {
	return withBaseRef(l.BaseRef, func(ptr C.struct_GoLuaValueV2) (any, error) {
		res := C.luago_get_userdata_handle(l.lua.ptr(), ptr)
		if res.error != nil {
			err := moveErrorToGo(res.error)
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
	})
}

// Metatable returns the metatable of the LuaUserData.
func (l *LuaUserData) Metatable() (*LuaTable, error) {
	return withBaseRef(l.BaseRef, func(ptr C.struct_GoLuaValueV2) (*LuaTable, error) {
		res := C.luago_userdata_metatable(l.lua.ptr(), ptr)
		if res.error != nil {
			err := moveErrorToGo(res.error)
			return nil, err
		}
		tab, ok := castValue(l.lua, res.value).(*LuaTable)
		if !ok {
			return nil, fmt.Errorf("expected LuaTable from userdata metatable, got %T", tab)
		}
		return tab, nil
	})
}

// String returns a string representation of the LuaUserData.
func (l *LuaUserData) String() string {
	return "LuaUserData(pointer: " + fmt.Sprintf("%#x", l.Pointer()) + ")"
}
