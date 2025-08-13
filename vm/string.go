package vm

import "github.com/gluau/gluau/internal/vm"

type LuaString struct {
	ptr *vm.LuaString
}

// Returns the LuaString as a byte slice
func (l *LuaString) Bytes() []byte {
	return l.ptr.Bytes()
}

// Returns the LuaString as a byte slice with nul terminator
func (l *LuaString) BytesWithNul() []byte {
	return l.ptr.BytesWithNul()
}

// Returns a 'pointer' to a LuaString
// Note: this pointer is only useful for hashing and debugging and you cannot
// get back a LuaString from it.
func (l *LuaString) Pointer() uint64 {
	return l.ptr.Pointer()
}

// Close cleans up the LuaString
func (l *LuaString) Close() {
	l.ptr.Close()
}
