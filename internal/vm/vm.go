package vm

/*
#cgo LDFLAGS: -L../../rustlib -lrustlib
#include "../../rustlib/rustlib.h"
*/
import "C"
import (
	"fmt"
	"runtime"
)

type GoLuaVmWrapper struct {
	lua *C.struct_LuaVmWrapper
}

func (l *GoLuaVmWrapper) Close() {
	if l.lua == nil {
		return
	}

	fmt.Println("Closing Lua VM")

	C.freeluavm(l.lua)
	l.lua = nil                  // Prevent double free
	runtime.SetFinalizer(l, nil) // Remove finalizer to prevent double calls
}

func CreateLuaVm() (*GoLuaVmWrapper, error) {
	ptr := C.newluavm()
	if ptr == nil {
		return nil, fmt.Errorf("failed to create Lua VM")
	}
	vm := &GoLuaVmWrapper{lua: ptr}
	runtime.SetFinalizer(vm, (*GoLuaVmWrapper).Close)
	return vm, nil
}
