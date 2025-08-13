// Package vm provides the user-facing API for interacting with the Lua VM.
package vm

import (
	"errors"

	"github.com/gluau/gluau/internal/vm"
)

// Lua is a handle to the underlying Lua VM
type Lua struct {
	lua *vm.GoLuaVmWrapper
}

// SetMemoryLimit sets the memory limit for the Lua VM.
func (l *Lua) SetMemoryLimit(limit int) error {
	return l.lua.SetMemoryLimit(limit)
}

// Close cleans up the Lua VM
func (l *Lua) Close() {
	l.lua.Close()
}

// CreateString creates a Lua string from a Go string
//
// The created Lua string is owned by the Lua VM
func (l *Lua) CreateString(s string) (*LuaString, error) {
	result := l.lua.CreateString([]byte(s))
	if result.Error != "" {
		return nil, errors.New(result.Error)
	}
	return &LuaString{ptr: result.Value}, nil
}

// CreateTable creates a Lua table
//
// The created Lua table is owned by the Lua VM
func (l *Lua) CreateTable() (*LuaTable, error) {
	result := l.lua.CreateTable()
	if result.Error != "" {
		return nil, errors.New(result.Error)
	}
	return &LuaTable{ptr: result.Value}, nil
}

// CreateTableWithCapacity creates a Lua table with specified capacity
//
// narr is the initial size of the array part, and nrec is the initial size of the hash part.
//
// The created Lua table is owned by the Lua VM
func (l *Lua) CreateTableWithCapacity(narr, nrec int) (*LuaTable, error) {
	result := l.lua.CreateTableWithCapacity(narr, nrec)
	if result.Error != "" {
		return nil, errors.New(result.Error)
	}
	return &LuaTable{ptr: result.Value}, nil
}

// DebugValue is a testing function to get a debug value from the Lua VM
func (l *Lua) DebugValue() [3]vm.Value {
	return l.lua.DebugValue()
}

func CreateLuaVm() (*Lua, error) {
	vm, err := vm.CreateLuaVm()
	if err != nil {
		return nil, err
	}
	return &Lua{lua: vm}, nil
}
