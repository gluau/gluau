package main

import (
	"fmt"
	"runtime"
	"time"
	"unsafe"

	"github.com/gluau/gluau/internal/callback" // Import to ensure callback package is initialized
	ivm "github.com/gluau/gluau/internal/vm"   // Import internal vm package for Lua VM operations
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

	vm, err := vm.CreateLuaVm()
	if err != nil {
		fmt.Println("Error creating Lua VM:", err)
		return
	}
	defer vm.Close() // Ensure we close the VM when done
	fmt.Println("Lua VM created successfully", vm)
	// Example of creating a Lua string
	luaString, err := vm.CreateString("Hello, Lua!")
	if err != nil {
		fmt.Println("Error creating Lua string:", err)
		return
	}
	fmt.Println("Lua string created successfully:", luaString)
	fmt.Println("Lua string as bytes:", string(luaString.Bytes()))
	fmt.Println("Lua string as bytes without nil:", luaString.Bytes())
	fmt.Println("Lua string as bytes with nil:", luaString.BytesWithNul())
	fmt.Printf("Lua string pointer: 0x%x\n", luaString.Pointer())
	luaString.Close() // Clean up the Lua string when done
	fmt.Println("Lua string as bytes after free (should be empty/nil):", luaString.Bytes())
	val := vm.DebugValue()
	fmt.Println("LuaValue:", string(val[0].(*ivm.ValueString).Value.Bytes()))
	fmt.Println("LuaValue:", string(val[1].(*ivm.ValueError).Value.Bytes()))

	time.Sleep(time.Millisecond)
}
