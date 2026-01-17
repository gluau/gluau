package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"

// A LuaString is an abstraction over a Lua string object.
type LuaString struct {
	*BaseRef
}

// Returns the length of the LuaString in bytes
func (l *LuaString) Len() int {
	return withBaseRefDefault(l.BaseRef, func(ptr C.struct_GoLuaValueV2) int {
		length := C.luago_string_len(l.lua.ptr(), ptr)
		return int(length)
	})
}

// Returns the LuaString as a byte slice
func (l *LuaString) Bytes() []byte {
	return withBaseRefDefault(l.BaseRef, func(ptr C.struct_GoLuaValueV2) []byte {
		len := C.luago_string_len(l.lua.ptr(), ptr)
		buf := make([]byte, int(len))
		C.luago_string_as_bytes(l.lua.ptr(), ptr, createGoOwnedBytes(buf))
		return []byte(buf)
	})
}

// String returns the LuaString as a Go string.
func (l *LuaString) String() string {
	// Convert the LuaString to a Go string
	return string(l.Bytes())
}
