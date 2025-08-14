package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"
import (
	"errors"
	"unsafe"
)

var tableTab = objectTab{
	dtor: func(ptr *C.void) {
		C.luago_free_table((*C.struct_LuaTable)(unsafe.Pointer(ptr)))
	},
}

// A LuaTable is an abstraction over a Lua table object.
type LuaTable struct {
	*object
}

// Clear the LuaTable
func (l *LuaTable) Clear() error {
	l.RLock()
	defer l.RUnlock()

	ptr, err := l.object.PointerNoLock()
	if err != nil {
		return err // Return error if the object is closed
	}

	res := C.luago_table_clear((*C.struct_LuaTable)(unsafe.Pointer(ptr)))

	var result = goResultFromC[bool](res)
	if result.Error != "" {
		return errors.New(result.Error)
	}
	return nil
}

// ContainsKey checks if the LuaTable contains a key
func (l *LuaTable) ContainsKey(key Value) error {
	l.RLock()
	defer l.RUnlock()

	ptr, err := l.object.PointerNoLock()
	if err != nil {
		return err // Return error if the object is closed
	}
	keyVal, err := valueToC(key)
	if err != nil {
		return err // Return error if the value cannot be converted
	}

	res := C.luago_table_contains_key((*C.struct_LuaTable)(unsafe.Pointer(ptr)), keyVal)

	var result = goResultFromC[bool](res)
	if result.Error != "" {
		return errors.New(result.Error)
	}
	return nil
}

func (l *LuaTable) Close() {
	if l == nil || l.object == nil {
		return // Nothing to close
	}
	// Close the LuaTable object
	l.object.Close()
}
