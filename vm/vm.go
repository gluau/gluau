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

// DebugValue is a testing function to get a debug value from the Lua VM
func (l *Lua) DebugValue() [2]vm.Value {
	return l.lua.DebugValue()
}

func CreateLuaVm() (*Lua, error) {
	vm, err := vm.CreateLuaVm()
	if err != nil {
		return nil, err
	}
	return &Lua{lua: vm}, nil
}
