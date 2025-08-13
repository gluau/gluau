package main

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/gluau/gluau/internal/callback" // Import to ensure callback package is initialized
	"github.com/gluau/gluau/vm"
)

// #include <stdlib.h>
// #include "./rustlib/rustlib.h"
import "C"

func main() {
	for i := 0; i < 10; i++ {
		fmt.Println("testing callbacks")
		callback := callback.NewGoCallback(func(val unsafe.Pointer) {
			// val for this callback example is a pointer to an integer
			num := *(*C.int)(val)
			fmt.Println("Callback called with value:", num)
		}, func() {
			fmt.Println("Callback is being dropped")
		})

		val := 12345 + i
		cIntPtr := (*C.int)(C.malloc(C.sizeof_int))
		defer C.free(unsafe.Pointer(cIntPtr))
		*cIntPtr = C.int(val)
		cCallback := callback.ToC()
		C.test_callback((*C.struct_IGoCallback)(unsafe.Pointer(cCallback)), unsafe.Pointer(cIntPtr))
	}

	// Basic test to ensure the Lua VM can be created and closed properly on GC
	createVm := func() {
		vm.CreateLuaVm()
	}

	luaVm, err := vm.CreateLuaVm()
	if err != nil {
		fmt.Println("Error creating Lua VM:", err)
		return
	}
	fmt.Println("Lua VM created successfully", luaVm)
	for i := 0; i < 10; i++ {
		fmt.Println("Creating another Lua VM instance")
		createVm()
	}
	for i := 0; i < 10; i++ {
		fmt.Println("Creating another Lua VM instance")
		createVm()
		runtime.GC() // Force garbage collection to test finalizers are called
	}
	runtime.GC() // Force garbage collection to test finalizers are called
	//defer luaVm.Drop() // Ensure we free the Lua VM when done

	// You can now use luaVm to interact with the Lua VM
	// For example, you might call methods on luaVm.lua
	// to execute Lua scripts or manipulate Lua state.
}
