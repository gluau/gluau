package vm

/*
#cgo LDFLAGS: -L../../rustlib -lrustlib
#include "../../rustlib/rustlib.h"
*/
import "C"
import (
	"runtime"
	"unsafe"
)

// A LuaTable is an abstraction over a Lua table object.
type LuaTable struct {
	ptr *C.void
}

// NewTable creates a LuaTable from a C pointer.
func NewTable(ptr *C.void) *LuaTable {
	if ptr == nil {
		return nil // Return nil if the pointer is nil
	}

	luaTab := &LuaTable{ptr: ptr}
	runtime.SetFinalizer(luaTab, (*LuaTable).Close) // Set finalizer to clean up LuaTable
	return luaTab
}

func (l *LuaTable) Close() {
	if l.ptr == nil {
		return
	}

	C.luago_free_table((*C.struct_LuaTable)(unsafe.Pointer(l.ptr)))
	l.ptr = nil                  // Prevent double free
	runtime.SetFinalizer(l, nil) // Remove finalizer to prevent double calls
}
