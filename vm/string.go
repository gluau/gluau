package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"

// A LuaString is an abstraction over a Lua string object.
type LuaString struct {
	*BaseRef
}

// Returns the LuaString as a byte slice
func (l *LuaString) Bytes() []byte {
	return withBaseRefDefault(l.BaseRef, func(ptr C.struct_GoLuaValueV2) []byte {
		data := C.luago_string_as_bytes(l.lua.ptr(), ptr)
		return moveBytesToGo(data)
	})
}

// Returns the LuaString as a byte slice with nul terminator
func (l *LuaString) BytesWithNUL() []byte {
	return withBaseRefDefault(l.BaseRef, func(ptr C.struct_GoLuaValueV2) []byte {
		data := C.luago_string_as_bytes_with_nul(l.lua.ptr(), ptr)
		return moveBytesToGo(data)
	})
}

// String returns the LuaString as a Go string.
func (l *LuaString) String() string {
	// Convert the LuaString to a Go string
	return string(l.Bytes())
}
