package vm

/*
#include "../rustlib/rustlib.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"unsafe"
)

// A handle to the Lua VM.
type Lua struct {
	// Pointer to the underlying Rust Lua VM
	p C.struct_Handle
	// Context handle (for example, callback lua state, interrupt lua state etc.)
	ctxp   C.struct_Handle
	closed atomic.Bool
}

func newLua(p C.struct_Handle, ctxp C.struct_Handle) *Lua {
	obj := &Lua{p: p, ctxp: ctxp}
	runtime.SetFinalizer(obj, (*Lua).Close)
	return obj
}

// Returns the string representation of the Lua VM.
func (l *Lua) String() string {
	if l == nil {
		return "<nil Lua VM>"
	}
	pt := l.CurrentThread().Pointer()
	if pt == 0 {
		return "<closed Lua VM>"
	}
	return fmt.Sprintf("Lua VM: 0x%x", pt)
}

func (l *Lua) ptr() C.struct_Handle {
	if l.closed.Load() {
		panic("attempted to access closed Lua VM")
	}
	return l.p
}

func (l *Lua) ctxlua() (C.struct_Handle, error) {
	if l.closed.Load() {
		return C.struct_Handle{}, errors.New("Lua VM is closed")
	}
	return l.ctxp, nil
}

func (l *Lua) lua() (C.struct_Handle, error) {
	if l.closed.Load() {
		return C.struct_Handle{}, errors.New("Lua VM is closed")
	}
	return l.p, nil
}

// Returns if two VM handles point to the same underlying base VM handle
func (l *Lua) IsSameVm(other *Lua) bool {
	if l.closed.Load() || other.closed.Load() {
		return false
	}
	return l.p == other.p
}

// SetCompilerOpts sets the default compiler options for the Lua VM.
func (l *Lua) SetCompilerOpts(opts CompilerOpts) {

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

	lua, err := l.lua()
	if err != nil {
		return 0 // Return 0 if the Lua VM is closed
	}

	used := C.luago_used_memory(lua)
	return int(used)
}

// MemoryLimit returns the memory limit set for the Lua VM.
func (l *Lua) MemoryLimit() int {

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

	lua, err := l.lua()
	if err != nil {
		return err
	}

	if mt != nil {
		if !mt.lua.IsSameVm(l) {
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

	lua, err := l.lua()
	if err != nil {
		return err
	}

	valueVal, err, _ := valueToC(l, value)

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
	if !tab.lua.IsSameVm(l) {
		return fmt.Errorf("cannot set globals from different Lua instance")
	}

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

type InterruptFn func(funcVm *Lua) (VmState, error)

// Sets an interrupt function that will periodically be called by Luau VM.
//
// Any Luau code is guaranteed to call this handler “eventually” (in practice this can happen at any function call or at any loop iteration).
//
// The provided interrupt function can error, and this error will be propagated through the Luau code that was executing at the time the interrupt was triggered.
//
// Also this can be used to implement continuous execution limits by instructing Luau VM to yield by returning VmState::Yield.
func (l *Lua) SetInterrupt(callback InterruptFn) {

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

		callbackVm := newLua(l.ptr(), cval.lua)

		vmState, err := callback(callbackVm)

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

	lua, err := l.lua()
	if err != nil {
		return // No-op if the Lua VM is closed
	}

	C.luago_remove_interrupt(lua)
}

// CreateString creates a Lua string from a Go string.
func (l *Lua) CreateString(s string) (*LuaString, error) {
	return l.createString([]byte(s))
}

// CreateString creates a Lua string from a Go string.
func (l *Lua) MustCreateString(s string) *LuaString {
	st, err := l.CreateString(s)

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

type FunctionFn func(funcVm *Lua, args []any) ([]any, error)

// CreateFunction creates a new Function
//
// # Note that funcVm will will remain open until all refs to it are closed.
func (l *Lua) CreateFunction(callback FunctionFn) (*LuaFunction, error) {

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

		callbackVm := newLua(l.ptr(), cval.lua)
		values, err := callback(callbackVm, mw)

		if err != nil {
			errv := moveStringToRust(err.Error())
			cval.error = errv // Rust side will deallocate it for us
			return
		}

		outmw, err := createMultiValue(l, values)
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

	if !fn.lua.IsSameVm(l) {
		return nil, fmt.Errorf("cannot create thread from different Lua instance")
	}

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

	lua, err := l.lua()
	if err != nil {
		return nil, err
	}

	var env = nullCValueV2()
	if opts.Env != nil {
		if !opts.Env.lua.IsSameVm(l) {
			return nil, fmt.Errorf("cannot set environment table from different Lua instance")
		}
		env, err, _ = valueToC(l, opts.Env)
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
	if !mt.lua.IsSameVm(l) {
		return nil, fmt.Errorf("cannot create userdata with metatable from different Lua instance")
	}

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

// Close closes the Lua VM.
func (l *Lua) Close() error {
	if l == nil || l.closed.Load() {
		return nil // Nothing to close
	}

	// Close the Lua VM object
	runtime.SetFinalizer(l, nil) // Remove finalizer to avoid double free
	l.closed.Store(true)
	C.freeluavm(l.ctxp) // for main state, p == ctxp, otherwise closes the context only
	return nil
}

// IsClosed returns whether the Lua VM is closed.
func (l *Lua) IsClosed() bool {
	if l == nil {
		return true
	}
	return l.closed.Load()
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
	res := C.newluavm(C.uint32_t(stdlib))
	if res.error != nil {
		return nil, moveErrorToGo(res.error)
	}
	ptr := res.value
	vm := newLua(ptr, ptr)
	return vm, nil
}

// Returns the currently running thread of the Lua VM.
//
// Returns nil if the Lua VM handle is closed.
func (l *Lua) CurrentThread() *LuaThread {
	lua, err := l.ctxlua()
	if err != nil {
		return nil // Return nil if the Lua VM is closed
	}

	thread := C.luago_current_thread(lua)
	th, ok := castValue(l, thread).(*LuaThread)
	if !ok {
		return nil
	}
	return th
}

// Sets the arguments to yield the current thread with.
//
// Notes:
// - the yield will only occur after return.
// - the arguments returned will be ignored internally (as such, you should just return a empty value list after calling this method).
func (l *Lua) YieldWith(args []any) error {
	lua, err := l.ctxlua()
	if err != nil {
		return nil // Return nil if the Lua VM is closed
	}

	outmw, err := createMultiValue(l, args)
	if err != nil {
		return err // Return error if the values cannot be converted (diff lua state, closed object, etc)
	}

	res := C.luago_yield_with(lua, outmw)
	if res.error != nil {
		return moveErrorToGo(res.error) // Return error if the yield failed
	}
	return nil
}
