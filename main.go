package main

import (
	"fmt"
	"runtime"
	"time"
	"unsafe"

	"github.com/gluau/gluau/internal/callback" // Import to ensure callback package is initialized
	vmlib "github.com/gluau/gluau/vm"
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
		vmlib.CreateLuaVm()
	}

	luaVm, err := vmlib.CreateLuaVm()
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

	//defer luaVm.Drop() // Ensure we free the Lua VM when done

	// You can now use luaVm to interact with the Lua VM
	// For example, you might call methods on luaVm.lua
	// to execute Lua scripts or manipulate Lua state.

	vm, err := vmlib.CreateLuaVm()
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

	// Debug test
	val := vm.DebugValue()
	fmt.Println("LuaValue:", string(val[0].(*vmlib.ValueString).Value().Bytes()))
	fmt.Println("LuaValue:", string(val[1].(*vmlib.ValueError).Value().Bytes()))
	fmt.Println("LuaValue:", val[2].(*vmlib.ValueInteger).Value())

	// IMPORTANT
	val[0].Close()
	val[1].Close()
	val[2].Close()

	time.Sleep(time.Millisecond)

	luaEmptyString, err := vm.CreateString("")
	if err != nil {
		fmt.Println("Error creating Lua string:", err)
		return
	}
	fmt.Println("Lua empty string created successfully:", luaEmptyString)
	fmt.Println("Lua empty string as bytes:", luaEmptyString.Bytes())
	fmt.Println("Lua empty string as bytes with nil:", luaEmptyString.BytesWithNul())
	fmt.Printf("Lua empty string pointer: 0x%x\n", luaEmptyString.Pointer())
	luaEmptyString.Close() // Clean up the Lua empty string when done
	fmt.Println("Lua empty string as bytes after free (should be empty/nil):", luaEmptyString.Bytes())

	// Create a Lua table
	if err := vm.SetMemoryLimit(100000000000000); err != nil {
		panic(fmt.Sprintf("Failed to set memory limit: %v", err))
	}
	luaTable, err := vm.CreateTableWithCapacity(100000000, 10)
	if err != nil {
		fmt.Println("Error creating Lua table:", err)
	} else {
		fmt.Println("Lua table created successfully:", luaTable)
		panic("this should never happen (table overflow expected)")
	}
	defer luaTable.Close() // Ensure we close the Lua table when done

	luaTable2, err := vm.CreateTableWithCapacity(10000, 10)
	if err != nil {
		panic(err)
	}
	fmt.Println("Lua table created successfully:", luaTable2)
	defer luaTable2.Close() // Ensure we close the Lua table when done
	if err := luaTable2.Clear(); err != nil {
		panic(fmt.Sprintf("Failed to clear Lua table: %v", err))
	}
	fooStr, err := vm.CreateString("foo")
	if err != nil {
		panic(fmt.Sprintf("Failed to create Lua string: %v", err))
	}
	defer fooStr.Close() // Ensure we close the Lua string when done
	containsKey, err := luaTable2.ContainsKey(fooStr.ToValue())
	if err != nil {
		panic(fmt.Sprintf("Failed to check if Lua table contains key: %v", err))
	}
	if containsKey {
		panic("Lua table should not contain 'foo' key")
	}
	fmt.Println("empty table contains 'foo'", containsKey)
}
