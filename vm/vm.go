package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"runtime/debug"
	"unsafe"
)

var luaVmTab = objectTab{
	dtor: func(ptr *C.void) {
		C.freeluavm((*C.struct_Lua)(unsafe.Pointer(ptr)))
	},
}

// A handle to the Lua VM.
type Lua struct {
	object *object
}

// Returns the string representation of the Lua VM.
func (l *Lua) String() string {
	if l == nil || l.object == nil {
		return "<nil Lua VM>"
	}
	pt := l.MainThread().Pointer()
	if pt == 0 {
		return "<closed Lua VM>"
	}
	return fmt.Sprintf("Lua VM: 0x%x", pt)
}

func (l *Lua) ptr() *C.struct_Lua {
	if l.object.ptr == nil {
		panic("attempted to access closed Lua VM")
	}
	return (*C.struct_Lua)(unsafe.Pointer(l.object.ptr))
}

func (l *Lua) lua() (*C.struct_Lua, error) {
	ptr, err := l.object.PointerNoLock()
	if err != nil {
		return nil, err // Return error if the object is closed
	}
	return (*C.struct_Lua)(unsafe.Pointer(ptr)), nil
}

// SetCompilerOpts sets the default compiler options for the Lua VM.
//
// This is a Luau-specific feature
func (l *Lua) SetCompilerOpts(opts CompilerOpts) {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return // No-op if the Lua VM is closed
	}

	cOpts := opts.toC()
	C.luavm_setcompileropts(lua, cOpts)
}

// SetMemoryLimit sets the memory limit for the Lua VM.
//
// Upon exceeding this limit, Luau will return a memory error
// back to the caller (which may either be in Luau still or in Go
// as a error value).
func (l *Lua) SetMemoryLimit(limit int) error {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return err
	}
	res := C.luavm_setmemorylimit(lua, C.size_t(limit))
	if res.error != nil {
		err := moveErrorToGo(res.error)
		return err
	}
	return nil
}

// UsedMemory returns the amount of memory used by the Lua VM.
func (l *Lua) UsedMemory() int {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return 0 // Return 0 if the Lua VM is closed
	}

	used := C.luago_used_memory(lua)
	return int(used)
}

// MemoryLimit returns the memory limit set for the Lua VM.
func (l *Lua) MemoryLimit() int {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return 0 // Return 0 if the Lua VM is closed
	}

	limit := C.luago_memory_limit(lua)
	return int(limit)
}

type TypeMetatableType uint8

const (
	TypeMetatableTypeBool TypeMetatableType = iota
	TypeMetatableTypeLightUserData
	TypeMetatableTypeNumber
	TypeMetatableTypeVector
	TypeMetatableTypeString
	TypeMetatableTypeFunction
	TypeMetatableTypeThread
	TypeMetatableTypeBuffer
)

// SetTypeMetatable sets the metatable for a Lua builtin type.
//
// - The metatable will be shared by all values of the given type.
// - mt can be set to nil to remove the metatable
func (l *Lua) SetTypeMetatable(typ TypeMetatableType, mt *LuaTable) error {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return err
	}

	if mt != nil {
		if mt.lua != l {
			return fmt.Errorf("cannot create userdata with metatable from different Lua instance")
		}

		return withBaseRefNoRet(mt.BaseRef, func(ptr C.struct_GoLuaValueV2) error {
			C.luago_set_type_metatable(lua, C.uint8_t(typ), ptr)
			return nil
		})
	}

	C.luago_set_type_metatable(lua, C.uint8_t(typ), nullCValueV2())
	return nil
}

// SetRegistryValue sets a value on the Lua registry with a given key name
func (l *Lua) SetRegistryValue(key string, value any) error {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return err
	}

	valueVal, err, _ := valueToC(value)

	if err != nil {
		return err // Return error if the value cannot be converted (diff lua state, closed object, etc)
	}

	if len(key) == 0 {
		res := C.luago_set_named_registry_value(lua, (*C.char)(nil), 0, valueVal)
		if res.error != nil {
			err := moveErrorToGo(res.error)
			return err
		}
		return nil
	}
	keyBytes := []byte(key)
	res := C.luago_set_named_registry_value(lua, (*C.char)(unsafe.Pointer(&keyBytes[0])), C.size_t(len(key)), valueVal)
	if res.error != nil {
		err := moveErrorToGo(res.error)
		return err
	}
	return nil
}

// RegistryValue returns a value on the Lua registry with a given key name
func (l *Lua) RegistryValue(key string) (any, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	if len(key) == 0 {
		res := C.luago_named_registry_value(lua, (*C.char)(nil), 0)
		if res.error != nil {
			err := moveErrorToGo(res.error)
			return nil, err
		}
		return castValue(l, res.value), nil
	}
	keyBytes := []byte(key)
	res := C.luago_named_registry_value(lua, (*C.char)(unsafe.Pointer(&keyBytes[0])), C.size_t(len(key)))
	if res.error != nil {
		err := moveErrorToGo(res.error)
		return nil, err
	}
	return castValue(l, res.value), nil
}

// RemoveRegistryValue removes a value on the Lua registry with a given key name
//
// Equivalent to SetRegistryValue with value of nil
func (l *Lua) RemoveRegistryValue(key string) error {
	return l.SetRegistryValue(key, nil)
}

// Sandbox enables or disables the sandbox mode for the Luau VM.
//
// This method, in particular:
//
// - Set all libraries to read-only
// - Set all builtin metatables to read-only
// - Set globals to read-only (and activates safeenv)
// - Setup local environment table that performs writes locally and proxies reads to the global environment.
// - Allow only count mode in collectgarbage function.
//
// Note that this is a Luau-specific feature.
func (l *Lua) Sandbox(enabled bool) error {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return err
	}
	res := C.luavm_sandbox(lua, C.bool(enabled))
	if res.error != nil {
		err := moveErrorToGo(res.error)
		return err
	}
	return nil
}

// Globals returns the global environment table of the Lua VM.
func (l *Lua) Globals() *LuaTable {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil
	}
	globals := C.luago_globals(lua)
	tab, ok := castValue(l, globals).(*LuaTable)
	if !ok {
		return nil
	}

	return tab
}

// SetGlobals sets the global environment table of the Lua VM.
//
// Note that any existing Lua functions have cached global environment and will not see the changes made by this method.
//
// To update the environment for existing Lua functions, use LuaFunction.SetEnvironment
func (l *Lua) SetGlobals(tab *LuaTable) error {
	if tab.lua != l {
		return fmt.Errorf("cannot set globals from different Lua instance")
	}

	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil
	}
	if tab == nil {
		return errors.New("globals table cannot be nil")
	}
	return withBaseRefNoRet(tab.BaseRef, func(tabptr C.struct_GoLuaValueV2) error {
		res := C.luago_setglobals(lua, tabptr)
		if res.error != nil {
			err := moveErrorToGo(res.error)
			return err
		}
		return nil
	})
}

type VmState int

const (
	VmStateContinue VmState = iota
	VmStateYield            // Yield the VM execution / stop execution
)

type InterruptFn func(funcVm *CallbackLua) (VmState, error)

// Sets an interrupt function that will periodically be called by Luau VM.
//
// Any Luau code is guaranteed to call this handler “eventually” (in practice this can happen at any function call or at any loop iteration).
//
// The provided interrupt function can error, and this error will be propagated through the Luau code that was executing at the time the interrupt was triggered.
//
// Also this can be used to implement continuous execution limits by instructing Luau VM to yield by returning VmState::Yield.
func (l *Lua) SetInterrupt(callback InterruptFn) {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return
	}

	cbWrapper := newGoCallback(func(val unsafe.Pointer) {
		cval := (*C.struct_InterruptData)(val)

		// Safety: it is undefined behavior for the callback to unwind into
		// Rust (or even C!) frames from Go, so we must recover() any panic
		// that occurs in the callback to prevent a crash.
		defer func() {
			if r := recover(); r != nil {
				// Deallocate any existing error
				if cval.error != nil {
					freeRustString(cval.error)
				}

				// Replace
				errv := moveStringToRust(fmt.Sprintf("panic in interrupt callback: %v", r))
				cval.error = errv // Rust side will deallocate it for us
			}
		}()

		callbackVm := &Lua{object: newObject((*C.void)(unsafe.Pointer(cval.lua)), luaVmTab)}
		defer callbackVm.Close() // Free the memory associated with the callback VM. TODO: Maybe switch to using a Deref API instead of Close?

		cbLua := &CallbackLua{
			mainstate: l,          // The main Lua VM that owns this callback
			cbstate:   callbackVm, // The callback Lua VM that is used to execute the callback
		}

		vmState, err := callback(cbLua)

		if err != nil {
			errv := moveStringToRust(err.Error())
			cval.error = errv // Rust side will deallocate it for us
			return
		}

		cval.vm_state = C.uint8_t(vmState)
	}, func() {
		fmt.Println("interrupt callback is being dropped")
	})

	C.luago_set_interrupt(lua, cbWrapper.ToC())
}

// Removes the interrupt function set by SetInterrupt.
func (l *Lua) RemoveInterrupt() {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return // No-op if the Lua VM is closed
	}

	C.luago_remove_interrupt(lua)
}

// Returns the main thread of the Lua VM.
//
// Note: if you want the currently running thread from a callback, use CallbackLua.CurrentThread() instead.
func (l *Lua) MainThread() *LuaThread {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil // Return nil if the Lua VM is closed
	}

	thread := C.luago_current_thread(lua)
	threadVal, ok := castValue(l, thread).(*LuaThread)
	if !ok {
		return nil
	}
	return threadVal
}

// CreateString creates a Lua string from a Go string.
func (l *Lua) CreateString(s string) (*LuaString, error) {
	return l.createString([]byte(s))
}

// CreateString creates a Lua string from a Go string.
func (l *Lua) MustCreateString(s string) *LuaString {
	st, err := l.createString([]byte(s))

	if err != nil {
		panic(fmt.Sprintf("failed to create Lua string: %v", err))
	}

	return st
}

// CreateStringBytes creates a Lua string from a byte slice.
// This is useful for creating strings from raw byte data.
func (l *Lua) CreateStringBytes(s []byte) (*LuaString, error) {
	return l.createString(s)
}

func (l *Lua) createString(s []byte) (*LuaString, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	if len(s) == 0 {
		// Passing nil to luago_create_string creates an empty string.
		res := C.luago_create_string(lua, (*C.char)(nil), C.size_t(len(s)))
		if res.error != nil {
			return nil, moveErrorToGo(res.error)
		}
		str, ok := castValue(l, res.value).(*LuaString)
		if !ok {
			return nil, fmt.Errorf("expected LuaString from string creation, got %T", str)
		}
		return str, nil
	}

	res := C.luago_create_string(lua, (*C.char)(unsafe.Pointer(&s[0])), C.size_t(len(s)))
	if res.error != nil {
		return nil, moveErrorToGo(res.error)
	}
	str, ok := castValue(l, res.value).(*LuaString)
	if !ok {
		return nil, fmt.Errorf("expected LuaString from string creation, got %T", str)
	}
	return str, nil
}

// CreateTable creates a new Lua table.
func (l *Lua) CreateTable() (*LuaTable, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	res := C.luago_create_table(lua)
	if res.error != nil {
		err := moveErrorToGo(res.error)
		return nil, err
	}
	tab, ok := castValue(l, res.value).(*LuaTable)
	if !ok {
		return nil, fmt.Errorf("expected LuaTable from table creation, got %T", tab)
	}
	return tab, nil
}

// CreateTableWithCapacity creates a new Lua table with specified capacity for array and record parts.
// with narr as the number of array elements and nrec as the number of record elements.
func (l *Lua) CreateTableWithCapacity(narr, nrec int) (*LuaTable, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	res := C.luago_create_table_with_capacity(lua, C.size_t(narr), C.size_t(nrec))
	if res.error != nil {
		err := moveErrorToGo(res.error)
		return nil, err
	}
	tab, ok := castValue(l, res.value).(*LuaTable)
	if !ok {
		return nil, fmt.Errorf("expected LuaTable from table creation, got %T", tab)
	}
	return tab, nil
}

type FunctionFn func(funcVm *CallbackLua, args []any) ([]any, error)

// CreateFunction creates a new Function
//
// # Note that funcVm will only be open until the callback function returns
func (l *Lua) CreateFunction(callback FunctionFn) (*LuaFunction, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	cbWrapper := newGoCallback(func(val unsafe.Pointer) {
		cval := (*C.struct_FunctionCallbackData)(val)

		// Safety: it is undefined behavior for the callback to unwind into
		// Rust (or even C!) frames from Go, so we must recover() any panic
		// that occurs in the callback to prevent a crash.
		defer func() {
			if r := recover(); r != nil {
				// Deallocate any existing error
				if cval.error != nil {
					freeRustString(cval.error)
				}

				// Replace
				errv := moveStringToRust(fmt.Sprintf("panic in CreateFunction callback: %v %v", r, string(debug.Stack())))
				cval.error = errv // Rust side will deallocate it for us
			}
		}()

		mw := copyAndFreeMultiValueArray(l, cval.args)

		callbackVm := &Lua{object: newObject((*C.void)(unsafe.Pointer(cval.lua)), luaVmTab)}
		//defer callbackVm.Close() // Free the memory associated with the callback VM. TODO: Maybe switch to using a Deref API instead of Close?

		cbLua := &CallbackLua{
			mainstate: l,          // The main Lua VM that owns this callback
			cbstate:   callbackVm, // The callback Lua VM that is used to execute the callback
		}

		values, err := callback(cbLua, mw)

		if err != nil {
			errv := moveStringToRust(err.Error())
			cval.error = errv // Rust side will deallocate it for us
			return
		}

		outmw, err := createMultiValue(values)
		if err != nil {
			errv := moveStringToRust(err.Error())
			cval.error = errv // Rust side will deallocate it for us
			return
		}

		cval.values = outmw
	}, func() {
		fmt.Println("function callback is being dropped")
	})

	res := C.luago_create_function(lua, cbWrapper.ToC())
	if res.error != nil {
		err := moveErrorToGo(res.error)
		return nil, err
	}
	fn, ok := castValue(l, res.value).(*LuaFunction)
	if !ok {
		return nil, fmt.Errorf("expected LuaFunction from function creation, got %T", fn)
	}
	return fn, nil
}

// CreateThread creates a new thread from a LuaFunction
//
// Locking behavior: Takes a read-lock on the LuaFunction object
// and the Lua VM object
func (l *Lua) CreateThread(fn *LuaFunction) (*LuaThread, error) {
	if fn == nil {
		return nil, fmt.Errorf("function cannot be nil")
	}

	if fn.lua != l {
		return nil, fmt.Errorf("cannot create thread from different Lua instance")
	}

	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	return withBaseRef(fn.BaseRef, func(ptr C.struct_GoLuaValueV2) (*LuaThread, error) {
		res := C.luago_create_thread(lua, ptr)
		if res.error != nil {
			err := moveErrorToGo(res.error)
			return nil, err
		}
		th, ok := castValue(l, res.value).(*LuaThread)
		if !ok {
			return nil, fmt.Errorf("expected LuaThread from create thread, got %T", th)
		}
		return th, nil
	})
}

// CreateBuffer creates a LuaBuffer from a byte slice.
func (l *Lua) CreateBuffer(s []byte) (*LuaBuffer, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	if len(s) == 0 {
		// Passing nil to luago_create_buffer creates an empty buffer.
		res := C.luago_create_buffer(lua, (*C.char)(nil), C.size_t(len(s)))
		if res.error != nil {
			return nil, moveErrorToGo(res.error)
		}
		buf, ok := castValue(l, res.value).(*LuaBuffer)
		if !ok {
			return nil, fmt.Errorf("expected LuaBuffer from create buffer, got %T", buf)
		}
		return buf, nil
	}

	res := C.luago_create_buffer(lua, (*C.char)(unsafe.Pointer(&s[0])), C.size_t(len(s)))
	if res.error != nil {
		return nil, moveErrorToGo(res.error)
	}
	buf, ok := castValue(l, res.value).(*LuaBuffer)
	if !ok {
		return nil, fmt.Errorf("expected LuaBuffer from create buffer, got %T", buf)
	}
	return buf, nil
}

// LoadChunk loads a Lua chunk from the given options.
func (l *Lua) LoadChunk(opts ChunkOpts) (*LuaFunction, error) {
	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	var env = nullCValueV2()
	if opts.Env != nil {
		if opts.Env.lua != l {
			return nil, fmt.Errorf("cannot set environment table from different Lua instance")
		}
		env, err, _ = valueToC(opts.Env)
		if err != nil {
			return nil, err // Return error if the environment table is closed
		}
	}

	var compilerOpts *C.struct_CompilerOpts = nil
	if opts.CompilerOpts != nil {
		compilerOptsC := opts.CompilerOpts.toC()
		compilerOpts = &compilerOptsC
	}

	var name = newChunkString([]byte(opts.Name))
	var code = newChunkString([]byte(opts.Code))

	res := C.luago_load_chunk(
		lua,
		C.struct_ChunkOpts{
			name:          name,
			env:           env,
			mode:          C.uint8_t(opts.Mode),
			compiler_opts: compilerOpts,
			code:          code,
		},
	)

	if res.error != nil {
		err := moveErrorToGo(res.error)
		return nil, err
	}

	fn, ok := castValue(l, res.value).(*LuaFunction)
	if !ok {
		return nil, fmt.Errorf("expected LuaFunction from load chunk, got %T", fn)
	}
	return fn, nil
}

// CreateUserData creates a LuaUserData with associated data and a metatable.
func (l *Lua) CreateUserData(associatedData any, mt *LuaTable) (*LuaUserData, error) {
	if mt == nil {
		return nil, fmt.Errorf("metatable cannot be nil")
	}
	if mt.lua != l {
		return nil, fmt.Errorf("cannot create userdata with metatable from different Lua instance")
	}

	l.object.RLock()
	defer l.object.RUnlock()

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	dynData := newDynamicData(associatedData, func() {
		fmt.Println("dynamic data is being dropped")
	})
	cDynData := dynData.ToC()

	return withBaseRef(mt.BaseRef, func(ptr C.struct_GoLuaValueV2) (*LuaUserData, error) {
		res := C.luago_create_userdata(lua, cDynData, ptr)
		if res.error != nil {
			err := moveErrorToGo(res.error)
			return nil, err
		}
		ud, ok := castValue(l, res.value).(*LuaUserData)
		if !ok {
			return nil, fmt.Errorf("expected LuaUserData from userdata creation, got %T", ud)
		}
		return ud, nil
	})
}

func (l *Lua) Close() error {
	if l == nil || l.object == nil {
		return nil // Nothing to close
	}

	// Close the Lua VM object
	return l.object.Close()
}

type StdLib uint32

const (
	StdLibCoroutine StdLib = 1 << 0
	StdLibTable     StdLib = 1 << 1
	StdLibOS        StdLib = 1 << 2
	StdLibString    StdLib = 1 << 3
	StdLibUtf8      StdLib = 1 << 4
	StdLibBit       StdLib = 1 << 5
	StdLibMath      StdLib = 1 << 6
	StdLibBuffer    StdLib = 1 << 7
	StdLibVector    StdLib = 1 << 8
	StdLibDebug     StdLib = 1 << 9
	StdLibAll       StdLib = 1 << 31 // All standard libraries
)

// CreateLuaVm creates a new Lua VM with the entire standard library enabled
// and some default compiler options set.
func CreateLuaVm() (*Lua, error) {
	vm, err := CreateLuaVmComplex(StdLibAll)
	if err != nil {
		return nil, err
	}
	// Set default compiler options
	vm.SetCompilerOpts(CompilerOpts{
		OptimizationLevel: OptimizationLevelFull,
	})
	return vm, nil
}

// CreateLuaVmComplex creates a new Lua VM with the specified standard libraries enabled.
//
// If you want the entire stdlib to be exposed to scripts, pass `StdLibAll` here
// or use the CreateLuaVm function.
func CreateLuaVmComplex(stdlib StdLib) (*Lua, error) {
	ptr := C.newluavm(C.uint32_t(stdlib))
	if ptr == nil {
		return nil, fmt.Errorf("failed to create Lua VM")
	}
	vm := &Lua{object: newObject((*C.void)(unsafe.Pointer(ptr)), luaVmTab)}
	return vm, nil
}

// A special 'borrowed' Lua VM that is passed to callbacks.
//
// Provides special context-specific data about the current Lua state
type CallbackLua struct {
	mainstate *Lua // The main Lua VM that owns this callback
	cbstate   *Lua // The callback Lua
}

// Returns the main Lua VM state
//
// Note: it is not possible to get the callback Lua state directly to avoid
// object lifetime related issues.
//
// Returns nil if the CallbackLua is closed (note that CallbackLua is closed automatically when the callback function returns).
func (c *CallbackLua) MainState() *Lua {
	if c == nil {
		return nil // No main state if the callback Lua is nil
	}
	return c.mainstate
}

// Returns the currently running thread of the Lua VM.
//
// Returns nil if the CallbackLua is closed (note that CallbackLua is closed automatically when the callback function returns).
func (c *CallbackLua) CurrentThread() *LuaThread {
	if c.mainstate == nil || c.cbstate == nil {
		return nil // No current thread if the main state or callback state is nil
	}

	c.cbstate.object.RLock()
	defer c.cbstate.object.RUnlock()
	c.mainstate.object.RLock()
	defer c.mainstate.object.RUnlock()

	lua, err := c.cbstate.lua()
	if err != nil {
		return nil // Return nil if the Lua VM is closed
	}

	thread := C.luago_current_thread(lua)
	th, ok := castValue(c.mainstate, thread).(*LuaThread)
	if !ok {
		return nil
	}
	return th
}

// Sets the arguments to yield the thread with.
//
// Notes:
// - the yield will only occur after return.
// - the arguments returned will be ignored internally (as such, you should just return a empty value list after calling this method).
func (c *CallbackLua) YieldWith(args []any) error {
	if c == nil || c.cbstate == nil {
		return fmt.Errorf("callback Lua VM is closed")
	}

	c.cbstate.object.RLock()
	defer c.cbstate.object.RUnlock()

	lua, err := c.cbstate.lua()
	if err != nil {
		return err // Return error if the callback Lua VM is closed
	}

	outmw, err := createMultiValue(args)
	if err != nil {
		return err // Return error if the values cannot be converted (diff lua state, closed object, etc)
	}

	res := C.luago_yield_with(lua, outmw)
	if res.error != nil {
		return moveErrorToGo(res.error) // Return error if the yield failed
	}
	return nil
}

// Closes the CallbackLua object.
//
// Note: this is automatically called when the callback function returns,
func (c *CallbackLua) Close() error {
	if c == nil || c.cbstate == nil {
		return nil // Nothing to close
	}
	// Close the callback Lua VM object
	err := c.cbstate.Close()
	if err != nil {
		return err // Return error if the callback Lua VM is closed
	}
	// Nil out the mainstate to allow GC
	c.mainstate = nil
	return nil
}
