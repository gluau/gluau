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
func (l *Lua) CreateString(s string) (*vm.LuaString, error) {
	result := l.lua.CreateString([]byte(s))
	if result.Error != "" {
		return nil, errors.New(result.Error)
	}
	return result.Value, nil
}

func CreateLuaVm() (*Lua, error) {
	vm, err := vm.CreateLuaVm()
	if err != nil {
		return nil, err
	}
	return &Lua{lua: vm}, nil
}
