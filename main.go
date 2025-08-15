package main

import (
	"errors"
	"fmt"
	"runtime"
	"time"

	// Import to ensure callback package is initialized
	vmlib "github.com/gluau/gluau/vm"
)

// #include <stdlib.h>
// #include "./rustlib/rustlib.h"
import "C"

func main() {
	// Basic test to ensure the Lua VM can be created and closed properly on GC
	/*createVm := func() {
		vm, err := vmlib.CreateLuaVm()
		if err == nil {
			defer vm.Close()
		}
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
	}*/

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

	tab := val[3].(*vmlib.ValueTable).Value()
	var testKey vmlib.Value
	tab.ForEach(func(key, value vmlib.Value) error {
		if key.Type() == vmlib.LuaValueString {
			fmt.Println("Key is a LuaString:", key.(*vmlib.ValueString).Value().String())
			testKey = key
		}
		if value.Type() == vmlib.LuaValueString {
			fmt.Println("Value is a LuaString:", value.(*vmlib.ValueString).Value().String())
		} else if value.Type() == vmlib.LuaValueInteger {
			fmt.Println("Value is a LuaInteger:", value.(*vmlib.ValueInteger).Value())
		}
		fmt.Println("Key:", key, "Value:", value)
		//time.Sleep(time.Second * 20) // Simulate some processing time
		go func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("Recovered from panic in goroutine:", r)
				}
			}()
			fmt.Println("Processing key-value pair in a goroutine:", key, value)
			// Simulate some processing
			fmt.Println("Finished processing key-value pair in goroutine:", key, value)
			panic("whee")
		}()
		return nil
	})

	fmt.Println("Key is a LuaString:", testKey.(*vmlib.ValueString).Value().String())

	err = tab.ForEach(func(key, value vmlib.Value) error {
		panic("test panic")
	})
	if err == nil {
		panic("Expected error from ForEach, got nil")
	} else if err.Error() != "panic in ForEach callback: test panic" {
		panic("Expected 'panic in ForEach callback: test panic' error, got: " + err.Error())
	}
	fmt.Println("ForEach callback error:", err)

	key, err := vm.CreateString("test2")
	if err != nil {
		panic(fmt.Sprintf("Failed to create Lua string: %v", err))
	}
	gotValue, err := tab.Get(key.ToValue())
	key.Close() // Clean up the Lua string when done
	if err != nil {
		panic(fmt.Sprintf("Failed to get value from Lua table: %v", err))
	}
	if gotValue.Type() == vmlib.LuaValueString {
		fmt.Println("Got LuaValueString:", gotValue.(*vmlib.ValueString).Value().String())
	} else {
		panic(fmt.Sprintf("Expected LuaValueString, got %d", gotValue.Type()))
	}

	isEmpty := tab.IsEmpty()
	if isEmpty {
		panic("Non-empty Lua table is empty")
	}
	len, err := tab.Len()
	if err != nil {
		panic(fmt.Sprintf("Failed to get Lua table length: %v", err))
	}
	if len != 0 {
		panic("Lua table length should be 0 (as all key-value pairs so no array indices), got " + fmt.Sprint(len))
	}
	mt := tab.Metatable()
	if mt != nil {
		panic("Lua table should not have a metatable")
	}
	poppedValue, err := tab.Pop()
	if err != nil {
		panic(fmt.Sprintf("Failed to pop value from Lua table: %v", err))
	}
	if poppedValue.Type() != vmlib.LuaValueNil {
		panic(fmt.Sprintf("Expected LuaValueNil, got %d", poppedValue.Type()))
	}
	err = tab.Push(vmlib.GoString("test"))
	if err != nil {
		panic(fmt.Sprintf("Failed to push value to Lua table: %v", err))
	}
	len, err = tab.Len()
	if err != nil {
		panic(fmt.Sprintf("Failed to get Lua table length after push: %v", err))
	}
	if len != 1 {
		panic("Lua table length should be 1 after push, got " + fmt.Sprint(len))
	}
	fmt.Printf("Lua table string %s with ptr 0x%x\n", tab, tab.Pointer())

	// Create a new Lua table to act as this table's metatable
	myNewMt, err := vm.CreateTable()
	if err != nil {
		panic(fmt.Sprintf("Failed to create Lua table for metatable: %v", err))
	}
	// Set the metatable for the Lua table
	err = tab.SetMetatable(myNewMt)
	if err != nil {
		panic(fmt.Sprintf("Failed to set metatable for Lua table: %v", err))
	}
	mt = tab.Metatable()
	if mt == nil {
		panic("Lua table should have a metatable after setting it")
	}
	doesItEqual, err := mt.Equals(myNewMt)
	if err != nil {
		panic(fmt.Sprintf("Failed to check if Lua table metatable equals another: %v", err))
	}
	if !doesItEqual {
		panic("Lua table metatable does not match the one we set")
	}
	err = tab.SetMetatable(nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to unset metatable for Lua table: %v", err))
	}
	mt = tab.Metatable()
	if mt != nil {
		panic("Lua table should not have a metatable after unsetting it")
	}

	// IMPORTANT
	val[0].Close()
	val[1].Close()
	val[2].Close()
	val[3].Close()

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
	equals, err := luaTable2.Equals(luaTable2)
	if err != nil {
		panic(fmt.Sprintf("Failed to check if Lua table equals another: %v", err))
	}
	if !equals {
		panic("Lua table should equal itself")
	}
	fmt.Println("empty table equals itself", equals)

	myFunc, err := vm.CreateFunction(func(lua *vmlib.GoLuaVmWrapper, args []vmlib.Value) ([]vmlib.Value, error) {
		return []vmlib.Value{
			vmlib.GoString("Hello world"),
		}, nil
	})
	if err != nil {
		panic(err)
	}

	res, err := myFunc.Call([]vmlib.Value{vmlib.GoString("foo")})
	if err != nil {
		panic(err)
	}
	fmt.Println("Function call response", res[0].(*vmlib.ValueString).Value().String())
	defer res[0].Close()

	res, err = myFunc.Call([]vmlib.Value{vmlib.GoString("foo")})
	if err != nil {
		panic(err)
	}
	fmt.Println("Function call response", res[0].(*vmlib.ValueString).Value().String())
	defer res[0].Close()

	myFunc, err = vm.CreateFunction(func(lua *vmlib.GoLuaVmWrapper, args []vmlib.Value) ([]vmlib.Value, error) {
		return nil, errors.New(args[0].(*vmlib.ValueString).Value().String())
	})
	if err != nil {
		panic(err)
	}

	_, err = myFunc.Call([]vmlib.Value{vmlib.GoString("foo")})
	if err != nil {
		fmt.Println("function error", err)
	}
	_, err = myFunc.Call([]vmlib.Value{vmlib.NewValueVector(1, 2, 3)})
	if err != nil {
		fmt.Println("function error", err)
	}

	runtime.GC()
	runtime.GC()

	tab, err = vm.CreateTable()
	if err != nil {
		panic(fmt.Sprintf("Failed to create Lua table: %v", err))
	}
	defer tab.Close() // Ensure we close the Lua table when done
	err = tab.Set(vmlib.GoString("test"), myFunc.ToValue())
	if err != nil {
		panic(fmt.Sprintf("Failed to set value in Lua table: %v", err))
	}

	testFn, err := tab.Get(vmlib.GoString("test"))
	if err != nil {
		panic(fmt.Sprintf("Failed to get value from Lua table: %v", err))
	}
	if testFn.Type() != vmlib.LuaValueFunction {
		panic(fmt.Sprintf("Expected LuaValueFunction, got %d", testFn.Type()))
	}
}
