package vm

import "github.com/gluau/gluau/internal/vm"

// A LuaTable is an abstraction over a Lua table object.
type LuaTable struct {
	ptr *vm.LuaTable
}

// Close cleans up the LuaTable
func (l *LuaTable) Close() {
	l.ptr.Close()
}
