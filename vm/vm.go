package vm

import (
	"github.com/gluau/gluau/internal/vm"
)

type Lua struct {
	lua *vm.GoLuaVmWrapper
}

func CreateLuaVm() (*Lua, error) {
	vm, err := vm.CreateLuaVm()
	if err != nil {
		return nil, err
	}
	return &Lua{lua: vm}, nil
}
