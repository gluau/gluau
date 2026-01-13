package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"
	"time"

	// Import to ensure callback package is initialized
	vmlib "github.com/gluau/gluau/vm"
	"github.com/gluau/gluau/vmutils"
	"github.com/gluau/gluau/vmutils/require"
)

// #include <stdlib.h>
// #include "./rustlib/rustlib.h"
import "C"

func main() {
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
	fmt.Println("Lua string as bytes without nul:", luaString.Bytes())
	fmt.Println("Lua string as bytes with nul:", luaString.BytesWithNUL())
	fmt.Printf("Lua string pointer: 0x%x\n", luaString.Pointer())
	luaString.Close() // Clean up the Lua string when done
	fmt.Println("Lua string as bytes after free (should be empty/nil):", luaString.Bytes())

	// Example of creating a Lua table
	tab, err := vm.CreateTable()
	if err != nil {
		panic(fmt.Sprintf("Failed to create Lua table: %v", err))
	}

	// Insert some values into the table
	err = tab.Set(vmlib.GoString("key1"), vmlib.GoString("value1"))
	if err != nil {
		panic(fmt.Sprintf("Failed to set value in Lua table: %v", err))
	}

	err = tab.Set(vmlib.GoString("key2"), vmlib.NewValueInteger(42))
	if err != nil {
		panic(fmt.Sprintf("Failed to set value in Lua table: %v", err))
	}

	err = tab.Set(vmlib.GoString("key3"), vmlib.NewValueVector(1, 2, 3))
	if err != nil {
		panic(fmt.Sprintf("Failed to set value in Lua table: %v", err))
	}

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
		} else if value.Type() == vmlib.LuaValueVector {
			vec := value.(*vmlib.ValueVector).Value()
			fmt.Println("Value is a LuaVector:", vec[0], vec[1], vec[2])
		} else {
			return fmt.Errorf("unexpected value type: %s", value.Type().String())
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

	isEmpty := tab.IsEmpty()
	if isEmpty {
		panic("Non-empty Lua table is empty")
	}
	tablen, err := tab.Len()
	if err != nil {
		panic(fmt.Sprintf("Failed to get Lua table length: %v", err))
	}
	if tablen != 0 {
		panic("Lua table length should be 0 (as all key-value pairs so no array indices), got " + fmt.Sprint(tablen))
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
	tablen, err = tab.Len()
	if err != nil {
		panic(fmt.Sprintf("Failed to get Lua table length after push: %v", err))
	}
	if tablen != 1 {
		panic("Lua table length should be 1 after push, got " + fmt.Sprint(tablen))
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

	// Clean up the Lua table when done
	err = tab.Close()
	if err != nil {
		fmt.Println("Error closing Lua table:", err)
		return
	}

	time.Sleep(time.Millisecond)

	luaEmptyString, err := vm.CreateString("")
	if err != nil {
		fmt.Println("Error creating Lua string:", err)
		return
	}
	fmt.Println("Lua empty string created successfully:", luaEmptyString)
	fmt.Println("Lua empty string as bytes:", luaEmptyString.Bytes())
	fmt.Println("Lua empty string as bytes with nul:", luaEmptyString.BytesWithNUL())
	fmt.Printf("Lua empty string pointer: 0x%x\n", luaEmptyString.Pointer())
	luaEmptyString.Close() // Clean up the Lua empty string when done
	fmt.Println("Lua empty string as bytes after free (should be empty/nil):", luaEmptyString.Bytes())

	// Create a Lua table
	fmt.Println("Current memory usage:", vm.UsedMemory())
	fmt.Println("Current memory limit:", vm.MemoryLimit())
	if err := vm.SetMemoryLimit(100000000000000); err != nil {
		panic(fmt.Sprintf("Failed to set memory limit: %v", err))
	}
	if vm.MemoryLimit() != 100000000000000 {
		panic(fmt.Sprintf("Expected memory limit to be 100000000000000, got %d", vm.MemoryLimit()))
	}
	fmt.Println("New memory limit set to:", vm.MemoryLimit())
	vm.Close()
	vm = vmutils.Must(vmlib.CreateLuaVm())

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

	myFunc, err := vm.CreateFunction(func(lua *vmlib.CallbackLua, args []vmlib.Value) ([]vmlib.Value, error) {
		return []vmlib.Value{
			vmlib.GoString("Hello world"),
		}, nil
	})
	if err != nil {
		panic(err)
	}

	res, err := myFunc.Call(vmlib.GoString("foo"))
	if err != nil {
		panic(err)
	}
	fmt.Println("Function call response", res[0].(*vmlib.ValueString).Value().String())
	defer res[0].Close()

	res, err = myFunc.Call(vmlib.GoString("foo"))
	if err != nil {
		panic(err)
	}
	fmt.Println("Function call response", res[0].(*vmlib.ValueString).Value().String())
	defer res[0].Close()

	myFunc, err = vm.CreateFunction(func(lua *vmlib.CallbackLua, args []vmlib.Value) ([]vmlib.Value, error) {
		return nil, errors.New(args[0].(*vmlib.ValueString).Value().String())
	})
	if err != nil {
		panic(err)
	}

	_, err = myFunc.Call(vmlib.GoString("foo"))
	if err != nil {
		fmt.Println("function error", err)
	}
	_, err = myFunc.Call(vmlib.NewValueVector(1, 2, 3))
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

	// Compiler API
	vm.SetCompilerOpts(vmlib.CompilerOpts{
		OptimizationLevel: vmlib.OptimizationLevelFull,
	})
	globTab, err := vm.CreateTable()
	if err != nil {
		panic("failed to make global table")
	}
	err = globTab.Set(vmlib.GoString("a"), vmlib.NewValueInteger(5))
	if err != nil {
		panic("failed to set a")
	}
	clonedGlobTab, err := vm.CloneValue(globTab.ToValue())
	if err != nil {
		panic("failed to clone global table: " + err.Error())
	}
	err = globTab.Set(vmlib.GoString("_G"), clonedGlobTab)
	if err != nil {
		panic("failed to set _G")
	}
	err = globTab.Set(vmlib.GoString("_G"), globTab.ToValue().Clone())
	if err != nil {
		panic("failed to set _G")
	}
	runtime.GC() // testing

	luaFunc, err := vm.LoadChunk(vmlib.ChunkOpts{
		Code: "_G.a = _G.a + 1; return _G.a",
		Env:  globTab,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Lua func: ", luaFunc)

	// Lets clone this function
	clonedFunc, err := luaFunc.DeepClone()
	if err != nil {
		panic(fmt.Sprintf("Failed to clone Lua function: %v", err))
	}
	fmt.Println("Cloned Lua function:", clonedFunc)
	myNewTab, err := vm.CreateTable()
	if err != nil {
		panic(fmt.Sprintf("Failed to create Lua table for cloned function: %v", err))
	}
	ok, err := clonedFunc.SetEnvironment(myNewTab)
	if err != nil {
		panic(fmt.Sprintf("Failed to set environment for cloned Lua function: %v", err))
	}
	if !ok {
		panic("Failed to set environment for cloned Lua function")
	}
	env, err := clonedFunc.Environment() // This will be myNewTab
	if err != nil {
		panic(fmt.Sprintf("Failed to get environment for cloned Lua function: %v", err))
	}
	if env.Pointer() != myNewTab.Pointer() {
		panic("Cloned Lua function environment does not match the one we set")
	}
	env, err = luaFunc.Environment()
	if err != nil {
		panic(fmt.Sprintf("Failed to get environment for original Lua function: %v", err))
	}
	if env.Pointer() != globTab.Pointer() {
		panic("Original Lua function environment does not match the one we set")
	}

	ret, err := luaFunc.Call()
	if err != nil {
		panic(err)
	}
	if len(ret) != 1 || ret[0].Type() != vmlib.LuaValueInteger {
		panic("ret is not a single integer")
	}
	if ret[0].(*vmlib.ValueInteger).Value() != 6 {
		panic("ret[0] must be 6")
	} else {
		fmt.Println("got ret:", ret[0].(*vmlib.ValueInteger).Value())
	}
	luaFunc.Close()

	udMt, err := vm.CreateTable()
	if err != nil {
		panic(fmt.Sprintf("Failed to create Lua table for userdata metatable: %v", err))
	}
	// Set the __type
	err = udMt.Set(vmlib.GoString("__type"), vmlib.GoString("MyUserDataType"))
	if err != nil {
		panic(fmt.Sprintf("Failed to set __type in Lua userdata metatable: %v", err))
	}

	ud, err := vm.CreateUserData("test data", udMt)
	if err != nil {
		panic(fmt.Sprintf("Failed to create Lua user data: %v", err))
	}
	fmt.Println("Lua user data created successfully:", ud)
	associatedData, err := ud.AssociatedData()
	if err != nil {
		panic(fmt.Sprintf("Failed to get associated data from Lua user data: %v", err))
	}
	fmt.Println("Associated data from Lua user data:", associatedData)
	if associatedData != "test data" {
		panic(fmt.Sprintf("Expected associated data 'test data', got '%v'", associatedData))
	}

	// Interrupt API
	vm.SetInterrupt(func(funcVm *vmlib.CallbackLua) (vmlib.VmState, error) {
		return vmlib.VmStateContinue, errors.New("test interrupt error")
	})

	// Create a Lua function that will trigger the interrupt
	luaFunc.Close() // Close the previous function to avoid memory leaks
	luaFunc, err = vm.LoadChunk(vmlib.ChunkOpts{
		Name: "test_interrupt",
		Code: "while true do end", // Infinite loop to trigger the interrupt
		Env:  globTab,
	})
	if err != nil {
		panic(err)
	}

	// Call the Lua function to trigger the interrupt
	_, err = luaFunc.Call()
	if err != nil {
		if !strings.Contains(err.Error(), "test interrupt error") {
			panic(fmt.Sprintf("Expected interrupt error, got: %v", err))
		}
		fmt.Println("Lua function call error (expected due to interrupt):", err)
	} else {
		panic("Expected an error from the Lua function call due to interrupt")
	}

	// Set a new interrupt which will yield the execution
	// after 100 milliseconds
	timeNow := time.Now()
	vm.SetInterrupt(func(funcVm *vmlib.CallbackLua) (vmlib.VmState, error) {
		if time.Since(timeNow) > 10*time.Millisecond {
			fmt.Println("Interrupt triggered after 1 second on thread with status", funcVm.CurrentThread().Status())
			return vmlib.VmStateYield, nil // Yield the execution after 100 milliseconds
		}
		return vmlib.VmStateContinue, nil // Continue execution
	})

	// Call the Lua function again as a thread to trigger the interrupt
	th := vmutils.Must(vm.CreateThread(luaFunc))
	_, err = th.Resume()
	if err != nil {
		panic("Expected graceful exit from the Lua function call due to interrupt")
	}
	if th.Status() != vmlib.ThreadStatusResumable {
		panic("vm thread not resumable")
	}

	// Thread API
	thread, err := vm.CreateThread(luaFunc)
	if err != nil {
		panic(fmt.Sprintf("Failed to create Lua thread: %v", err))
	}
	if !thread.Equals(thread) {
		panic("Expected thread to equal itself")
	}
	if thread.Equals(nil) {
		panic("Expected thread to not equal nil")
	}
	defer thread.Close() // Ensure we close the Lua thread when done
	fmt.Println("Lua thread created successfully:", thread)

	// Resume the thread with no arguments
	//
	// As this is now a thread that is not main thread, this wont error
	// with a yield across metamethod/C-call boundary error anymore
	_, err = thread.Resume()
	if err != nil {
		panic(fmt.Sprintf("Failed to resume Lua thread: %v", err))
	}
	fmt.Println("Lua thread resumed successfully, returned values:", res)

	thread.Close()

	// Test with erroring interrupt
	vm.SetInterrupt(func(funcVm *vmlib.CallbackLua) (vmlib.VmState, error) {
		return vmlib.VmStateContinue, errors.New("test interrupt error")
	})

	thread, err = vm.CreateThread(luaFunc)
	if err != nil {
		panic(fmt.Sprintf("Failed to create Lua thread: %v", err))
	}
	defer thread.Close() // Ensure we close the Lua thread when done
	fmt.Println("Lua thread 2 created successfully:", thread)

	// Resume the thread with no arguments
	_, err = thread.Resume()
	if err != nil {
		if !strings.Contains(err.Error(), "test interrupt error") {
			panic(fmt.Sprintf("Expected interrupt error, got: %v", err))
		}
		fmt.Println("Lua thread call error (expected due to interrupt):", err)
	} else {
		panic("Expected an error from the Lua thread call due to interrupt")
	}

	thread.Close()
	if thread.Equals(thread) {
		panic("expected closing thread to not equal itself")
	}
	luaFunc.Close() // Close the Lua function to avoid memory leaks

	luaFunc2, err := vm.LoadChunk(vmlib.ChunkOpts{
		Name: "test_yield_2",
		Code: "local yielder = ...; yielder(); return 1", // This will yield the execution
		Env:  vm.Globals(),
	})

	if err != nil {
		panic(fmt.Sprintf("Failed to load Lua chunk for yielding: %v", err))
	}

	if luaFunc2.Equals(luaFunc) {
		panic("Expected luaFunc2 to be a different function than luaFunc")
	} else if !luaFunc2.Equals(luaFunc2) {
		panic("Expected luaFunc2 to equal itself")
	}

	defer luaFunc2.Close() // Ensure we close the Lua function when done

	// A simple yielding test
	vm.RemoveInterrupt()
	thread2, err := vm.CreateThread(luaFunc2)
	if err != nil {
		panic(fmt.Sprintf("Failed to create Lua thread: %v", err))
	}
	defer thread2.Close() // Ensure we close the Lua thread when done
	fmt.Println("Lua thread 3 created successfully:", thread2)
	yieldFunc, err := vm.CreateFunction(func(lua *vmlib.CallbackLua, args []vmlib.Value) ([]vmlib.Value, error) {
		lua.YieldWith([]vmlib.Value{vmlib.GoString("yielded value")})
		return []vmlib.Value{}, nil
	})

	if err != nil {
		panic(fmt.Sprintf("Failed to create Lua yield function: %v", err))
	}
	defer yieldFunc.Close() // Ensure we close the Lua function when done
	res, err = thread2.Resume(yieldFunc.ToValue())
	if err != nil {
		panic(fmt.Sprintf("Failed to resume Lua thread with yield function: %v", err))
	}
	fmt.Println("Lua thread resumed successfully with yield function, returned values:", res[0])
	if res[0].Type() != vmlib.LuaValueString {
		panic(fmt.Sprintf("Expected LuaValueString, got %d", res[0].Type()))
	}
	if res[0].(*vmlib.ValueString).Value().String() != "yielded value" {
		panic(fmt.Sprintf("Expected 'yielded value', got '%s'", res[0].(*vmlib.ValueString).Value().String()))
	}

	if thread2.Status() != vmlib.ThreadStatusResumable {
		panic(fmt.Sprintf("Expected thread status to be running, got %s", thread2.Status().String()))
	}

	// Resume the thread again to finish it
	res, err = thread2.Resume()
	if err != nil {
		panic(fmt.Sprintf("Failed to resume Lua thread after yield: %v", err))
	}
	fmt.Println("Lua thread resumed successfully after yield, returned values:", res[0])
	if res[0].Type() != vmlib.LuaValueInteger {
		panic(fmt.Sprintf("Expected LuaValueInteger, got %d", res[0].Type()))
	}
	if res[0].(*vmlib.ValueInteger).Value() != 1 {
		panic(fmt.Sprintf("Expected 1, got %d", res[0].(*vmlib.ValueInteger).Value()))
	}

	luaFunc.Close()  // Close the previous function to avoid memory leaks
	luaFunc2.Close() // Close the previous function to avoid memory leaks

	// Test set type metatable
	vm2, err := vmlib.CreateLuaVm()
	if err != nil {
		panic(err)
	}
	myTypeMt, err := vm2.CreateTable()
	if err != nil {
		panic(err)
	}
	vmutils.MustOk(
		myTypeMt.Set(
			vmlib.GoString("__tostring"),
			vmutils.Must(
				vm2.CreateFunction(func(funcVm *vmlib.CallbackLua, args []vmlib.Value) ([]vmlib.Value, error) {
					fmt.Println("test")
					return []vmlib.Value{vmlib.GoString("hello")}, nil
				}),
			).ToValue(),
		),
	)
	vmutils.MustOk(vm2.SetTypeMetatable(vmlib.TypeMetatableTypeBool, myTypeMt))

	luaFunc, err = vm2.LoadChunk(vmlib.ChunkOpts{
		Name: "test_typemt",
		Code: "local b = true; return tostring(b)",
	})
	if err != nil {
		panic(err)
	}
	res = vmutils.Must(luaFunc.Call())
	if res[0].(*vmlib.ValueString).Value().String() != "hello" {
		panic("type metatable set failed")
	}

	vm2.Close()

	// Test registry API
	vm3, err := vmlib.CreateLuaVm()
	if err != nil {
		panic(err)
	}

	val := vmutils.Must(vm3.RegistryValue("test"))
	if ok, _ := val.Equals(&vmlib.ValueNil{}); !ok {
		panic("val is not nil")
	}
	vmutils.MustOk(vm3.SetRegistryValue("", vmlib.GoString("foo")))
	val = vmutils.Must(vm3.RegistryValue("test"))
	if ok, _ := val.Equals(&vmlib.ValueNil{}); !ok {
		panic("val is not nil")
	}
	val = vmutils.Must(vm3.RegistryValue(""))
	if ok, _ := val.Equals(vmlib.GoString("foo")); !ok {
		panic("val is not foo")
	}

	vmutils.MustOk(vm3.SetRegistryValue("test", vmlib.GoString("foo")))
	val = vmutils.Must(vm3.RegistryValue("test"))
	if ok, _ := val.Equals(vmlib.GoString("foo")); !ok {
		panic("val is not foo")
	}

	vmutils.MustOk(vm3.RemoveRegistryValue("test"))
	val = vmutils.Must(vm3.RegistryValue("test"))
	if ok, _ := val.Equals(&vmlib.ValueNil{}); !ok {
		panic("val is not nil")
	}

	valuePassthroughFn, err := vm3.LoadChunk(vmlib.ChunkOpts{
		Code: "local v = ...; return v",
	})
	if err != nil {
		panic(err)
	}

	// Buffer
	buffer := vmutils.Must(vm3.CreateBuffer([]byte("test buffer data")))
	if string(buffer.Bytes()) != "test buffer data" {
		panic("buffer bytes mismatch")
	}
	vmutils.MustOk(buffer.WriteBytes(0, []byte("ob")))
	if string(buffer.Bytes()) != "obst buffer data" {
		panic("buffer bytes mismatch after write")
	}
	bufferPtr := buffer.Pointer()
	fmt.Println("Lua buffer created successfully:", buffer)
	res, err = valuePassthroughFn.Call(buffer.ToValue()) // Takes ownership of the buffer (hence the above bufferPtr)
	if err != nil {
		panic(fmt.Sprintf("Failed to call Lua function with buffer: %v", err))
	}
	if len(res) != 1 || res[0].Type() != vmlib.LuaValueBuffer {
		panic(fmt.Sprintf("Expected LuaValueBuffer, got %d", res[0].Type()))
	}
	fmt.Println("Lua buffer passed through function successfully:", res[0].(*vmlib.ValueBuffer).Value().String())
	if res[0].(*vmlib.ValueBuffer).Value().Pointer() != bufferPtr {
		panic("Returned buffer pointer does not match original buffer pointer")
	}

	// String
	luaString = vmutils.Must(vm3.CreateString("test string data"))
	luaStringPtr := luaString.Pointer()
	fmt.Println("Lua string created successfully:", luaString)
	res, err = valuePassthroughFn.Call(luaString.ToValue()) // Takes ownership of the string (hence the above luaStringPtr)
	if err != nil {
		panic(fmt.Sprintf("Failed to call Lua function with string: %v", err))
	}
	if len(res) != 1 || res[0].Type() != vmlib.LuaValueString {
		panic(fmt.Sprintf("Expected LuaValueString, got %d", res[0].Type()))
	}
	fmt.Println("Lua string passed through function successfully:", res[0].(*vmlib.ValueString).Value().String())
	if res[0].(*vmlib.ValueString).Value().Pointer() != luaStringPtr {
		panic("Returned string pointer does not match original string pointer")
	}

	// Table
	luaTable = vmutils.Must(vm3.CreateTable())
	luaTablePtr := luaTable.Pointer()
	fmt.Println("Lua table created successfully:", luaTable)
	res, err = valuePassthroughFn.Call(luaTable.ToValue()) // Takes ownership of the table (hence the above luaTablePtr)
	if err != nil {
		panic(fmt.Sprintf("Failed to call Lua function with table: %v", err))
	}
	if len(res) != 1 || res[0].Type() != vmlib.LuaValueTable {
		panic(fmt.Sprintf("Expected LuaValueTable, got %d", res[0].Type()))
	}
	fmt.Println("Lua table passed through function successfully:", res[0].(*vmlib.ValueTable).Value().String())
	if res[0].(*vmlib.ValueTable).Value().Pointer() != luaTablePtr {
		panic("Returned table pointer does not match original table pointer")
	}

	// UserData
	ud = vmutils.Must(vm3.CreateUserData("test userdata", res[0].(*vmlib.ValueTable).Value()))
	udPtr := ud.Pointer()
	fmt.Println("Lua user data created successfully:", ud)
	res, err = valuePassthroughFn.Call(ud.ToValue()) // Takes ownership of the user data (hence the above udPtr)
	if err != nil {
		panic(fmt.Sprintf("Failed to call Lua function with user data: %v", err))
	}
	if len(res) != 1 || res[0].Type() != vmlib.LuaValueUserData {
		panic(fmt.Sprintf("Expected LuaValueUserData, got %d", res[0].Type()))
	}
	fmt.Println("Lua user data passed through function successfully:", res[0].(*vmlib.ValueUserData).Value())
	if res[0].(*vmlib.ValueUserData).Value().Pointer() != udPtr {
		panic("Returned user data pointer does not match original user data pointer")
	}

	vm3.Close()

	fmt.Println("testing require")

	// Require API
	vm4, err := vmlib.CreateLuaVm()
	if err != nil {
		panic(err)
	}

	fmt.Println("Lua VM created successfully for require test", vm4)

	mapFs := NewMapFs(map[string]string{
		"init.luau":               "",
		"test.luau":               "return require('./foo/test')",
		"foo/test.luau":           "return require('./test2')",
		"foo/test2.luau":          "return require('./doo/test2')",
		"foo/doo/test2.luau":      "return require('@dir-alias/bar')",
		"foo/dir-alias/bar.luau":  "return require('./baz')",
		"foo/dir-alias/baz.luau":  "return require('@dir-alias/bat')",
		"foo/dir-alias/bat.luau":  "return require('./baz2')",
		"foo/dir-alias/baz2.luau": "return require('../commacomma')",
		"foo/commacomma.luau":     "return require('./commacomma2')",
		"foo/commacomma2.luau":    "return require('../roothelper')",
		"roothelper.luau":         "return require('./roothelper2')",
		"roothelper2.luau":        "return require('@dir-alias-2/baz')",
		"dogs/2/baz.luau":         "return require('../../nextluaurcarea/baz')",
		"nextluaurcarea/baz.luau": "return require('@dir-alias-2/chainy')",
		"dogs/3/chainy.luau":      "return 3",
		".luaurc": `{
			"aliases": {
				"dir-alias": "./foo/dir-alias",
				"dir-alias-2": "./dogs/2",
			}
		}`,
		"nextluaurcarea/.luaurc": `{
			"aliases": {
				"dir-alias": "../foo/dir-alias",
				"dir-alias-2": "../dogs/3"
			}
		}`,
	})
	simpleRequirer := require.NewSimpleRequirer("requireCachePrefix", vm4.Globals(), require.NewUnixVfs(mapFs), true)
	requireFunc, err := vm4.CreateRequireFunction(simpleRequirer)
	if err != nil {
		panic(fmt.Sprintf("Failed to create require function: %v", err))
	}

	fmt.Println("Require function created successfully:", requireFunc)

	vm4.Globals().Set(vmlib.GoString("require"), requireFunc.ToValue()) // This consumes requireFunc so no need to explictly close it

	chunkFunc, err := vm4.LoadChunk(vmlib.ChunkOpts{
		Name: "/",
		Code: "return require('@self/test')",
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to load chunk for require test: %v", err))
	}
	res, err = chunkFunc.Call()
	if err != nil {
		panic(fmt.Sprintf("Failed to call chunk for require test: %v", err))
	}
	if len(res) != 1 {
		panic("expected 1 result from require test, got " + fmt.Sprint(len(res)))
	}
	if ok, _ := res[0].Equals(vmlib.NewValueInteger(3)); !ok {
		panic("expected require test to return 3, got " + res[0].String())
	}

	vm4.Close() // Ensure we close the VM when done
}

// NewMapFs returns a new FileSystem from the provided map.
// Map keys must be forward slash-separated paths with
// no leading slash, such as "file1.txt" or "dir/file2.txt".
// New panics if any of the paths contain a leading slash.
func NewMapFs(m map[string]string) fs.ReadDirFS {
	// Verify all provided paths are relative before proceeding.
	var pathsWithLeadingSlash []string
	for p := range m {
		if strings.HasPrefix(p, "/") {
			pathsWithLeadingSlash = append(pathsWithLeadingSlash, p)
		}
	}
	if len(pathsWithLeadingSlash) > 0 {
		panic(fmt.Errorf("mapfs.New: invalid paths with a leading slash: %q", pathsWithLeadingSlash))
	}

	return mapFS(m)
}

// mapFS is the map based implementation of FileSystem
//
// Used in require impl
type mapFS map[string]string

func (fs mapFS) Close() error { return nil }

func filename(p string) string {
	return strings.TrimPrefix(p, "/")
}

func (fs mapFS) Open(p string) (fs.File, error) {
	p = pathFix(p)
	b, ok := fs[filename(p)]
	if !ok {
		return nil, os.ErrNotExist
	}
	return nopCloser{strings.NewReader(b)}, nil
}

func fileInfo(name, contents string) mapFI {
	return mapFI{name: path.Base(name), size: len(contents)}
}

func dirInfo(name string) mapFI {
	return mapFI{name: path.Base(name), dir: true}
}

// slashdir returns path.Dir(p), but special-cases paths not beginning
// with a slash to be in the root.
func slashdir(p string) string {
	d := path.Dir(p)
	if d == "." {
		return "/"
	}
	if strings.HasPrefix(p, "/") {
		return d
	}
	return "/" + d
}

func pathFix(path string) string {
	if strings.HasPrefix(path, "./") {
		return "/" + strings.TrimPrefix(path, "./")
	} else if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

func (fs mapFS) ReadDir(p string) ([]fs.DirEntry, error) {
	p = pathFix(p)
	p = path.Clean(p)
	var ents []string
	fim := make(map[string]mapFI) // base -> fi
	for fn, b := range fs {
		dir := slashdir(fn)
		isFile := true
		var lastBase string
		for {
			if dir == p {
				base := lastBase
				if isFile {
					base = path.Base(fn)
				}
				if _, ok := fim[base]; !ok {
					var fi mapFI
					if isFile {
						fi = fileInfo(fn, b)
					} else {
						fi = dirInfo(base)
					}
					ents = append(ents, base)
					fim[base] = fi
				}
			}
			if dir == "/" {
				break
			} else {
				isFile = false
				lastBase = path.Base(dir)
				dir = path.Dir(dir)
			}
		}
	}
	if len(ents) == 0 {
		return nil, os.ErrNotExist
	}

	sort.Strings(ents)
	return createDirEntry(ents, fim), nil
}

func createDirEntry(ents []string, fim map[string]mapFI) []fs.DirEntry {
	var list []fs.DirEntry
	for _, dir := range ents {
		list = append(list, fim[dir])
	}
	return list
}

// mapFI is the map-based implementation of FileInfo.
type mapFI struct {
	name string
	size int
	dir  bool
}

func (fi mapFI) IsDir() bool        { return fi.dir }
func (fi mapFI) ModTime() time.Time { return time.Time{} }
func (fi mapFI) Mode() os.FileMode {
	if fi.IsDir() {
		return 0755 | os.ModeDir
	}
	return 0444
}
func (fi mapFI) Name() string { return path.Base(fi.name) }
func (fi mapFI) Size() int64  { return int64(fi.size) }
func (fi mapFI) Sys() any     { return nil }
func (fi mapFI) Info() (os.FileInfo, error) {
	return fi, nil
}
func (fi mapFI) Type() fs.FileMode {
	if fi.IsDir() {
		return fs.ModeDir
	}
	return 0
}

type nopCloser struct {
	io.ReadSeeker
}

func (nc nopCloser) Close() error { return nil }
func (nc nopCloser) Stat() (os.FileInfo, error) {
	return nil, fs.ErrInvalid
}
